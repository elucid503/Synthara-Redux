package Receive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"Synthara-Redux/Utils"

	"github.com/gorilla/websocket"
)

const (

	sttLanguage = "en" // English only

	sttLanguageEnv = "VOICE_STT_LANGUAGE"

	xaiPCMChunkBytes = 3200 // 100ms PCM16 mono @ 16kHz.

	sttDebugEnv = "VOICE_STT_DEBUG"

	transcribeHardTimeout = 12 * time.Second
	transcribeReadyTimeout = 5 * time.Second
	transcribeDoneWait = 5 * time.Second

	envSTTEndpointing  = "VOICE_STT_ENDPOINTING_MS"
	defaultEndpointing = 200

)

// TranscriptUpdate is delivered to the session on each STT partial.
type TranscriptUpdate struct {

	Text string

	IsFinal bool
	SpeechFinal bool

}

// Called from the transcriber read loop.
type OnTranscriptFunc func(TranscriptUpdate)

// Transcriber owns a single xAI STT WebSocket session.
type Transcriber struct {

	conn *websocket.Conn
	cancel context.CancelFunc

	ready chan struct{}
	done chan struct{}

	writeMu sync.Mutex
	pcmBuf []byte

	startOnce sync.Once
	timeoutOnce sync.Once
	closeOnce sync.Once
	doneOnce sync.Once

	textMu sync.Mutex
	text string
	interim string

	onUpdate   OnTranscriptFunc
	onUpdateMu sync.Mutex

}

type sttEnvelope struct {

	Type string `json:"type"`

	Text string `json:"text,omitempty"`
	IsFinal bool `json:"is_final,omitempty"`
	SpeechFinal bool `json:"speech_final,omitempty"`

	Message string `json:"message,omitempty"`

}

// NewTranscriber dials xAI and waits for transcript.created.
func NewTranscriber(Parent context.Context) (*Transcriber, error) {

	APIKey := os.Getenv("XAI_API_KEY")

	if APIKey == "" {

		return nil, errors.New("XAI_API_KEY not set")

	}

	DialCtx, DialCancel := context.WithTimeout(Parent, 5*time.Second)
	defer DialCancel()

	Headers := http.Header{}
	Headers.Set("Authorization", "Bearer "+APIKey)

	Conn, _, ErrDial := websocket.DefaultDialer.DialContext(DialCtx, xaiSTTWebSocketURL(), Headers)

	if ErrDial != nil {

		return nil, fmt.Errorf("xai stt dial: %w", ErrDial)

	}

	Ctx, Cancel := context.WithCancel(Parent)

	T := &Transcriber{

		conn: Conn,

		cancel: Cancel,
		ready: make(chan struct{}),
		done: make(chan struct{}),

		pcmBuf: make([]byte, 0, xaiPCMChunkBytes*2),
	}

	go T.readLoop(Ctx)

	select {

	case <-T.ready:

	case <-time.After(transcribeReadyTimeout):

		T.Close()
		return nil, errors.New("xai stt never sent transcript.created")

	case <-Ctx.Done():

		T.Close()
		return nil, Ctx.Err()

	}

	return T, nil

}

// SetOnUpdate registers a callback for streaming transcript events.
func (T *Transcriber) SetOnUpdate(Fn OnTranscriptFunc) {

	T.onUpdateMu.Lock()
	T.onUpdate = Fn
	T.onUpdateMu.Unlock()

}

// Send appends PCM and flushes in ~100ms chunks (xAI recommendation).
func (T *Transcriber) Send(PCM []byte) error {

	if T == nil || len(PCM) == 0 {

		return errors.New("transcriber closed")

	}

	select {

	case <-T.done:

		return errors.New("transcriber finished")

	default:

	}

	T.writeMu.Lock()
	defer T.writeMu.Unlock()

	if T.conn == nil {

		return errors.New("transcriber closed")

	}

	T.pcmBuf = append(T.pcmBuf, PCM...)

	for len(T.pcmBuf) >= xaiPCMChunkBytes {

		if Err := T.writePCM(T.pcmBuf[:xaiPCMChunkBytes]); Err != nil {

			return Err

		}

		T.pcmBuf = T.pcmBuf[xaiPCMChunkBytes:]

	}

	return nil

}

func (T *Transcriber) writePCM(PCM []byte) error {

	T.startHardTimeout()

	T.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	if Err := T.conn.WriteMessage(websocket.BinaryMessage, PCM); Err != nil {

		return Err

	}

	return nil

}

func (T *Transcriber) startHardTimeout() {

	T.timeoutOnce.Do(func() {

		go func() {

			select {

			case <-time.After(transcribeHardTimeout):

				Utils.Logger.Warn("Receive", "Transcriber: hard timeout reached, finalizing")
				T.Finalize()

			case <-T.done:

			}

		}()

	})

}

func (T *Transcriber) Done() bool {

	if T == nil {

		return true

	}

	select {

	case <-T.done:

		return true

	default:

		return false

	}

}

