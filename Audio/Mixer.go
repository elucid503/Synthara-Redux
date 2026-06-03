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

// PCMFrameProvider supplies raw 20ms stereo PCM (FrameSize*Channels int16 values).
type PCMFrameProvider interface {

	ProvidePCMFrame() ([]int16, error)
	Close()

}

// MixerProvider is the single Opus encode point for guild voice: speed, effects, volume, duck, overlays.
type MixerProvider struct {

	mu sync.Mutex

	source PCMFrameProvider
	effects *EffectsProcessor
	volume *VolumeProcessor

	residual []float32
	pos float64
	srcEOF bool

	cueMu sync.Mutex

	cueFrames [][]int16
	cuePos int

	ttsFrames [][]int16
	ttsPos int

	overlayActive atomic.Bool
	ttsActive atomic.Bool

	captureDuckActive atomic.Bool
	captureDuckEpoch atomic.Uint64

	encoder *gopus.Encoder

	work []float32
	pcmOut []int16

}

func NewMixerProvider() (*MixerProvider, error) {

	loadVoiceCues()

	encoder, err := gopus.NewEncoder(SampleRate, Channels, gopus.Audio)

	if err != nil {

		return nil, err

	}

	encoder.SetBitrate(OpusBitrate)

	return &MixerProvider{

		encoder: encoder,

		work: make([]float32, FrameSize*Channels),
		pcmOut: make([]int16, FrameSize*Channels),


	}, nil

}

func (mixer *MixerProvider) SetSource(provider PCMFrameProvider) {

	if mixer == nil {

		return

	}

	mixer.mu.Lock()

	mixer.source = provider
	mixer.residual = mixer.residual[:0]

	mixer.pos = 0
	mixer.srcEOF = false

	mixer.mu.Unlock()

}

func (mixer *MixerProvider) SetEffects(processor *EffectsProcessor) {

	if mixer == nil {

		return

	}

	mixer.mu.Lock()
	mixer.effects = processor
	mixer.mu.Unlock()

}

func (mixer *MixerProvider) SetVolumeProcessor(processor *VolumeProcessor) {

	if mixer == nil {

		return

	}

	mixer.mu.Lock()
	mixer.volume = processor
	mixer.mu.Unlock()

}

func (mixer *MixerProvider) SetCaptureDuck(active bool) {

	if mixer == nil {

		return

	}

	if active {

		mixer.captureDuckEpoch.Add(1)
		mixer.captureDuckActive.Store(true)

		return

	}

	mixer.captureDuckActive.Store(false)

}

func (mixer *MixerProvider) EndCaptureDuckAfter(delay time.Duration) {

	if mixer == nil {

		return

	}

	if delay <= 0 {

		mixer.captureDuckActive.Store(false)
		return

	}

	epoch := mixer.captureDuckEpoch.Load()

	go func(target *MixerProvider, expected uint64, wait time.Duration) {

		time.Sleep(wait)

		if target.captureDuckEpoch.Load() != expected {

			return

		}

		target.captureDuckActive.Store(false)

	}(mixer, epoch, delay)

}

func (mixer *MixerProvider) PlayTTSOverlay(frames [][]int16) {

	if mixer == nil || len(frames) == 0 {

		return

	}

	ttsDuration := time.Duration(len(frames)) * 20 * time.Millisecond

	mixer.SetCaptureDuck(true)
	mixer.EndCaptureDuckAfter(ttsDuration + 100*time.Millisecond)

	mixer.cueMu.Lock()

	mixer.ttsFrames = frames
	mixer.ttsPos = 0

	mixer.cueMu.Unlock()

	mixer.ttsActive.Store(true)

}

func (mixer *MixerProvider) PlayCue(kind CueKind) {

	if mixer == nil {
		return
	}

	frames := cueFrames(kind)

	if len(frames) == 0 {

		return

	}

	mixer.cueMu.Lock()
	mixer.cueFrames = frames
	mixer.cuePos = 0
	mixer.cueMu.Unlock()

	mixer.overlayActive.Store(true)

}

