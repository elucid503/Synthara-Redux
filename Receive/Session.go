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

	captureID uint64
	transcriber *Transcriber

	err error

}

const (

	stateListening int32 = iota
	stateCapturing // 1

)

const (

	// Capture bounds

	commandCaptureMin = 2 * time.Second
	commandCaptureMax = 8 * time.Second

	commandTailAfterPacket = 2 * time.Second

	minCommandPCMBytes = TargetSampleRate * 2 * 3 / 2 // 1.5s @ 16kHz mono

	prerollMaxFrames = 50 // ~1s @ 20ms/frame — enough to cover the wake word + KWS detection latency

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
	captureID atomic.Uint64

	captureBuf [][]byte
	capturePCM []byte

	captureStartedAt time.Time

	cooldownUntil time.Time

	finalizing atomic.Bool
	dispatched atomic.Bool

	awaitingCommandTail bool // set for commands that take a multi-word arg; waits for silence before finalizing

	ctx context.Context
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
		UserID:  UserID,

		dispatcher: Dispatcher,
		decoder:    Decoder,

		wake:  NewWakeStream(),
		inbox: make(chan []byte, 512),

		opusPreroll: newOpusPreroll(prerollMaxFrames),

		sttUpdates: make(chan TranscriptUpdate, 32),

		transcriberReady: make(chan transcriberOpenResult, 1),

		ctx:    Ctx,
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

		// Acoustic silence marks a speech-segment boundary. Discard the preroll
		// so audio from a prior utterance cannot bleed into the next capture.
		// (DTX-based clearing via checkPrerollExpiry requires a 2-second packet
		// gap, which is too slow when the user speaks again shortly after.)
		S.opusPreroll.Clear()

		S.speechStartAt.Store(0)

		if Hit := S.wake.FlushPending(); Hit != "" {

			S.beginCapture()
			return

		}

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

}

func (S *Session) beginCapture() {

	if !S.state.CompareAndSwap(stateListening, stateCapturing) {

		return

	}

	S.wake.Reset()

	captureID := S.captureID.Add(1)
	S.captureStartedAt = time.Now()
	S.resetCaptureBuffers()

	S.decoder.ResetCapturePath()

	for _, OpusFrame := range S.opusPreroll.Drain() {

		PCMFrame, ErrPre := S.decoder.DecodeCapture(OpusFrame)

		if ErrPre != nil {

			continue

		}

		Chunk := Int16ToBytesLE(PCMFrame)
		S.capturePCM = append(S.capturePCM, Chunk...)
		S.captureBuf = append(S.captureBuf, Chunk)

	}

	if Drops := S.inboxDrops.Swap(0); Drops > 0 {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Dropped %d opus frame(s) before capture (user %s)", Drops, S.UserID))

	}

	S.finalizing.Store(false)
	S.dispatched.Store(false)
	S.awaitingCommandTail = false

	emitVoiceCue(S.GuildID, VoiceCueWake)
	emitCaptureDuck(S.GuildID, true)

	go S.openTranscriber(captureID)

}

func (S *Session) handleTranscriberReady(Res transcriberOpenResult) {

	if Res.err != nil {

		Utils.Logger.Error("Receive", fmt.Sprintf("Transcriber open failed: %s", Res.err.Error()))

		if S.state.Load() == stateCapturing && Res.captureID == S.captureID.Load() {

			S.abortCapture()

		}

		return

	}

	if S.state.Load() != stateCapturing || Res.captureID != S.captureID.Load() {

		if Res.transcriber != nil {

			Res.transcriber.Close()

		}

		return

	}

	S.attachTranscriber(Res.transcriber)

}

func (S *Session) attachTranscriber(Trans *Transcriber) {

	if Trans == nil || Trans.Done() {

		S.transcriber = nil
		return

	}

	S.transcriber = Trans

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

		}

	}

	S.captureBuf = nil

}

func (S *Session) openTranscriber(captureID uint64) {

	Trans, ErrTrans := NewTranscriber(S.ctx)

	Res := transcriberOpenResult{captureID: captureID, transcriber: Trans, err: ErrTrans}

	select {

		case S.transcriberReady <- Res:

		case <-S.ctx.Done():

			if Trans != nil {

				Trans.Close()

			}

	}

}

func (S *Session) runCapturing(PCM []int16) {

	PCMBytes := Int16ToBytesLE(PCM)
	S.capturePCM = append(S.capturePCM, PCMBytes...)

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

	}

}

// handleTranscriptUpdate runs JIT command detection on streaming partials.
func (S *Session) handleTranscriptUpdate(Upd TranscriptUpdate) {

	if S.state.Load() != stateCapturing || S.dispatched.Load() {

		return

	}

	if Upd.Text == "" {

		if Upd.SpeechFinal && S.awaitingCommandTail {

			S.finalizeCapture()
		}

		return

	}

	Cmd, OK := Parse(Upd.Text)

	if !OK {

		if Upd.SpeechFinal && S.awaitingCommandTail {

			S.finalizeCapture()

		}

		return

	}

	if CommandClearsTrailingArgs(Cmd.Command) {

		Cmd.Args = ""

	}

	// Commands that take a free-form multi-word argument must never dispatch on a partial result, since the user may still be speaking.

	if CommandNeedsMultiWordArgs(Cmd.Command) {

		S.awaitingCommandTail = true

		if Upd.SpeechFinal {

			S.finalizeCapture()

		}

		return

	}

	if CommandDispatchesImmediately(Cmd.Command, Cmd.Args) {

		S.dispatchCommand(Cmd)
		S.abortCapture()

		return

	}

	// xAI utterance-end: flush STT if JIT did not fire...

	if Upd.SpeechFinal && !S.dispatched.Load() {

		S.finalizeCapture()

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

		if !S.transcriber.Done() {

			return S.transcriber

		}

		S.transcriber = nil

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

	AlreadyDispatched := S.dispatched.Load()

	go func(T *Transcriber, SkipDispatch bool) {

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

	}(Trans, AlreadyDispatched)

	S.endCapture()

}

func (S *Session) endCapture() {

	S.state.Store(stateListening)
	S.captureStartedAt = time.Time{}

	S.resetCaptureBuffers()

	S.finalizing.Store(false)
	S.speechStartAt.Store(0)

	S.awaitingCommandTail = false

	S.cooldownUntil = time.Now().Add(postCaptureCooldown)

	if S.transcriber != nil {

		S.transcriber.Close()
		S.transcriber = nil

	}

	S.clearListeningAudio()

	if S.decoder != nil {

		S.decoder.ResetWakePath()
		S.decoder.ResetCapturePath()

	}

	if S.wake != nil {

		S.wake.Reset()

	}

	emitVoiceCue(S.GuildID, VoiceCueEnd)
	emitCaptureDuck(S.GuildID, false)

}

func (S *Session) resetCaptureBuffers() {

	S.captureBuf = nil
	S.capturePCM = nil

}

func (S *Session) clearListeningAudio() {

	S.opusPreroll.Clear()

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

const prerollExpireAfterSilence = 2 * time.Second // config for how long to keep frames after speech ends

// checkPrerollExpiry discards the preroll when a speech segment has ended
func (S *Session) checkPrerollExpiry() {

	SincePacket := S.silenceSincePacket()

	if SincePacket == 0 || SincePacket < prerollExpireAfterSilence {

		return

	}

	S.opusPreroll.Clear()

}

func (S *Session) checkTimeouts() {

	if S.state.Load() != stateCapturing {

		S.checkHotMic()
		S.checkPrerollExpiry()

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

	if S.awaitingCommandTail {

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
