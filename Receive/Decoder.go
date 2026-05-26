package Receive

import (
	"errors"
	"sync"
	"time"

	"layeh.com/gopus"
)

const (
	OpusSampleRate = 48000
	OpusChannels = 2
	OpusFrameSamples = 960 // 20ms @ 48kHz, per channel

	TargetSampleRate = 16000
	TargetChannels = 1
	TargetFrameSamples = 320 // 20ms @ 16kHz, mono

	DownsampleRatio = OpusSampleRate / TargetSampleRate // 3

	opusGapResetThreshold = 120 * time.Millisecond // avoids desync

)

// OpusPCMDecoder uses separate decoders for wake-word vs command capture
type OpusPCMDecoder struct {

	wakeDecoder *gopus.Decoder
	captureDecoder *gopus.Decoder
	mu sync.Mutex

	lastWakePacketAt time.Time
	lastCapturePacketAt time.Time

	wakeResets int
	captureResets int

}

// NewOpusPCMDecoder constructs decoders for Discord's opus output format.
func NewOpusPCMDecoder() (*OpusPCMDecoder, error) {

	WakeDec, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels)

	if Err != nil {

		return nil, Err

	}

	CaptureDec, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels)

	if Err != nil {

		return nil, Err

	}

	return &OpusPCMDecoder{

		wakeDecoder: WakeDec,
		captureDecoder: CaptureDec,

	}, nil

}

// DecodeWake decodes for on-device KWS only (listening state).
func (D *OpusPCMDecoder) DecodeWake(Opus []byte) ([]int16, error) {

	return D.decode(&D.wakeDecoder, Opus, &D.lastWakePacketAt, &D.wakeResets)

}

// DecodeCapture decodes for STT capture only (capturing state).
func (D *OpusPCMDecoder) DecodeCapture(Opus []byte) ([]int16, error) {

	return D.decode(&D.captureDecoder, Opus, &D.lastCapturePacketAt, &D.captureResets)

}

func (D *OpusPCMDecoder) decode(Dec **gopus.Decoder, Opus []byte, lastAt *time.Time, resetCount *int) ([]int16, error) {

	if D == nil || Dec == nil || *Dec == nil {

		return nil, errors.New("decoder closed")

	}

	if len(Opus) == 0 {

		return make([]int16, TargetFrameSamples), nil

	}

	D.mu.Lock()
	defer D.mu.Unlock()

	Now := time.Now()

	if !lastAt.IsZero() && Now.Sub(*lastAt) > opusGapResetThreshold {

		D.recreateDecoder(Dec)
		*resetCount++

	}

	*lastAt = Now

	Stereo, ErrDecode := (*Dec).Decode(Opus, OpusFrameSamples, false)

	if ErrDecode != nil {

		D.recreateDecoder(Dec)
		*resetCount++
		return nil, ErrDecode

	}

	return downsampleStereoTo16kMono(Stereo), nil

}

func downsampleStereoTo16kMono(Stereo []int16) []int16 {

	Frames := len(Stereo) / OpusChannels
	OutLen := Frames / DownsampleRatio
	Out := make([]int16, OutLen)

	for i := 0; i < OutLen; i++ {

		Sum := int32(0)

		for j := 0; j < DownsampleRatio; j++ {

			Idx := (i*DownsampleRatio + j) * OpusChannels

			if Idx+1 < len(Stereo) {

				Sum += int32(Stereo[Idx]) + int32(Stereo[Idx+1])

			}

		}

		Out[i] = int16(Sum / int32(DownsampleRatio*OpusChannels))

	}

	return Out

}

func (D *OpusPCMDecoder) recreateDecoder(Dec **gopus.Decoder) {

	Fresh, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels)

	if Err != nil {

		return

	}

	*Dec = Fresh

}

// ResetWakePath clears wake decoder state (after capture / prolonged loss).
func (D *OpusPCMDecoder) ResetWakePath() int {

	if D == nil {

		return 0

	}

	D.mu.Lock()
	defer D.mu.Unlock()

	D.recreateDecoder(&D.wakeDecoder)
	D.lastWakePacketAt = time.Time{}
	D.wakeResets++
	return D.wakeResets

}

// ResetCapturePath clears capture decoder state at the start of a new utterance.
func (D *OpusPCMDecoder) ResetCapturePath() int {

	if D == nil {

		return 0

	}

	D.mu.Lock()
	defer D.mu.Unlock()

	D.recreateDecoder(&D.captureDecoder)
	D.lastCapturePacketAt = time.Time{}
	D.captureResets++

	return D.captureResets

}

// WakeResets returns how often the wake decoder was reset (gap/error recovery).
func (D *OpusPCMDecoder) WakeResets() int {

	if D == nil {

		return 0

	}

	D.mu.Lock()
	defer D.mu.Unlock()
	return D.wakeResets

}