// Finalize flushes audio, sends audio.done, and waits for transcript.done.
func (T *Transcriber) Finalize() {

	T.startOnce.Do(func() {

		T.writeMu.Lock()

		if T.conn != nil && len(T.pcmBuf) > 0 {

			_ = T.writePCM(T.pcmBuf)
			T.pcmBuf = T.pcmBuf[:0]

		}

		if T.conn != nil {

			T.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			_ = T.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"audio.done"}`))

		}

		T.writeMu.Unlock()

	})

	select {

	case <-T.done:

	case <-time.After(transcribeDoneWait):

		Utils.Logger.Warn("Receive", "Transcriber: transcript.done not received in time")

	}

	T.Close()

}

// Result returns the best transcript text after Finalize.
func (T *Transcriber) Result() string {

	return T.bestText()

}

func (T *Transcriber) bestText() string {

	T.textMu.Lock()
	defer T.textMu.Unlock()

	if T.text != "" {

		return T.text

	}

	return T.interim

}

// Close shuts the WebSocket. Idempotent.
func (T *Transcriber) Close() {

	T.closeOnce.Do(func() {

		if T.cancel != nil {

			T.cancel()

		}

		T.writeMu.Lock()

		if T.conn != nil {

			_ = T.conn.Close()

			T.conn = nil

		}

		T.writeMu.Unlock()

		T.signalDone()

	})

}

func (T *Transcriber) signalDone() {

	T.doneOnce.Do(func() {

		close(T.done)

	})

}

func (T *Transcriber) emitUpdate(Upd TranscriptUpdate) {

	if Upd.Text == "" && !Upd.SpeechFinal {

		return

	}

	T.onUpdateMu.Lock()
	Fn := T.onUpdate // Copies under lock in case it changes concurrently
	T.onUpdateMu.Unlock()

	if Fn != nil {

		Fn(Upd)

	}

}

func (T *Transcriber) absorbPartial(Env sttEnvelope) {

	if Env.Text == "" {

		if Env.SpeechFinal {

			T.emitUpdate(TranscriptUpdate{Text: T.bestText(), IsFinal: true, SpeechFinal: true})

		}

		return

	}

	T.textMu.Lock()

	if Env.IsFinal {

		if T.text == "" {

			T.text = Env.Text

		} else {

			T.text = T.text + " " + Env.Text

		}

	} else {

		T.interim = Env.Text

	}

	Best := T.text

	if Best == "" {

		Best = T.interim

	}

	T.textMu.Unlock()

	T.emitUpdate(TranscriptUpdate{

		Text: Best,

		IsFinal:     Env.IsFinal,
		SpeechFinal: Env.SpeechFinal,
	})

}

func (T *Transcriber) readLoop(Ctx context.Context) {

	defer T.signalDone()

	defer func() {

		if r := recover(); r != nil {

			Utils.Logger.Error("Receive", fmt.Sprintf("Transcriber readLoop panic: %v", r))

		}

	}()

	Conn := T.conn

	if Conn == nil {

		return

	}

	for {

		if Ctx.Err() != nil {

			return

		}

		Conn.SetReadDeadline(time.Now().Add(transcribeHardTimeout + 5*time.Second))

		MessageType, Data, ErrRead := Conn.ReadMessage()

		if ErrRead != nil {

			if Ctx.Err() == nil {

				Utils.Logger.Warn("Receive", fmt.Sprintf("Transcriber read ended: %v", ErrRead))

			}

			return

		}

		if MessageType != websocket.TextMessage {

			continue

		}

		var Env sttEnvelope

		if ErrUnmarshal := json.Unmarshal(Data, &Env); ErrUnmarshal != nil {

			if os.Getenv(sttDebugEnv) != "" {

				Utils.Logger.Warn("Receive", fmt.Sprintf("STT JSON unmarshal: %v raw=%q", ErrUnmarshal, string(Data)))

			}

			continue

		}

		if os.Getenv(sttDebugEnv) != "" {

			Utils.Logger.Info("Receive", fmt.Sprintf("STT event: %s", string(Data)))

		}

		switch Env.Type {

		case "transcript.created":

			select {

			case <-T.ready:

			default:

				close(T.ready)

			}

		case "transcript.partial":

			T.absorbPartial(Env)

		case "transcript.done":

			if Env.Text != "" {

				T.textMu.Lock()

				if T.text == "" {

					T.text = Env.Text

				}

				T.textMu.Unlock()

			} else if os.Getenv(sttDebugEnv) != "" {

				Utils.Logger.Info("Receive", "STT transcript.done with empty text (WebSocket)")

			}

			T.emitUpdate(TranscriptUpdate{Text: T.bestText(), IsFinal: true, SpeechFinal: true})
			return

		case "error":

			Utils.Logger.Error("Receive", fmt.Sprintf("xAI STT error: %s", Env.Message))
			return

		}

	}

}

func sttLanguageValue() string {

	if V := os.Getenv(sttLanguageEnv); V != "" {

		return V

	}

	return sttLanguage

}

func xaiSTTWebSocketURL() string {

	Lang := url.QueryEscape(sttLanguageValue())
	Endpointing := strconv.Itoa(sttEndpointingMS())

	return "wss://api.x.ai/v1/stt" +
		"?sample_rate=16000" +
		"&encoding=pcm" +
		"&language=" + Lang +
		"&interim_results=true" +
		"&endpointing=" + Endpointing +
		"&filler_words=false"

}

func sttEndpointingMS() int {

	V := os.Getenv(envSTTEndpointing)

	if V == "" {

		return defaultEndpointing

	}

	Parsed, Err := strconv.Atoi(V)

	if Err != nil || Parsed < 0 || Parsed > 5000 {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Invalid %s=%q, using %d", envSTTEndpointing, V, defaultEndpointing))
		return defaultEndpointing

	}

	return Parsed

}
