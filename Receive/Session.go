package Receive

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

type transcriberOpenResult struct {

	transcriber *Transcriber
	err error

}

const (

	stateListening int32 = iota
	stateCapturing // 1
	stateBlacklisted // 2

)

const (

	// Capture bounds

	commandCaptureMin = 2 * time.Second
	commandCaptureMax = 8 * time.Second

	commandTailAfterPacket  = 2 * time.Second

	minCommandPCMBytes = TargetSampleRate * 2 * 3 / 2 // 1.5s @ 16kHz mono

	prerollMaxFrames = 100 // ~2s @ 20ms/frame

	playArgsPacketSilence = 2 * time.Second
	minPlayArgsCapture = 2 * time.Second

	postCaptureCooldown = 150 * time.Millisecond

	hotMicBlacklistAfter = 30 * time.Second // triage users with sustained audio input
	hotMicBlacklistFor = 15 * time.Second // constant re-eval after 15s
	hotMicEnergyThreshold float32 = 0.0006 // abt -40dBFS

	captureMaxDuration = 10 * time.Second

)

type Session struct {

	GuildID snowflake.ID
	UserID  snowflake.ID

	dispatcher *Dispatcher

	decoder *OpusPCMDecoder
	wake *WakeStream

	inbox chan []byte
	inboxDrops atomic.Uint64
	sttUpdates chan TranscriptUpdate
	opusPreroll opusPreroll

	state atomic.Int32
	lastPacketAt atomic.Int64
	speechStartAt atomic.Int64
	blacklistUntil atomic.Int64

	transcriber *Transcriber
	transcriberReady chan transcriberOpenResult

	captureBuf [][]byte
	capturePCM []byte

	captureFramesSent int
	captureSpeechFrames int
	captureStartedAt time.Time

	cooldownUntil time.Time

	finalizing atomic.Bool
	dispatched atomic.Bool

	awaitingPlayArgs bool // for play specific; TODO: generalize for "command tail" detection?

	probePCM []byte
	lastWakeProbeAt  time.Time

	wakeProbeBusy atomic.Bool
	wakeProbeHit chan string

	cancel context.CancelFunc
	closed atomic.Bool

	wg sync.WaitGroup

}

func NewSession(GuildID, UserID snowflake.ID, Dispatcher *Dispatcher) (*Session, error) {

	Decoder, ErrDecoder := NewOpusPCMDecoder()

	if ErrDecoder != nil {

		return nil, fmt.Errorf("opus decoder: %w", ErrDecoder)

	}

	Ctx, Cancel := context.WithCancel(context.Background())

	S := &Session{

		GuildID: GuildID,
		UserID: UserID,

		dispatcher: Dispatcher,
		decoder: Decoder,

		wake: NewWakeStream(),
		inbox: make(chan []byte, 512),

		opusPreroll: newOpusPreroll(prerollMaxFrames),

		sttUpdates: make(chan TranscriptUpdate, 32),

		transcriberReady: make(chan transcriberOpenResult, 1),
		wakeProbeHit: make(chan string, 1),

		cancel: Cancel,

	}

	S.state.Store(stateListening)

	S.wg.Add(1)
	go S.run(Ctx)

	S.wg.Add(1)
	go S.timeoutLoop(Ctx)

	return S, nil

}

func (S *Session) Push(Opus []byte) {

	if S.closed.Load() || S.wake == nil {

		return

	}

	if Until := S.blacklistUntil.Load(); Until > 0 && time.Now().UnixNano() < Until {

		return

	}

	S.lastPacketAt.Store(time.Now().UnixNano())

	if S.speechStartAt.Load() == 0 {

		S.speechStartAt.Store(time.Now().UnixNano())

	}

	Frame := make([]byte, len(Opus))
	copy(Frame, Opus)

	select {

	case S.inbox <- Frame:

	default:

		S.inboxDrops.Add(1)

	}

}

func (S *Session) Close() {

	if !S.closed.CompareAndSwap(false, true) {

		return

	}

	if S.cancel != nil {

		S.cancel()

	}

	S.wg.Wait()

	if S.transcriber != nil {

		S.transcriber.Close()
		S.transcriber = nil

	}

	if S.wake != nil {

		S.wake.Close()
		S.wake = nil

	}

	if S.decoder != nil {

		S.decoder.Close()
		S.decoder = nil

	}

}

