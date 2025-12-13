//go:build linux || darwin || windows
// +build linux darwin windows

package Utils

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
	OutputSize := NumChannels * FrameSize

	// Convert C.short to int16

	PCMData := make([]int16, OutputSize)

	for Index := 0; Index < OutputSize; Index++ {

		PCMData[Index] = int16(OutputBuffer[Index])

	}

	return PCMData, nil

}

func (Decoder *FDKAACDecoder) Close() {

	if Decoder.Decoder != nil {

		C.aacDecoder_Close(Decoder.Decoder)
		Decoder.Decoder = nil

	}

}