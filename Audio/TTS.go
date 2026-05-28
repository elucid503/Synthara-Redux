//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (

	ttsAPIEndpoint = "https://api.x.ai/v1/tts"

	ttsCacheDir = "./Cache/TTS"

	ttsModel = "grok-voice-latest"
	ttsVoice = "eve"
	ttsLanguage = "en"

	ttsHTTPTimeout = 15 * time.Second
	ttsMaxChars = 200

)

type ttsOutputFormat struct {

	Codec string `json:"codec"`
	SampleRate int `json:"sample_rate"`

}

type ttsRequest struct {

	Text string `json:"text"`
	Model string `json:"model"`

	VoiceID string `json:"voice_id"`
	Language string `json:"language"`

	OutputFormat ttsOutputFormat `json:"output_format"`

}

// GenerateTTS returns 48kHz stereo PCM frames for text, loading from disk cache when available.
func GenerateTTS(text string) ([][]int16, error) {

	text = sanitizeTTSInput(text)

	if text == "" {

		return nil, fmt.Errorf("empty TTS input after sanitization")

	}

	CachePath := filepath.Join(ttsCacheDir, ttsCacheKey(text)+".pcm")

	if Cached, Err := loadTTSCache(CachePath); Err == nil {

		return Cached, nil

	}

	Raw, Err := callTTSAPI(text)

	if Err != nil {

		return nil, Err

	}

	saveTTSCache(CachePath, Raw)

	return monoToStereoFrames(Raw), nil

}

// sanitizeTTSInput strips bracketed content, collapses whitespace, and enforces ttsMaxChars.
func sanitizeTTSInput(text string) string {

	var B strings.Builder
	Depth := 0

	for _, R := range text {

		switch R {

			case '(', '[', '{':
				Depth++

			case ')', ']', '}':

				if Depth > 0 {

					Depth--

				}

			default:

				if Depth == 0 {

					B.WriteRune(R)

				}

		}

	}

	Result := strings.TrimSpace(strings.Join(strings.Fields(B.String()), " "))

	if len(Result) <= ttsMaxChars {

		return Result

	}

	Truncated := Result[:ttsMaxChars]

	if I := strings.LastIndex(Truncated, " "); I > 0 {

		return Truncated[:I]

	}

	return Truncated

}

func ttsCacheKey(text string) string {

	H := sha256.Sum256([]byte(text))

	return hex.EncodeToString(H[:])

}

func loadTTSCache(path string) ([][]int16, error) {

	Data, Err := os.ReadFile(path)

	if Err != nil {

		return nil, Err

	}

	return monoToStereoFrames(Data), nil

}

func saveTTSCache(path string, raw []byte) {

	if Err := os.MkdirAll(filepath.Dir(path), 0755); Err != nil {

		return

	}

	_ = os.WriteFile(path, raw, 0644)

}

func callTTSAPI(text string) ([]byte, error) {

	APIKey := os.Getenv("XAI_API_KEY")

	if APIKey == "" {

		return nil, fmt.Errorf("XAI_API_KEY not set")

	}

	Body, Err := json.Marshal(ttsRequest{

		Text: text,
		Model: ttsModel,

		VoiceID: ttsVoice,
		Language: ttsLanguage,

		OutputFormat: ttsOutputFormat{

			Codec: "pcm",
			SampleRate: SampleRate,

		},

	})

	if Err != nil {

		return nil, Err

	}

	Req, Err := http.NewRequest(http.MethodPost, ttsAPIEndpoint, bytes.NewReader(Body))

	if Err != nil {

		return nil, Err

	}

	Req.Header.Set("Authorization", "Bearer "+APIKey)
	Req.Header.Set("Content-Type", "application/json")

	Client := &http.Client{Timeout: ttsHTTPTimeout}

	Resp, Err := Client.Do(Req)

	if Err != nil {

		return nil, fmt.Errorf("TTS request: %w", Err)

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != http.StatusOK {

		ErrBody, _ := io.ReadAll(Resp.Body)

		return nil, fmt.Errorf("TTS API %d: %s", Resp.StatusCode, string(ErrBody))

	}

	return io.ReadAll(Resp.Body)

}

// monoToStereoFrames converts raw 48kHz mono int16 LE PCM bytes into 20ms stereo frames.
func monoToStereoFrames(raw []byte) [][]int16 {

	MonoSamples := len(raw) / 2
	SamplesPerFrame := FrameSize * Channels

	Stereo := make([]int16, MonoSamples*Channels)

	for I := 0; I < MonoSamples; I++ {

		S := int16(binary.LittleEndian.Uint16(raw[I*2:]))
		Stereo[I*2] = S
		Stereo[I*2+1] = S

	}

	FrameCount := len(Stereo) / SamplesPerFrame
	Frames := make([][]int16, FrameCount)

	for I := 0; I < FrameCount; I++ {

		Frames[I] = Stereo[I*SamplesPerFrame : (I+1)*SamplesPerFrame]

	}

	return Frames

}
