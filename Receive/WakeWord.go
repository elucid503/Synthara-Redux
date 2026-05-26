package Receive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"Synthara-Redux/Utils"

	sherpa "github.com/k2-fsa/sherpa-onnx-go-linux"
)

// Default model paths for the on-device KWS model. Can be overridden with environment variables.
const (

	defaultModelDir = "./Receive/Models/kws"

	envEncoder = "KWS_ENCODER"
	envDecoder = "KWS_DECODER"
	envJoiner = "KWS_JOINER"
	envTokens = "KWS_TOKENS"

	envKeywordsFile = "KWS_KEYWORDS_FILE"
	envBpeVocab = "KWS_BPE_VOCAB"
	envKeywordsScore = "KWS_KEYWORDS_SCORE"
	envKeywordsThreshold = "KWS_KEYWORDS_THRESHOLD"
	envMaxActivePaths = "KWS_MAX_ACTIVE_PATHS"

	defaultKeywordsScore = 16.0
	defaultKeywordsThreshold = 0.000025 // admittedly low, but false-positives not really a big problem here...
	defaultMaxActivePaths = 16

)

var (

	wakeSpotter *sherpa.KeywordSpotter
	wakeInitOnce sync.Once
	wakeInitErr error

)

// initWakeSpotter lazily loads the on-device KWS model. The KWS model only needs to be loaded once and is shared across all sessions under a mutex-protected singleton.
func initWakeSpotter() {

	Encoder := envOrDefault(envEncoder, filepath.Join(defaultModelDir, "encoder.onnx"))
	Decoder := envOrDefault(envDecoder, filepath.Join(defaultModelDir, "decoder.onnx"))

	Joiner := envOrDefault(envJoiner, filepath.Join(defaultModelDir, "joiner.onnx"))
	Tokens := envOrDefault(envTokens, filepath.Join(defaultModelDir, "tokens.txt"))

	KeywordsFile := envOrDefault(envKeywordsFile, filepath.Join(defaultModelDir, "keywords.txt"))
	BpeVocab := envOrDefault(envBpeVocab, filepath.Join(defaultModelDir, "bpe.model"))

	for _, Path := range []string{Encoder, Decoder, Joiner, Tokens, KeywordsFile, BpeVocab} {

		if _, ErrStat := os.Stat(Path); ErrStat != nil {

			wakeInitErr = fmt.Errorf("missing KWS file %q: %w", Path, ErrStat)
			Utils.Logger.Warn("Receive", fmt.Sprintf("Wake word disabled: %s", wakeInitErr.Error()))

			return

		}

	}

	Cfg := &sherpa.KeywordSpotterConfig{

		FeatConfig: sherpa.FeatureConfig{

			SampleRate: TargetSampleRate,
			FeatureDim: 80,

		},

		ModelConfig: sherpa.OnlineModelConfig{

			Transducer: sherpa.OnlineTransducerModelConfig{

				Encoder: Encoder,
				Decoder: Decoder,
				Joiner:  Joiner,

			},

			Tokens: Tokens,
			BpeVocab: BpeVocab,

			NumThreads:   1,

			Provider: "cpu",

			ModelType: "zipformer2",
			ModelingUnit: "bpe",

		},

		MaxActivePaths: envIntOrDefault(envMaxActivePaths, defaultMaxActivePaths),
		KeywordsFile: KeywordsFile,

		KeywordsScore: envFloatOrDefault(envKeywordsScore, defaultKeywordsScore),
		KeywordsThreshold: envFloatOrDefault(envKeywordsThreshold, defaultKeywordsThreshold),

	}

	Spotter := sherpa.NewKeywordSpotter(Cfg)

	if Spotter == nil {

		wakeInitErr = errors.New("sherpa-onnx returned nil keyword spotter (check model files)")
		Utils.Logger.Warn("Receive", fmt.Sprintf("Wake word disabled: %s", wakeInitErr.Error()))

		return

	}

	wakeSpotter = Spotter

}

// WakeWordEnabled returns true once the global KWS model is loaded and ready.
func WakeWordEnabled() bool {

	wakeInitOnce.Do(initWakeSpotter)
	return wakeSpotter != nil

}

const (

	envKWSBatchMS = "KWS_BATCH_MS"
	defaultKWSBatchMS = 40

)

