//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"sync/atomic"

	"layeh.com/gopus"
)

type VolumeProcessor struct {

	VolumePercent atomic.Int32

	dec *gopus.Decoder
	enc *gopus.Encoder

}

func NewVolumeProcessor() (*VolumeProcessor, error) {

	Dec, Err := gopus.NewDecoder(SampleRate, Channels)

	if Err != nil {

		return nil, Err

	}

	Enc, Err := gopus.NewEncoder(SampleRate, Channels, gopus.Audio)

	if Err != nil {

		return nil, Err

	}

	Enc.SetBitrate(128000)

	Processor := &VolumeProcessor{

		dec: Dec,
		enc: Enc,

	}

	Processor.VolumePercent.Store(100)

	return Processor, nil

}

func (V *VolumeProcessor) SetVolume(Percent int) {

	if V == nil {

		return

	}

	V.VolumePercent.Store(int32(Percent))

}

func (V *VolumeProcessor) VolumeGain() float32 {

	if V == nil {

		return 1

	}

	return float32(V.VolumePercent.Load()) / 100.0

}

func ScalePCM(PCM []int16, Gain float32) []int16 {

	if Gain == 1 || len(PCM) == 0 {

		return PCM

	}

	Out := make([]int16, len(PCM))

	for I, Sample := range PCM {

		V := float32(Sample) * Gain

		if V > 32767 {

			V = 32767

		} else if V < -32768 {

			V = -32768

		}

		Out[I] = int16(V)

	}

	return Out

}

func (V *VolumeProcessor) ProcessOpusFrame(Opus []byte) ([]byte, error) {

	if V == nil || len(Opus) == 0 {

		return Opus, nil

	}

	Gain := V.VolumeGain()

	if Gain == 1 {

		return Opus, nil

	}

	Decoded, Err := V.dec.Decode(Opus, FrameSize, false)

	if Err != nil || len(Decoded) < FrameSize*Channels {

		return Opus, nil

	}

	Scaled := ScalePCM(Decoded[:FrameSize*Channels], Gain)

	return V.enc.Encode(Scaled, FrameSize, MaxPacketSize)

}

// VolumeOpusProvider applies live volume when Opus frames are consumed. A decorator for OpusFrameProvider.
type VolumeOpusProvider struct {

	Inner OpusFrameProvider // Underlying Opus frame provider to wrap.
	Volume *VolumeProcessor

}

func (P *VolumeOpusProvider) ProvideOpusFrame() ([]byte, error) {

	if P == nil || P.Inner == nil {

		return nil, nil

	}

	Frame, Err := P.Inner.ProvideOpusFrame()

	if Err != nil || len(Frame) == 0 {

		return Frame, Err

	}

	if P.Volume == nil {

		return Frame, nil

	}

	return P.Volume.ProcessOpusFrame(Frame)

}

func (P *VolumeOpusProvider) Close() {

	if P == nil {

		return

	}

	if P.Inner != nil {

		P.Inner.Close()

	}

}
