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

// RawAACDecoder handles decoding raw AAC frames (without ADTS headers)
type RawAACDecoder struct {
	
	Decoder     C.HANDLE_AACDECODER

	SampleRate  int
	NumChannels int

}

const (

	SampleRate      = 48000
	Channels        = 2
	FrameSize       = 960 // 20ms at 48kHz
	MaxPacketSize   = 4000
	AudioBufferSize = FrameSize * Channels * 2 // 16-bit samples

)

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

	// Resamples if needed (source audio may be different rate, Discord needs 48kHz)

	if SampleRate != 48000 {

		PCMData = ResamplePCM(PCMData, SampleRate, 48000, NumChannels)

	}

	return PCMData, nil

}

// DecodeRaw decodes raw AAC data (without ADTS headers) from MP4 container
// Requires AudioSpecificConfig for initialization
func (Decoder *FDKAACDecoder) DecodeRaw(RawAAC []byte, ASC []byte) ([]int16, error) {

	if len(RawAAC) == 0 {

		return nil, errors.New("empty raw AAC data")

	}

	// Create a new decoder for raw AAC

	RawDecoder := C.aacDecoder_Open(C.TT_MP4_RAW, 1)

	if RawDecoder == nil {

		return nil, errors.New("failed to open raw AAC decoder")

	}

	defer C.aacDecoder_Close(RawDecoder)

	// If we have ASC, configure the decoder with it

	if len(ASC) > 0 {

		ASCBuffer := (*C.uchar)(C.CBytes(ASC))
		ASCSize := C.uint(len(ASC))
		
		ErrorCode := C.aacDecoder_ConfigRaw(RawDecoder, &ASCBuffer, &ASCSize)
		C.free(unsafe.Pointer(ASCBuffer))
		
		if ErrorCode != C.AAC_DEC_OK {

			// Try without ASC config

		}

	}

	var AllPCM []int16
	
	// Raw AAC frames in MP4 need to be fed one at a time
		
	InputBuffer := (*C.uchar)(C.CBytes(RawAAC))
	InputSize := C.uint(len(RawAAC))
	BytesValid := InputSize

	ErrorCode := C.aacDecoder_Fill(RawDecoder, &InputBuffer, &InputSize, &BytesValid)
	C.free(unsafe.Pointer(InputBuffer))

	if ErrorCode == C.AAC_DEC_OK {

		// Try to decode all available frames

		for {

			OutputBuffer := make([]C.short, 8192)
			ErrorCode = C.aacDecoder_DecodeFrame(RawDecoder, &OutputBuffer[0], C.int(len(OutputBuffer)), 0)

			if ErrorCode == C.AAC_DEC_NOT_ENOUGH_BITS {

				break

			}
			
			if ErrorCode != C.AAC_DEC_OK {

				break

			}

			StreamInfo := C.aacDecoder_GetStreamInfo(RawDecoder)

			if StreamInfo == nil {

				continue
				
			}

			DecodedChannels := int(StreamInfo.numChannels)
			DecodedSampleRate := int(StreamInfo.sampleRate)
			FrameSize := int(StreamInfo.frameSize)
			OutputSize := DecodedChannels * FrameSize

			if OutputSize <= 0 || OutputSize > len(OutputBuffer) {

				continue

			}

			PCMData := make([]int16, OutputSize)

			for i := 0; i < OutputSize; i++ {

				PCMData[i] = int16(OutputBuffer[i])

			}

			// Resamples to 48kHz if needed

			if DecodedSampleRate != 48000 && DecodedSampleRate > 0 {

				PCMData = ResamplePCM(PCMData, DecodedSampleRate, 48000, DecodedChannels)

			}

			// Convert mono to stereo if needed

			if DecodedChannels == 1 {

				StereoPCM := make([]int16, len(PCMData)*2)

				for i, Sample := range PCMData {

					StereoPCM[i*2] = Sample
					StereoPCM[i*2+1] = Sample

				}

				PCMData = StereoPCM

			}

			AllPCM = append(AllPCM, PCMData...)

		}

	}

	if len(AllPCM) == 0 {

		return nil, errors.New("no PCM data decoded from raw AAC")

	}

	return AllPCM, nil

}