func (S *Session) run(Ctx context.Context) {

	defer S.wg.Done()

	defer func() {

		if r := recover(); r != nil {

			Utils.Logger.Error("Receive", fmt.Sprintf("Session worker panic (user %s): %v", S.UserID, r))

		}

	}()

	for {

		select {

		case <-Ctx.Done():

			return

		case Opus := <-S.inbox:

			S.handleFrame(Opus)

		case Res := <-S.transcriberReady:

			S.handleTranscriberReady(Res)

		case Upd := <-S.sttUpdates:

			S.handleTranscriptUpdate(Upd)

		case Text := <-S.wakeProbeHit:

			S.drainWakeProbeHit(Text)

		}

	}

}

func (S *Session) handleFrame(Opus []byte) {

	switch S.state.Load() {

	case stateListening:

		S.opusPreroll.Push(Opus)

		PCM, ErrDecode := S.decoder.DecodeWake(Opus)

		if ErrDecode != nil {

			return

		}

		InCooldown := !S.cooldownUntil.IsZero() && time.Now().Before(S.cooldownUntil)

		if InCooldown {

			// Still probe via STT during post-command cooldown

			S.runSTTProbePath(PCM)
			return

		}

		S.runListening(PCM)

	case stateCapturing:

		PCM, ErrDecode := S.decoder.DecodeCapture(Opus)

		if ErrDecode != nil {

			return

		}

		S.runCapturing(PCM)

	}

}

func (S *Session) runListening(PCM []int16) {

	Energy := FrameRMS(PCM)

	if Energy < hotMicEnergyThreshold {

		S.speechStartAt.Store(0)

		if Hit := S.wake.FlushPending(); Hit != "" {

			S.beginCapture()
			return

		}

		S.flushSTTWakeProbeOnSilence()
		return

	}

	WakeSamples := Int16ToFloat32(AmplifyInt16ForKWS(PCM))

	if Hit := S.wake.Feed(WakeSamples); Hit != "" {

		S.beginCapture()
		return

	}

	if Hit := S.wake.FlushPartial(); Hit != "" {

		S.beginCapture()
		return

	}

	S.runSTTProbePath(PCM)

}

func (S *Session) runSTTProbePath(PCM []int16) {

	S.maybeSTTWakeProbe(PCM)

}

func (S *Session) beginCapture() {

	if !S.state.CompareAndSwap(stateListening, stateCapturing) {

		return

	}

	S.wake.Reset()
	S.probePCM = nil

	S.captureStartedAt = time.Now()
	S.captureFramesSent = 0
	S.captureSpeechFrames = 0
	S.capturePCM = nil
	S.captureBuf = nil

	S.decoder.ResetCapturePath()

	for _, OpusFrame := range S.opusPreroll.Drain() {

		PCMFrame, ErrPre := S.decoder.DecodeCapture(OpusFrame)

		if ErrPre != nil {

			continue

		}

		Chunk := Int16ToBytesLE(PCMFrame)
		S.capturePCM = append(S.capturePCM, Chunk...)
		S.captureBuf = append(S.captureBuf, Chunk)

		if FrameHasSpeech(PCMFrame) {

			S.captureSpeechFrames++

		}

	}

	if Drops := S.inboxDrops.Swap(0); Drops > 0 {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Dropped %d opus frame(s) before capture (user %s)", Drops, S.UserID, ))

	}

	S.finalizing.Store(false)
	S.dispatched.Store(false)
	S.awaitingPlayArgs = false

	go func() {

		Trans, ErrTrans := NewTranscriber(context.Background())

		S.transcriberReady <- transcriberOpenResult{transcriber: Trans, err: ErrTrans}

	}()

}

