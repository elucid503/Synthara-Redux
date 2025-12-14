//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

/*
#cgo pkg-config: fdk-aac
#include <fdk-aac/aacdecoder_lib.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type FDKAACDecoder struct {

	Decoder C.HANDLE_AACDECODER

}

func NewFDKAACDecoder() (*FDKAACDecoder, error) {

	Decoder := C.aacDecoder_Open(C.TT_MP4_ADTS, 1)

	if Decoder == nil {

		return nil, errors.New("failed to open FDK-AAC decoder")

	}

	return &FDKAACDecoder{Decoder: Decoder}, nil

}

func (Decoder *FDKAACDecoder) Decode(ADTSFrame []byte) ([]int16, error) {

	if len(ADTSFrame) == 0 {

		return nil, errors.New("empty frame")

	}

	// Fill decoder with input data

	InputBuffer := (*C.uchar)(C.CBytes(ADTSFrame))
	defer C.free(unsafe.Pointer(InputBuffer))

	InputSize := C.uint(len(ADTSFrame))
	BytesValid := InputSize

	ErrorCode := C.aacDecoder_Fill(Decoder.Decoder, &InputBuffer, &InputSize, &BytesValid)

	if ErrorCode != C.AAC_DEC_OK {

		return nil, errors.New("aacDecoder_Fill failed")

	}

	// Decode frame

	OutputBuffer := make([]C.short, 2048*2) // Max frame size * channels
	ErrorCode = C.aacDecoder_DecodeFrame(Decoder.Decoder, &OutputBuffer[0], C.int(len(OutputBuffer)), 0)

	if ErrorCode != C.AAC_DEC_OK {

		if ErrorCode == C.AAC_DEC_NOT_ENOUGH_BITS {

			return nil, nil // Need more data

		}

		return nil, errors.New("aacDecoder_DecodeFrame failed")

	}

	// Get stream info to determine actual output size

	StreamInfo := C.aacDecoder_GetStreamInfo(Decoder.Decoder)

	if StreamInfo == nil {

		return nil, errors.New("failed to get stream info")

	}

	NumChannels := int(StreamInfo.numChannels)
	FrameSize := int(StreamInfo.frameSize)
	SampleRate := int(StreamInfo.sampleRate)
	OutputSize := NumChannels * FrameSize

	// Convert C.short to int16

	PCMData := make([]int16, OutputSize)

	for Index := 0; Index < OutputSize; Index++ {

		PCMData[Index] = int16(OutputBuffer[Index])

	}

	// Resample if needed (YouTube audio is often 44.1kHz, Discord needs 48kHz)
	if SampleRate != 48000 {
		PCMData = ResamplePCM(PCMData, SampleRate, 48000, NumChannels)
	}

	return PCMData, nil

}

func (Decoder *FDKAACDecoder) Close() {

	if Decoder.Decoder != nil {

		C.aacDecoder_Close(Decoder.Decoder)
		Decoder.Decoder = nil

	}

}

// ResamplePCM resamples PCM audio from one sample rate to another using linear interpolation
func ResamplePCM(Input []int16, InputRate, OutputRate, Channels int) []int16 {

	if InputRate == OutputRate {
		return Input
	}

	Ratio := float64(InputRate) / float64(OutputRate)
	InputFrames := len(Input) / Channels
	OutputFrames := int(float64(InputFrames) / Ratio)
	Output := make([]int16, OutputFrames*Channels)

	for OutFrame := 0; OutFrame < OutputFrames; OutFrame++ {

		InPos := float64(OutFrame) * Ratio
		InFrame := int(InPos)
		Fraction := InPos - float64(InFrame)

		if InFrame >= InputFrames-1 {
			InFrame = InputFrames - 2
			Fraction = 1.0
		}

		for Ch := 0; Ch < Channels; Ch++ {

			Sample1 := float64(Input[InFrame*Channels+Ch])
			Sample2 := float64(Input[(InFrame+1)*Channels+Ch])
			Interpolated := Sample1 + (Sample2-Sample1)*Fraction
			Output[OutFrame*Channels+Ch] = int16(Interpolated)

		}

	}

	return Output

}