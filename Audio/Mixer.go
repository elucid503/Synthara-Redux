//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"layeh.com/gopus"
)

// OpusFrameProvider supplies 20ms Opus frames (which is same contract as disgo voice.OpusFrameProvider).
type OpusFrameProvider interface {
	ProvideOpusFrame() ([]byte, error)
	Close()
}

// MixerProvider is the persistent outbound audio source for a guild voice connection.
type MixerProvider struct {

	mu sync.Mutex

	inner OpusFrameProvider

	cueMu sync.Mutex
	cueFrames [][]int16
	cuePos int

	overlayActive atomic.Bool
	captureDuckActive atomic.Bool
	captureDuckEpoch atomic.Uint64

	dec *gopus.Decoder
	enc *gopus.Encoder

}

// NewMixerProvider builds a mixer with preloaded wake/end cue PCM.
func NewMixerProvider() (*MixerProvider, error) {

	loadVoiceCues()

	if cueLoadErr != nil {

		return nil, cueLoadErr

	}

	Dec, Err := gopus.NewDecoder(SampleRate, Channels)

	if Err != nil {

		return nil, Err

	}

	Enc, Err := gopus.NewEncoder(SampleRate, Channels, gopus.Audio)

	if Err != nil {

		return nil, Err

	}

	Enc.SetBitrate(128000)

	return &MixerProvider{

		dec: Dec,
		enc: Enc,

	}, nil

}

// SetInner swaps the music source without recreating Discord's audio sender.
func (M *MixerProvider) SetInner(provider OpusFrameProvider) {

	if M == nil {

		return

	}

	if Dec, Err := gopus.NewDecoder(SampleRate, Channels); Err == nil {

		M.mu.Lock()
		M.inner = provider
		M.dec = Dec
		M.mu.Unlock()

		return

	}

	M.mu.Lock()
	M.inner = provider
	M.mu.Unlock()

}

// SetCaptureDuck enables or disables music ducking for the full voice-command capture window.
func (M *MixerProvider) SetCaptureDuck(active bool) {

	if M == nil {

		return

	}

	if active {

		M.captureDuckEpoch.Add(1)
		M.captureDuckActive.Store(true)

		return

	}

	M.captureDuckActive.Store(false)

}

// EndCaptureDuckAfter keeps ducking until the end cue finishes, then restores full volume.
func (M *MixerProvider) EndCaptureDuckAfter(delay time.Duration) {

	if M == nil {

		return

	}

	if delay <= 0 {

		M.captureDuckActive.Store(false)

		return

	}

	Epoch := M.captureDuckEpoch.Load()

	go func(Mixer *MixerProvider, E uint64, D time.Duration) {

		time.Sleep(D)

		if Mixer.captureDuckEpoch.Load() != E {

			return

		}

		Mixer.captureDuckActive.Store(false)

	}(M, Epoch, delay)

}

// PlayCue starts (or replaces) a wake/end overlay; safe to call from any goroutine.
func (M *MixerProvider) PlayCue(kind CueKind) {

	if M == nil {

		return

	}

	Frames := cueFrames(kind)

	if len(Frames) == 0 {

		return

	}

	M.cueMu.Lock()
	M.cueFrames = Frames
	M.cuePos = 0
	M.cueMu.Unlock()

	M.overlayActive.Store(true)

}

func (M *MixerProvider) ProvideOpusFrame() ([]byte, error) {

	if M == nil {

		return nil, nil

	}

	M.mu.Lock()
	Inner := M.inner
	Dec := M.dec
	M.mu.Unlock()

	var BaseOpus []byte
	var BaseErr error

	if Inner != nil {

		BaseOpus, BaseErr = Inner.ProvideOpusFrame()

		if BaseErr != nil && BaseErr != io.EOF {

			return nil, BaseErr

		}

	}

	M.cueMu.Lock()

	Overlay := M.overlayActive.Load() && M.cuePos < len(M.cueFrames)

	var CuePCM []int16

	if Overlay {

		CuePCM = M.cueFrames[M.cuePos]
		M.cuePos++

		if M.cuePos >= len(M.cueFrames) {

			M.overlayActive.Store(false)

		}

	}

	M.cueMu.Unlock()

	Duck := M.captureDuckActive.Load()

	// Fast path: normal playback, no capture duck, no cue overlay.
	if !Duck && !Overlay {

		if len(BaseOpus) > 0 {

			return BaseOpus, nil

		}

		return nil, nil

	}

	// Cue-only (no music).
	if Overlay && len(BaseOpus) == 0 {

		Mixed := mixPCMFrame(silencePCMFrame(), CuePCM, 1)

		return M.enc.Encode(Mixed, FrameSize, MaxPacketSize)

	}

	BasePCM := silencePCMFrame()

	if len(BaseOpus) > 0 && Dec != nil {

		Decoded, Err := Dec.Decode(BaseOpus, FrameSize, false)

		if Err == nil && len(Decoded) >= FrameSize*Channels {

			BasePCM = Decoded[:FrameSize*Channels]

		}

	}

	DuckGain := float32(1)

	if Duck {

		DuckGain = playbackDuckGain

	}

	if Overlay {

		Mixed := mixPCMFrame(BasePCM, CuePCM, DuckGain)

		return M.enc.Encode(Mixed, FrameSize, MaxPacketSize)

	}

	// Capture duck only (music playing, no cue this frame).
	Scaled := mixPCMFrame(BasePCM, nil, DuckGain)

	return M.enc.Encode(Scaled, FrameSize, MaxPacketSize)

}

func (M *MixerProvider) Close() {

	if M == nil {

		return

	}

	M.overlayActive.Store(false)
	M.captureDuckActive.Store(false)

	M.cueMu.Lock()
	M.cueFrames = nil
	M.cuePos = 0
	M.cueMu.Unlock()

	M.mu.Lock()
	Inner := M.inner
	M.inner = nil
	M.mu.Unlock()

	if Inner != nil {

		Inner.Close()

	}

}

func silencePCMFrame() []int16 {

	return make([]int16, FrameSize*Channels)

}

func mixPCMFrame(base, cue []int16, duck float32) []int16 {

	N := FrameSize * Channels
	Out := make([]int16, N)

	for I := 0; I < N; I++ {

		B := float32(0)

		if I < len(base) {

			B = float32(base[I]) * duck

		}

		C := float32(0)

		if I < len(cue) {

			C = float32(cue[I])

		}

		V := B + C

		if V > 32767 {

			V = 32767

		} else if V < -32768 {

			V = -32768

		}

		Out[I] = int16(V)

	}

	return Out

}