// PCMStats summarizes a PCM buffer for logging.
type PCMStats struct {

	Frames int

	RMS float32
	Peak int16

}

// AmplifyInt16ForKWS boosts quiet Discord frames so the on-device spotter can lock on.
func AmplifyInt16ForKWS(Samples []int16) []int16 {

	const targetPeak int16 = 8000

	Stats := PCMStatsFrom(Samples)

	if Stats.Peak < 40 {

		return Samples

	}

	if Stats.Peak >= targetPeak {

		return Samples

	}

	Gain := float64(targetPeak) / float64(Stats.Peak)

	const maxGain = 20.0

	if Gain > maxGain {

		Gain = maxGain

	}

	Out := make([]int16, len(Samples))

	for i, S := range Samples {

		V := int32(float64(S) * Gain)

		if V > 32767 {

			V = 32767

		} else if V < -32768 {

			V = -32768

		}

		Out[i] = int16(V)

	}

	return Out

}

// FrameHasSpeech returns true when a 16kHz mono frame likely contains voice.
func FrameHasSpeech(Samples []int16) bool {

	Stats := PCMStatsFrom(Samples)

	return Stats.Peak >= 300 || Stats.RMS >= 0.0006

}

// FrameHasSpeechForWake uses a lower bar so STT wake probes collect audio sooner.
func FrameHasSpeechForWake(Samples []int16) bool {

	Stats := PCMStatsFrom(Samples)

	return Stats.Peak >= 150 || Stats.RMS >= 0.0003

}

// PreparePCMForSTT boosts quiet Discord audio so cloud STT can lock onto speech.
func PreparePCMForSTT(PCM []byte) []byte {

	return NormalizePCMBytes(PCM, 12000)

}

// NormalizePCMForSTT scales PCM so peak approaches targetPeak (capped gain).
func NormalizePCMBytes(PCM []byte, targetPeak int16) []byte {

	if len(PCM) < 2 || targetPeak <= 0 {

		return PCM

	}

	Stats := PCMStatsFromBytes(PCM)

	if Stats.Peak < 200 {

		return PCM

	}

	Gain := float64(targetPeak) / float64(Stats.Peak)

	if Gain < 1.0 {

		return PCM

	}

	const maxGain = 25.0

	if Gain > maxGain {

		Gain = maxGain

	}

	Out := make([]byte, len(PCM))

	for i := 0; i < len(PCM)/2; i++ {

		S := int16(uint16(PCM[i*2]) | uint16(PCM[i*2+1])<<8)
		V := int32(float64(S) * Gain)

		if V > 32767 {

			V = 32767

		} else if V < -32768 {

			V = -32768

		}

		U := uint16(int16(V))
		Out[i*2] = byte(U)
		Out[i*2+1] = byte(U >> 8)

	}

	return Out

}

func PCMStatsFromBytes(PCM []byte) PCMStats {

	if len(PCM) < 2 {

		return PCMStats{}

	}

	Samples := make([]int16, len(PCM)/2)

	for i := range Samples {

		Samples[i] = int16(uint16(PCM[i*2]) | uint16(PCM[i*2+1])<<8)

	}

	return PCMStatsFrom(Samples)

}

func PCMStatsFrom(Samples []int16) PCMStats {

	if len(Samples) == 0 {

		return PCMStats{}

	}

	var Peak int16

	for _, S := range Samples {

		if S < 0 {

			if -S > Peak || Peak == 0 {

				Peak = -S

			}

		} else if S > Peak {

			Peak = S

		}

	}

	return PCMStats{

		Frames: 1,
		RMS:    FrameRMS(Samples),
		Peak:   Peak,
	}

}

// Int16ToFloat32 converts PCM samples to normalized [-1, 1] float32, which is expected for on-device KWS models.
func Int16ToFloat32(Samples []int16) []float32 {

	Out := make([]float32, len(Samples))

	for i, S := range Samples {

		Out[i] = float32(S) / 32768.0

	}

	return Out

}

// Int16ToBytesLE serializes int16 samples to little-endian bytes for the xAI streaming endpoint.
func Int16ToBytesLE(Samples []int16) []byte {

	Out := make([]byte, len(Samples)*2)

	for i, S := range Samples {

		U := uint16(S)
		Out[i*2] = byte(U)
		Out[i*2+1] = byte(U >> 8)

	}

	return Out

}

// FrameRMS computes the RMS energy of a PCM frame, normalized to [0, 1].
func FrameRMS(Samples []int16) float32 {

	if len(Samples) == 0 {

		return 0

	}

	var Sum float64

	for _, S := range Samples {

		F := float64(S) / 32768.0
		Sum += F * F

	}

	return float32(Sum / float64(len(Samples)))

}

// Close releases the underlying opus decoders.
func (D *OpusPCMDecoder) Close() {

	D.mu.Lock()
	defer D.mu.Unlock()

	D.wakeDecoder = nil
	D.captureDecoder = nil

}