func (S *Session) handleTranscriberReady(Res transcriberOpenResult) {

	if S.state.Load() != stateCapturing {

		if Res.transcriber != nil {

			Res.transcriber.Close()

		}

		return

	}

	if Res.err != nil {

		Utils.Logger.Error("Receive", fmt.Sprintf("Transcriber open failed: %s", Res.err.Error()))
		S.abortCapture()

		return

	}

	S.transcriber = Res.transcriber

	S.transcriber.SetOnUpdate(func(Upd TranscriptUpdate) {

		select {

		case S.sttUpdates <- Upd:

		default:

			Utils.Logger.Warn("Receive", "STT update dropped (worker busy)")

		}

	})

	for _, PCM := range S.captureBuf {

		if ErrSend := S.transcriber.Send(PCM); ErrSend != nil {

			Utils.Logger.Warn("Receive", fmt.Sprintf("Transcriber buffered send failed: %s", ErrSend.Error()))

		} else {

			S.captureFramesSent++

		}

	}

	S.captureBuf = nil

}

func (S *Session) runCapturing(PCM []int16) {

	PCMBytes := Int16ToBytesLE(PCM)
	S.capturePCM = append(S.capturePCM, PCMBytes...)

	if FrameHasSpeech(PCM) {

		S.captureSpeechFrames++

	}

	if S.transcriber == nil {

		const maxBufferedFrames = 300

		S.captureBuf = append(S.captureBuf, PCMBytes)

		if len(S.captureBuf) > maxBufferedFrames {

			S.captureBuf = S.captureBuf[len(S.captureBuf)-maxBufferedFrames:]

		}

		return

	}

	if time.Since(S.captureStartedAt) > captureMaxDuration {

		S.finalizeCapture()
		return

	}

	if ErrSend := S.transcriber.Send(PCMBytes); ErrSend != nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Transcriber send failed: %s", ErrSend.Error()))
		S.finalizeCapture()

	} else {

		S.captureFramesSent++

	}

}

// handleTranscriptUpdate runs JIT command detection on streaming partials.
func (S *Session) handleTranscriptUpdate(Upd TranscriptUpdate) {

	if S.state.Load() != stateCapturing || S.dispatched.Load() {

		return

	}

	if Upd.Text == "" {

		if Upd.SpeechFinal && S.awaitingPlayArgs {

			S.finalizeCapture()
		}

		return

	}

	Cmd, OK := Parse(Upd.Text)

	if !OK {

		if Upd.SpeechFinal && S.awaitingPlayArgs {

			S.finalizeCapture()

		}

		return

	}

	if Cmd.Command == CommandPause || Cmd.Command == CommandResume {

		Cmd.Args = ""

	}

	switch Cmd.Command {

	case CommandPause, CommandResume:

		S.dispatchCommand(Cmd)
		S.abortCapture()

		return

	case CommandPlay:

		if Cmd.Args != "" {

			S.dispatchCommand(Cmd)
			S.abortCapture()

			return

		}

		S.awaitingPlayArgs = true

	}

	// xAI utterance-end: flush STT for play-args tail or if JIT did not fire.
	if Upd.SpeechFinal && !S.dispatched.Load() {

		if S.awaitingPlayArgs {

			S.finalizeCapture()

		}

	}

}

func (S *Session) dispatchCommand(Cmd ParsedCommand) {

	if S.dispatched.Swap(true) {

		return

	}

	if S.dispatcher != nil {

		S.dispatcher.Dispatch(S.GuildID, S.UserID, Cmd)

	}

}

func (S *Session) ensureTranscriber(Timeout time.Duration) *Transcriber {

	if S.transcriber != nil {

		return S.transcriber

	}

	select {

	case Res := <-S.transcriberReady:

		S.handleTranscriberReady(Res)

	case <-time.After(Timeout):

	}

	return S.transcriber

}

func (S *Session) abortCapture() {

	if !S.finalizing.CompareAndSwap(false, true) {

		return

	}

	Trans := S.transcriber
	S.transcriber = nil

	go func(T *Transcriber) {

		if T != nil {

			T.Finalize()

		}

	}(Trans)

	S.endCapture()

}

