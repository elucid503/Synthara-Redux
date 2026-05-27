package Receive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	commandCaptureMin = 2 * time.Second
	commandCaptureMax = 8 * time.Second
	commandTailAfterSilence = 2 * time.Second

	minCommandPCMBytes = TargetSampleRate * 2 * 3 / 2
	prerollMaxFrames   = 50

	tailSilenceTimeout = 1 * time.Second
	postCaptureCooldown = 150 * time.Millisecond

	discordSpeakingMax = 10 * time.Second

	sttPreconnectMaxBytes = TargetSampleRate * 2 * 3

)

type Session struct {

	GuildID snowflake.ID
	UserID snowflake.ID

	dispatcher *Dispatcher
	decoder *OpusDecoder

	inbox chan []byte
	inboxDrops atomic.Uint64

	wakeCh chan struct{}

	sttUpdates chan TranscriptUpdate
	opusPreroll opusPreroll

	state atomic.Int32

	discordSpeaking atomic.Bool
	speakingSince atomic.Int64
	lastSilentAt atomic.Int64

	transcriber *Transcriber
	transcriberReady chan transcriberOpenResult
	captureID atomic.Uint64

	preCapture *PCMBuffer
	capturePCM []byte

	captureStartedAt time.Time
	cooldownUntil time.Time

	finalizing atomic.Bool
	dispatched atomic.Bool

	awaitingCommandTail bool

	lastTranscriptChange atomic.Int64 // UnixNano; 0 = no change yet this capture
	lastTranscriptText   string       // run-goroutine only

	ctx context.Context
	cancel context.CancelFunc
	closed atomic.Bool

	wg sync.WaitGroup

}

func NewSession(GuildID, UserID snowflake.ID, Dispatcher *Dispatcher) (*Session, error) {

	Decoder, ErrDecoder := NewOpusDecoder()

	if ErrDecoder != nil {

		return nil, fmt.Errorf("opus decoder: %w", ErrDecoder)

	}

	Ctx, Cancel := context.WithCancel(context.Background())

	S := &Session{

		GuildID: GuildID,
		UserID: UserID,

		dispatcher: Dispatcher,
		decoder: Decoder,

		inbox: make(chan []byte, 256),
		wakeCh: make(chan struct{}, 1),

		opusPreroll: newOpusPreroll(prerollMaxFrames),
		sttUpdates: make(chan TranscriptUpdate, 32),
		transcriberReady: make(chan transcriberOpenResult, 1),

		preCapture: NewPCMBuffer(sttPreconnectMaxBytes),

		ctx: Ctx,
		cancel: Cancel,

	}

	S.state.Store(stateListening)

	if ErrOpen := PicoOpenStream(GuildID, UserID); ErrOpen != nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Pico open stream: %s", ErrOpen.Error()))

	}

	S.wg.Add(1)
	go S.run(Ctx)

	S.wg.Add(1)
	go S.timeoutLoop(Ctx)

	return S, nil

}

func (S *Session) Push(Opus []byte) {

	if S.closed.Load() {

		return

	}

	Frame := make([]byte, len(Opus))
	copy(Frame, Opus)

	select {

	case S.inbox <- Frame:

	default:

		S.inboxDrops.Add(1)

	}

}

func (S *Session) NotifyWake() {

	if S.closed.Load() {

		return

	}

	select {

	case S.wakeCh <- struct{}{}:

	default:

	}

}

func (S *Session) SetDiscordSpeaking(Active bool) {

	Was := S.discordSpeaking.Load()
	S.discordSpeaking.Store(Active)

	Now := time.Now().UnixNano()

	if Active && !Was {

		S.speakingSince.Store(Now)

	}

	if !Active {

		S.speakingSince.Store(0)
		S.lastSilentAt.Store(Now)

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

	_ = PicoCloseStream(S.GuildID, S.UserID)

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

		case <-S.wakeCh:

			S.beginCapture()

		case Res := <-S.transcriberReady:

			S.handleTranscriberReady(Res)

		case Upd := <-S.sttUpdates:

			S.handleTranscriptUpdate(Upd)

		}

	}

}

func (S *Session) handleFrame(Opus []byte) {

	PCM, ErrDecode := S.decoder.Decode(Opus)

	if ErrDecode != nil {

		return

	}

	switch S.state.Load() {

	case stateListening:

		S.opusPreroll.Push(Opus)

		if !S.cooldownUntil.IsZero() && time.Now().Before(S.cooldownUntil) {

			return

		}

		if ErrFeed := PicoFeedPCM(S.GuildID, S.UserID, Int16ToBytesLE(PCM)); ErrFeed != nil {

			Utils.Logger.Warn("Receive", fmt.Sprintf("Pico feed: %s", ErrFeed.Error()))

		}

	case stateCapturing:

		S.runCapturing(PCM)

	}

}