func (mixer *MixerProvider) ProvideOpusFrame() ([]byte, error) {

	if mixer == nil {

		return nil, nil

	}

	mixer.mu.Lock()
	defer mixer.mu.Unlock()

	speed := 1.0

	if mixer.effects != nil {
		speed = mixer.effects.SpeedRatio()
	}

	mixer.refillLocked(speed)
	hasMusic := mixer.readMusicFrameLocked(speed)

	overlayPCM, hasOverlay := mixer.nextOverlayFrame()

	if !hasMusic && !hasOverlay {

		return nil, nil

	}

	if !hasMusic {

		clearFloats(mixer.work)

	} else {

		if mixer.effects != nil {

			mixer.effects.ApplyReverb(mixer.work)

		}

		gain := float32(1)

		if mixer.volume != nil {

			gain = mixer.volume.VolumeGain()

		}

		if mixer.captureDuckActive.Load() {

			gain *= playbackDuckGain

		}

		if gain != 1 {

			for i := range mixer.work {

				mixer.work[i] *= gain

			}

		}

	}

	if hasOverlay {

		// Mix overlay on top of music. No ducking or effects applied to overlay.

		limit := FrameSize * Channels

		if len(overlayPCM) < limit {

			limit = len(overlayPCM)

		}

		for i := 0; i < limit; i++ {

			mixer.work[i] += float32(overlayPCM[i]) / 32768.

		}

	}

	floatToPCM(mixer.work, mixer.pcmOut) // Clips to [-1,1] and converts to int16

	return mixer.encoder.Encode(mixer.pcmOut, FrameSize, MaxPacketSize) // back to Opus

}

func (mixer *MixerProvider) refillLocked(speed float64) {

	if mixer.source == nil {

		return

	}

	need := int(mixer.pos+float64(FrameSize-1)*speed) + 2

	for len(mixer.residual)/Channels < need {

		frame, err := mixer.source.ProvidePCMFrame()

		if err == io.EOF {

			mixer.srcEOF = true
			return

		}

		if err != nil || frame == nil {

			return

		}

		for _, sample := range frame {

			mixer.residual = append(mixer.residual, float32(sample)/32768.0)

		}

	}

}

func (mixer *MixerProvider) readMusicFrameLocked(speed float64) bool {

	inFrames := len(mixer.residual) / Channels
	need := int(mixer.pos+float64(FrameSize-1)*speed) + 2

	if inFrames < need && !mixer.srcEOF {
		return false
	}

	if inFrames < 2 {
		return false
	}

	for outFrame := 0; outFrame < FrameSize; outFrame++ {

		inIdx := int(mixer.pos)

		if inIdx+1 >= inFrames {

			for k := outFrame * Channels; k < FrameSize*Channels; k++ {

				mixer.work[k] = 0

			}

			mixer.pos += float64(FrameSize-outFrame) * speed
			break

		}

		frac := float32(mixer.pos - float64(inIdx))

		base0 := inIdx * Channels
		base1 := base0 + Channels

		for ch := 0; ch < Channels; ch++ {

			s1 := mixer.residual[base0+ch]
			s2 := mixer.residual[base1+ch]

			mixer.work[outFrame*Channels+ch] = s1 + (s2-s1)*frac

		}

		mixer.pos += speed

	}

	consumed := int(mixer.pos)

	if consumed > inFrames {

		consumed = inFrames

	}

	if consumed > 0 {

		mixer.residual = append(mixer.residual[:0], mixer.residual[consumed*Channels:]...)
		mixer.pos -= float64(consumed)

	}

	return true

}

func (mixer *MixerProvider) nextOverlayFrame() ([]int16, bool) {

	mixer.cueMu.Lock()
	defer mixer.cueMu.Unlock()

	if mixer.overlayActive.Load() && mixer.cuePos < len(mixer.cueFrames) {

		frame := mixer.cueFrames[mixer.cuePos]
		mixer.cuePos++

		if mixer.cuePos >= len(mixer.cueFrames) {

			mixer.overlayActive.Store(false)

		}

		return frame, true

	}

	if mixer.ttsActive.Load() && mixer.ttsPos < len(mixer.ttsFrames) {

		frame := mixer.ttsFrames[mixer.ttsPos]
		mixer.ttsPos++

		if mixer.ttsPos >= len(mixer.ttsFrames) {

			mixer.ttsActive.Store(false)

		}

		return frame, true

	}

	return nil, false

}

func (mixer *MixerProvider) Close() {

	if mixer == nil {

		return

	}

	mixer.overlayActive.Store(false)
	mixer.ttsActive.Store(false)
	mixer.captureDuckActive.Store(false)

	mixer.cueMu.Lock()

	mixer.cueFrames = nil
	mixer.cuePos = 0

	mixer.ttsFrames = nil
	mixer.ttsPos = 0

	mixer.cueMu.Unlock()

	mixer.mu.Lock()

	source := mixer.source

	mixer.source = nil
	mixer.residual = nil

	mixer.mu.Unlock()

	if source != nil {

		source.Close()

	}

}