func (S *Session) finalizeCapture() {

	if !S.finalizing.CompareAndSwap(false, true) {

		return

	}

	Trans := S.ensureTranscriber(transcribeReadyTimeout + time.Second)

	if Trans == nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Capture ended with no transcriber (user %s, %d buffered frame(s))", S.UserID, len(S.captureBuf)))
		S.endCapture()

		return

	}

	S.transcriber = nil

	PCMCopy := PreparePCMForSTT(append([]byte(nil), S.capturePCM...))
	AlreadyDispatched := S.dispatched.Load()

	go func(T *Transcriber, PCM []byte, SkipDispatch bool) {

		defer func() {

			if r := recover(); r != nil {

				Utils.Logger.Error("Receive", fmt.Sprintf("finalizeCapture panic: %v", r))

			}

		}()

		T.Finalize()
		Text := T.Result()

		if SkipDispatch {

			return

		}

		if Text == "" && len(PCM) > 0 {

			if RESTText, ErrREST := TranscribeREST(context.Background(), PCM, TargetSampleRate); ErrREST == nil {

				Text = RESTText

			}

		}

		if Text == "" {

			return

		}

		Cmd, OK := Parse(Text)

		if !OK {

			return

		}

		if S.dispatcher != nil {

			S.dispatcher.Dispatch(S.GuildID, S.UserID, Cmd)

		}

	}(Trans, PCMCopy, AlreadyDispatched)

	S.endCapture()

}

func (S *Session) endCapture() {

	S.state.Store(stateListening)
	S.captureStartedAt = time.Time{}

	S.captureBuf = nil
	S.capturePCM = nil

	S.captureFramesSent = 0
	S.captureSpeechFrames = 0

	S.finalizing.Store(false)
	S.speechStartAt.Store(0)

	S.awaitingPlayArgs = false

	S.cooldownUntil = time.Now().Add(postCaptureCooldown)

	if S.transcriber != nil {

		S.transcriber.Close()
		S.transcriber = nil

	}

	S.opusPreroll.Clear()
	S.probePCM = nil

	if S.decoder != nil {

		S.decoder.ResetWakePath()
		S.decoder.ResetCapturePath()

	}

	if S.wake != nil {

		S.wake.Reset()

	}

}

func (S *Session) timeoutLoop(Ctx context.Context) {

	defer S.wg.Done()

	Ticker := time.NewTicker(100 * time.Millisecond)
	defer Ticker.Stop()

	for {

		select {

		case <-Ctx.Done():

			return

		case <-Ticker.C:

			S.checkTimeouts()

		}

	}

}

func (S *Session) checkTimeouts() {

	if S.state.Load() != stateCapturing {

		S.checkHotMic()
		return

	}

	Now := time.Now()
	CaptureAge := Now.Sub(S.captureStartedAt)

	if CaptureAge > captureMaxDuration {

		S.finalizeCapture()
		return

	}

	if S.dispatched.Load() {

		return

	}

	if S.awaitingPlayArgs {

		if CaptureAge >= minPlayArgsCapture {

			SincePacket := S.silenceSincePacket()

			if SincePacket > playArgsPacketSilence {

				S.finalizeCapture()

			}

		}

		return

	}

	SincePacket := S.silenceSincePacket()

	if CaptureAge < commandCaptureMin {

		return

	}

	if CaptureAge >= commandCaptureMax {

		S.finalizeCapture()
		return

	}

	if len(S.capturePCM) >= minCommandPCMBytes && SincePacket >= commandTailAfterPacket {

		S.finalizeCapture()

	}

	S.checkHotMic()

}

func (S *Session) silenceSincePacket() time.Duration {

	At := S.lastPacketAt.Load()

	if At == 0 {

		return 0

	}

	return time.Duration(time.Now().UnixNano() - At)

}

func (S *Session) checkHotMic() {

	Now := time.Now().UnixNano()
	SpeechStart := S.speechStartAt.Load()

	if SpeechStart > 0 && time.Duration(Now-SpeechStart) > hotMicBlacklistAfter {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Hot mic detected (user %s); blacklisting briefly", S.UserID))

		S.blacklistUntil.Store(time.Now().Add(hotMicBlacklistFor).UnixNano())
		S.speechStartAt.Store(0)

		if S.state.Load() == stateCapturing {

			S.finalizeCapture()

		}

	}

}
