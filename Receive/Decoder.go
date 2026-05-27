package Receive

import (
	"errors"
	"sync"
	"time"

	"layeh.com/gopus"
)

const (
	OpusSampleRate = 48000
	OpusChannels   = 2
	OpusFrameSamples = 960

	TargetSampleRate = 16000
	TargetChannels   = 1
	TargetFrameSamples = 320

	DownsampleRatio = OpusSampleRate / TargetSampleRate

	opusGapResetThreshold = 120 * time.Millisecond
)

// OpusDecoder decodes Discord voice Opus to 16 kHz mono PCM16.
type OpusDecoder struct {

	decoder *gopus.Decoder
	mu      sync.Mutex
	lastAt  time.Time
}

func NewOpusDecoder() (*OpusDecoder, error) {

	Dec, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels)

	if Err != nil {

		return nil, Err

	}

	return &OpusDecoder{decoder: Dec}, nil

}

func (D *OpusDecoder) Decode(Opus []byte) ([]int16, error) {

	if D == nil || D.decoder == nil {

		return nil, errors.New("decoder closed")

	}

	if len(Opus) == 0 {

		return make([]int16, TargetFrameSamples), nil

	}

	D.mu.Lock()
	defer D.mu.Unlock()

	Now := time.Now()

	if !D.lastAt.IsZero() && Now.Sub(D.lastAt) > opusGapResetThreshold {

		if Fresh, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels); Err == nil {

			D.decoder = Fresh

		}

	}

	D.lastAt = Now

	Stereo, ErrDecode := D.decoder.Decode(Opus, OpusFrameSamples, false)

	if ErrDecode != nil {

		if Fresh, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels); Err == nil {

			D.decoder = Fresh

		}

		return nil, ErrDecode

	}

	return downsampleStereoTo16kMono(Stereo), nil

}

func downsampleStereoTo16kMono(Stereo []int16) []int16 {

	Frames := len(Stereo) / OpusChannels
	OutLen := Frames / DownsampleRatio
	Out := make([]int16, OutLen)

	for i := 0; i < OutLen; i++ {

		var Sum int32

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

func Int16ToBytesLE(Samples []int16) []byte {

	Out := make([]byte, len(Samples)*2)

	for i, S := range Samples {

		U := uint16(S)
		Out[i*2] = byte(U)
		Out[i*2+1] = byte(U >> 8)

	}

	return Out

}

func (D *OpusDecoder) Reset() {

	if D == nil {

		return

	}

	D.mu.Lock()
	defer D.mu.Unlock()

	if Fresh, Err := gopus.NewDecoder(OpusSampleRate, OpusChannels); Err == nil {

		D.decoder = Fresh

	}

	D.lastAt = time.Time{}

}

func (D *OpusDecoder) Close() {

	D.mu.Lock()
	defer D.mu.Unlock()
	D.decoder = nil

}