// NewRawAACDecoder creates a decoder for raw AAC frames using AudioSpecificConfig
func NewRawAACDecoder(ASC []byte) (*RawAACDecoder, error) {
	
	Decoder := C.aacDecoder_Open(C.TT_MP4_RAW, 1)

	if Decoder == nil {

		return nil, errors.New("failed to open raw AAC decoder")

	}
	
	RawDecoder := &RawAACDecoder{

		Decoder:     Decoder,

		SampleRate:  44100,
		NumChannels: 2,

	}
	
	// Configure with AudioSpecificConfig if provided

	if len(ASC) > 0 {

		ASCBuffer := (*C.uchar)(C.CBytes(ASC))
		ASCSize := C.uint(len(ASC))
		
		ErrorCode := C.aacDecoder_ConfigRaw(Decoder, &ASCBuffer, &ASCSize)
		C.free(unsafe.Pointer(ASCBuffer))
		
		if ErrorCode != C.AAC_DEC_OK {

			C.aacDecoder_Close(Decoder)
			return nil, errors.New("failed to configure decoder with ASC")

		}
		
	}
	
	return RawDecoder, nil
}

// DecodeFrame decodes a single raw AAC access unit
func (D *RawAACDecoder) DecodeFrame(Frame []byte) ([]int16, error) {
	
	if len(Frame) == 0 {
		return nil, errors.New("empty frame")
	}
	
	// Fill decoder with frame data

	InputBuffer := (*C.uchar)(C.CBytes(Frame))
	InputSize := C.uint(len(Frame))
	BytesValid := InputSize
	
	ErrorCode := C.aacDecoder_Fill(D.Decoder, &InputBuffer, &InputSize, &BytesValid)
	C.free(unsafe.Pointer(InputBuffer))
	
	if ErrorCode != C.AAC_DEC_OK {

		return nil, errors.New("aacDecoder_Fill failed")

	}
	
	// Decode the frame

	OutputBuffer := make([]C.short, 8192)
	ErrorCode = C.aacDecoder_DecodeFrame(D.Decoder, &OutputBuffer[0], C.int(len(OutputBuffer)), 0)
	
	if ErrorCode == C.AAC_DEC_NOT_ENOUGH_BITS {

		return nil, nil // Need more data

	}
	
	if ErrorCode != C.AAC_DEC_OK {

		return nil, errors.New("aacDecoder_DecodeFrame failed")

	}
	
	// Get stream info

	StreamInfo := C.aacDecoder_GetStreamInfo(D.Decoder)

	if StreamInfo == nil {

		return nil, errors.New("failed to get stream info")

	}
	
	DecodedChannels := int(StreamInfo.numChannels)
	DecodedSampleRate := int(StreamInfo.sampleRate)

	FrameSize := int(StreamInfo.frameSize)
	OutputSize := DecodedChannels * FrameSize
	
	if OutputSize <= 0 || OutputSize > len(OutputBuffer) {

		return nil, errors.New("invalid output size")

	}
	
	// Update decoder info

	D.SampleRate = DecodedSampleRate
	D.NumChannels = DecodedChannels
	
	// Convert to int16

	PCMData := make([]int16, OutputSize)

	for i := 0; i < OutputSize; i++ {

		PCMData[i] = int16(OutputBuffer[i])

	}
	
	// Resample to 48kHz if needed (Tidal typically sends 44100Hz)

	if DecodedSampleRate != 48000 && DecodedSampleRate > 0 {

		PCMData = ResamplePCM(PCMData, DecodedSampleRate, 48000, DecodedChannels)

	}
	
	// Convert mono to stereo if needed (Discord requires stereo)

	if DecodedChannels == 1 {

		StereoPCM := make([]int16, len(PCMData)*2)

		for i, Sample := range PCMData {

			StereoPCM[i*2] = Sample
			StereoPCM[i*2+1] = Sample
			
		}

		return StereoPCM, nil

	}
	
	// If stereo but not 2 channels, remix to stereo

	if DecodedChannels > 2 {

		StereoData := make([]int16, (len(PCMData)/DecodedChannels)*2)

		for i := 0; i < len(PCMData)/DecodedChannels; i++ {

			// Simple downmix: average all channels to stereo

			var Left, Right int32

			for ch := 0; ch < DecodedChannels; ch++ {

				Sample := int32(PCMData[i*DecodedChannels+ch])

				if ch%2 == 0 {

					Left += Sample

				} else {

					Right += Sample

				}

			}

			StereoData[i*2] = int16(Left / int32((DecodedChannels+1)/2))
			StereoData[i*2+1] = int16(Right / int32(DecodedChannels/2))

		}

		return StereoData, nil

	}
	
	return PCMData, nil
}

// Close releases decoder resources
func (D *RawAACDecoder) Close() {

	if D.Decoder != nil {

		C.aacDecoder_Close(D.Decoder)
		D.Decoder = nil

	}

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