func (S *Session) beginCapture() {

	if !S.state.CompareAndSwap(stateListening, stateCapturing) {

		return

	}

	captureID := S.captureID.Add(1)

	S.captureStartedAt = time.Now()
	S.resetCaptureBuffers()
	S.decoder.Reset()

	for _, OpusFrame := range S.opusPreroll.Drain() {

		PCMFrame, ErrPre := S.decoder.Decode(OpusFrame)

		if ErrPre != nil {

			continue

		}

		Chunk := Int16ToBytesLE(PCMFrame)
		S.capturePCM = append(S.capturePCM, Chunk...)
		S.preCapture.Append(Chunk)

	}

	if Drops := S.inboxDrops.Swap(0); Drops > 0 {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Dropped %d opus frame(s) before capture (user %s)", Drops, S.UserID))

	}

	S.finalizing.Store(false)
	S.dispatched.Store(false)
	S.awaitingCommandTail = false

	emitFeedbackCue(S.GuildID, FeedbackCueCaptureStart)
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

	for _, PCM := range S.preCapture.DrainChunks(xaiPCMChunkBytes) {

		_ = S.transcriber.Send(PCM)

	}

	if Tail := S.preCapture.Remainder(); len(Tail) > 0 {

		_ = S.transcriber.Send(Tail)

	}

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

	if S.speakingDuration() > discordSpeakingMax {

		S.finalizeCapture()
		return

	}

	PCMBytes := Int16ToBytesLE(PCM)
	S.capturePCM = append(S.capturePCM, PCMBytes...)

	if S.transcriber == nil {

		S.preCapture.Append(PCMBytes)
		return

	}

	if time.Since(S.captureStartedAt) > commandCaptureMax+2*time.Second {

		S.finalizeCapture()
		return

	}

	if ErrSend := S.transcriber.Send(PCMBytes); ErrSend != nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Transcriber send failed: %s", ErrSend.Error()))
		S.finalizeCapture()

	}

}

func (S *Session) speakingDuration() time.Duration {

	At := S.speakingSince.Load()

	if At == 0 {

		return 0

	}

	return time.Duration(time.Now().UnixNano() - At)

}

func (S *Session) handleTranscriptUpdate(Upd TranscriptUpdate) {

	if S.state.Load() != stateCapturing || S.dispatched.Load() {

		return

	}

	if Upd.Text != "" && Upd.Text != S.lastTranscriptText {

		S.lastTranscriptText = Upd.Text
		S.lastTranscriptChange.Store(time.Now().UnixNano())

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

	if S.transcriber != nil && !S.transcriber.Done() {

		return S.transcriber

	}

	S.transcriber = nil

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

			T.Close()

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

		Utils.Logger.Warn("Receive", fmt.Sprintf("Capture ended with no transcriber (user %s)", S.UserID))
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

		if SkipDispatch {

			T.Close()
			return

		}

		T.Finalize()
		Text := T.Result()

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
	S.awaitingCommandTail = false
	S.cooldownUntil = time.Now().Add(postCaptureCooldown)

	if S.transcriber != nil {

		S.transcriber.Close()
		S.transcriber = nil

	}

	S.opusPreroll.Clear()

	if S.decoder != nil {

		S.decoder.Reset()

	}

	emitFeedbackCue(S.GuildID, FeedbackCueCaptureEnd)
	emitCaptureDuck(S.GuildID, false)

}

func (S *Session) resetCaptureBuffers() {

	S.preCapture.Reset()
	S.capturePCM = nil
	S.lastTranscriptChange.Store(0)
	S.lastTranscriptText = ""

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

		return

	}

	Now := time.Now()
	CaptureAge := Now.Sub(S.captureStartedAt)

	if S.speakingDuration() > discordSpeakingMax {

		S.finalizeCapture()
		return

	}

	if CaptureAge > commandCaptureMax+2*time.Second {

		S.finalizeCapture()
		return

	}

	if S.dispatched.Load() {

		return

	}

	if S.awaitingCommandTail {

		if CaptureAge >= commandCaptureMax {

			S.finalizeCapture()
			return

		}

		if At := S.lastTranscriptChange.Load(); At != 0 {

			if time.Duration(time.Now().UnixNano()-At) >= tailSilenceTimeout {

				S.finalizeCapture()

			}

		}

		return

	}

	if CaptureAge < commandCaptureMin {

		return

	}

	if CaptureAge >= commandCaptureMax {

		S.finalizeCapture()
		return

	}

	if len(S.capturePCM) >= minCommandPCMBytes && S.silenceSinceDiscordStopped() >= commandTailAfterSilence {

		S.finalizeCapture()

	}

}

func (S *Session) silenceSinceDiscordStopped() time.Duration {

	At := S.lastSilentAt.Load()

	if At == 0 {

		return 0

	}

	return time.Duration(time.Now().UnixNano() - At)

}

func parseStreamID(StreamID string) (snowflake.ID, snowflake.ID, bool) {

	Parts := strings.SplitN(StreamID, ":", 2)

	if len(Parts) != 2 {

		return 0, 0, false

	}

	GuildU, ErrG := strconv.ParseUint(Parts[0], 10, 64)

	if ErrG != nil {

		return 0, 0, false

	}

	UserU, ErrU := strconv.ParseUint(Parts[1], 10, 64)

	if ErrU != nil {

		return 0, 0, false

	}

	return snowflake.ID(GuildU), snowflake.ID(UserU), true

}
