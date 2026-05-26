package Receive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"Synthara-Redux/Utils"

	"github.com/gorilla/websocket"
)

const (

	sttLanguage = "en" 	// English only

	sttLanguageEnv = "VOICE_STT_LANGUAGE"

	xaiPCMChunkBytes = 3200 // ~100ms PCM16 mono @ 16kHz (xAI native rate).

	sttDebugEnv = "VOICE_STT_DEBUG"

	transcribeHardTimeout = 12 * time.Second
	transcribeReadyTimeout = 5 * time.Second
	transcribeDoneWait = 5 * time.Second

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
	done  chan struct{}

	writeMu sync.Mutex
	pcmBuf []byte

	startOnce sync.Once
	closeOnce sync.Once
	doneOnce  sync.Once

	textMu sync.Mutex
	text string
	interim string
	hadFinal atomic.Bool

	onUpdate OnTranscriptFunc
	onUpdateMu sync.Mutex

	bytesSent atomic.Int64
	finishedAt atomic.Int64

}

type sttEnvelope struct {

	Type string `json:"type"`

	Text string `json:"text,omitempty"`
	IsFinal bool `json:"is_final,omitempty"`
	SpeechFinal bool `json:"speech_final,omitempty"`

	Message string `json:"message,omitempty"`

}

// TranscribeREST uploads raw PCM in one shot (fallback when WebSocket STT is empty).
func TranscribeREST(Ctx context.Context, PCM []byte, SampleRate int) (string, error) {

	return transcribeRESTPost(Ctx, PCM, SampleRate, true)

}

// TranscribeRESTWake is for short wake-word probes (no keyterm on short clips).
func TranscribeRESTWake(Ctx context.Context, PCM []byte, SampleRate int) (string, error) {

	return transcribeRESTPost(Ctx, PCM, SampleRate, false)

}

func transcribeRESTPost(Ctx context.Context, PCM []byte, SampleRate int, withKeyterm bool) (string, error) {

	if len(PCM) == 0 {

		return "", errors.New("empty pcm")

	}

	APIKey := os.Getenv("XAI_API_KEY")

	if APIKey == "" {

		return "", errors.New("XAI_API_KEY not set")

	}

	var Body bytes.Buffer
	Writer := multipart.NewWriter(&Body)

	Lang := sttLanguageValue()

	_ = Writer.WriteField("audio_format", "pcm")
	_ = Writer.WriteField("sample_rate", strconv.Itoa(SampleRate))
	_ = Writer.WriteField("language", Lang)
	_ = Writer.WriteField("format", "true")

	if withKeyterm {

		_ = Writer.WriteField("keyterm", "Synthara")

	}

	FilePart, Err := Writer.CreateFormFile("file", "audio.pcm")

	if Err != nil {

		return "", Err

	}

	if _, Err = FilePart.Write(PCM); Err != nil {

		return "", Err

	}

	if Err = Writer.Close(); Err != nil {

		return "", Err

	}

	Req, Err := http.NewRequestWithContext(Ctx, http.MethodPost, "https://api.x.ai/v1/stt", &Body)

	if Err != nil {

		return "", Err

	}

	Req.Header.Set("Authorization", "Bearer "+APIKey)
	Req.Header.Set("Content-Type", Writer.FormDataContentType())

	Resp, Err := http.DefaultClient.Do(Req)

	if Err != nil {

		return "", Err

	}

	defer Resp.Body.Close()

	RespBody, Err := io.ReadAll(Resp.Body)

	if Err != nil {

		return "", Err

	}

	if Resp.StatusCode != http.StatusOK {

		return "", fmt.Errorf("xai rest stt http %d: %s", Resp.StatusCode, string(RespBody))

	}

	var Parsed struct {
		Text string `json:"text"`
	}

	if Err = json.Unmarshal(RespBody, &Parsed); Err != nil {

		return "", fmt.Errorf("xai rest stt json: %w body=%s", Err, string(RespBody))

	}

	return Parsed.Text, nil

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

	go func() {

		select {

		case <-time.After(transcribeHardTimeout):

			Utils.Logger.Warn("Receive", "Transcriber: hard timeout reached, finalizing")
			T.Finalize()

		case <-T.done:

		}

	}()

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

	T.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	if Err := T.conn.WriteMessage(websocket.BinaryMessage, PCM); Err != nil {

		return Err

	}

	T.bytesSent.Add(int64(len(PCM)))
	return nil

}

// BytesSent returns total PCM bytes written to the STT socket.
func (T *Transcriber) BytesSent() int64 {

	return T.bytesSent.Load()

}

// Flush sends any buffered PCM still waiting to go out.
func (T *Transcriber) Flush() error {

	T.writeMu.Lock()
	defer T.writeMu.Unlock()

	if T.conn == nil || len(T.pcmBuf) == 0 {

		return nil

	}

	Err := T.writePCM(T.pcmBuf)
	T.pcmBuf = T.pcmBuf[:0]
	return Err

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

// SpeechEnded reports whether any is_final partial arrived.
func (T *Transcriber) SpeechEnded() bool {

	return T.hadFinal.Load()

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
		T.finishedAt.Store(time.Now().UnixNano())

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

		T.hadFinal.Store(true)

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

		IsFinal: Env.IsFinal,
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

	return "wss://api.x.ai/v1/stt" +
		"?sample_rate=16000" +
		"&encoding=pcm" +
		"&language=" + Lang +
		"&interim_results=true" +
		"&endpointing=800" +
		"&filler_words=false"

}