// WakeStream wraps a per-session sherpa OnlineStream so callers don't need to import the C-bound package directly.
type WakeStream struct {

	stream *sherpa.OnlineStream
	mu sync.Mutex

	pending []float32

	batchSamples int

}

// NewWakeStream creates a fresh KWS stream tied to the global wakeSpotter. Returns nil if KWS isn't available.
func NewWakeStream() *WakeStream {

	if !WakeWordEnabled() {

		return nil

	}

	BatchMS := envIntOrDefault(envKWSBatchMS, defaultKWSBatchMS)

	if BatchMS < 20 {

		BatchMS = 20

	}

	BatchSamples := TargetSampleRate * BatchMS / 1000

	return &WakeStream{

		stream: sherpa.NewKeywordStream(wakeSpotter),
		batchSamples: BatchSamples,

		pending: make([]float32, 0, BatchSamples*2),

	}

}

// Feed pushes a chunk of 16kHz mono float32 PCM through the spotter and returns the matched keyword (or "" if none)
func (W *WakeStream) Feed(Samples []float32) string {

	if W == nil || W.stream == nil || wakeSpotter == nil || len(Samples) == 0 {

		return ""

	}

	W.mu.Lock()
	defer W.mu.Unlock()

	W.pending = append(W.pending, Samples...)

	for len(W.pending) >= W.batchSamples {

		Chunk := W.pending[:W.batchSamples]
		W.pending = W.pending[W.batchSamples:]

		if Hit := W.feedChunk(Chunk); Hit != "" {

			W.pending = W.pending[:0]
			return Hit

		}

	}

	return ""

}

// FlushPartial runs KWS on buffered audio before a full batch is ready.
func (W *WakeStream) FlushPartial() string {

	if W == nil || W.stream == nil || wakeSpotter == nil {

		return ""

	}

	W.mu.Lock()
	defer W.mu.Unlock()

	if len(W.pending) < W.batchSamples/2 {

		return ""

	}

	Chunk := append([]float32(nil), W.pending...)
	W.pending = W.pending[:0]

	return W.feedChunk(Chunk)

}

// FlushPending feeds any buffered audio (call after speech ends for lower latency).
func (W *WakeStream) FlushPending() string {

	if W == nil || len(W.pending) == 0 {

		return ""

	}

	W.mu.Lock()
	defer W.mu.Unlock()

	Chunk := append([]float32(nil), W.pending...)
	W.pending = W.pending[:0]

	return W.feedChunk(Chunk)

}

func (W *WakeStream) feedChunk(Samples []float32) string {

	if len(Samples) == 0 {

		return ""

	}

	W.stream.AcceptWaveform(TargetSampleRate, Samples)

	for wakeSpotter.IsReady(W.stream) {

		wakeSpotter.Decode(W.stream)

	}

	Result := wakeSpotter.GetResult(W.stream)

	if Result == nil || Result.Keyword == "" {

		return ""

	}

	return strings.ToLower(strings.TrimPrefix(Result.Keyword, "@"))

}

// Reset clears the spotter's accumulated state. Must be called after a detection to avoid repeated hits!
func (W *WakeStream) Reset() {

	if W == nil || W.stream == nil || wakeSpotter == nil {

		return

	}

	W.mu.Lock()
	defer W.mu.Unlock()

	wakeSpotter.Reset(W.stream)

}

// Close releases the underlying sherpa stream.
func (W *WakeStream) Close() {

	W.mu.Lock()
	defer W.mu.Unlock()

	if W.stream != nil {

		sherpa.DeleteOnlineStream(W.stream)
		W.stream = nil

	}

}

func envOrDefault(Key, Default string) string {

	if V := os.Getenv(Key); V != "" {

		return V

	}

	return Default

}

func envFloatOrDefault(Key string, Default float64) float32 {

	V := os.Getenv(Key)

	if V == "" {

		return float32(Default)

	}

	Parsed, Err := strconv.ParseFloat(V, 32)

	if Err != nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Invalid %s=%q, using %.3f", Key, V, Default))
		return float32(Default)

	}

	return float32(Parsed)

}

func envIntOrDefault(Key string, Default int) int {

	V := os.Getenv(Key)

	if V == "" {

		return Default

	}

	Parsed, Err := strconv.Atoi(V)

	if Err != nil {

		Utils.Logger.Warn("Receive", fmt.Sprintf("Invalid %s=%q, using %d", Key, V, Default))
		return Default

	}

	return Parsed

}
