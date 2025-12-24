//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/asticode/go-astits"
	"github.com/nareix/joy4/codec/aacparser"
	"layeh.com/gopus"
)

const (

	SampleRate      = 48000
	Channels        = 2
	FrameSize       = 960 // 20ms at 48kHz
	MaxPacketSize   = 4000
	AudioBufferSize = FrameSize * Channels * 2 // 16-bit samples

)

type Playback struct {

	Streamer *SegmentStreamer

	Stopped  atomic.Bool

}

func (P *Playback) Pause() {

	if P.Streamer != nil {

		P.Streamer.Pause()

	}

}

func (P *Playback) Resume() {

	if P.Streamer != nil {

		P.Streamer.Resume()

	}

}

func (P *Playback) Stop() {

	if P.Stopped.Swap(true) {

		return

	}

	if P.Streamer != nil {

		P.Streamer.Stop()

	}

}
type AudioProcessor struct {

	AACDecoder  *FDKAACDecoder
	OpusEncoder *gopus.Encoder

}

func NewAudioProcessor() (*AudioProcessor, error) {

	AACDecoder, ErrorCreatingDecoder := NewFDKAACDecoder()

	if ErrorCreatingDecoder != nil {

		return nil, ErrorCreatingDecoder

	}

	OpusEncoder, ErrorCreatingEncoder := gopus.NewEncoder(SampleRate, Channels, gopus.Audio)

	if ErrorCreatingEncoder != nil {

		AACDecoder.Close()
		return nil, ErrorCreatingEncoder

	}

	OpusEncoder.SetBitrate(128000) // 128 kbps

	return &AudioProcessor{

		AACDecoder:  AACDecoder,
		OpusEncoder: OpusEncoder,

	}, nil

}

func (Processor *AudioProcessor) ProcessSegment(SegmentBytes []byte) ([][]byte, error) {

	AACFrames, ErrorExtractingAAC := Processor.ExtractAACFrames(SegmentBytes)

	if ErrorExtractingAAC != nil {

		return nil, ErrorExtractingAAC

	}

	OpusFrames, ErrorEncoding := Processor.EncodeAACToOpus(AACFrames)

	if ErrorEncoding != nil {

		return nil, ErrorEncoding

	}

	return OpusFrames, nil

}

func (Processor *AudioProcessor) ExtractAACFrames(SegmentBytes []byte) ([][]byte, error) {

	Reader := bytes.NewReader(SegmentBytes)
	Demuxer := astits.NewDemuxer(context.Background(), Reader)

	var AACFrames [][]byte
	var AudioPID uint16
	
	AudioPIDFound := false

	for {

		Data, ErrorReadingPacket := Demuxer.NextData()

		if ErrorReadingPacket != nil {

			if ErrorReadingPacket == astits.ErrNoMorePackets || ErrorReadingPacket == io.EOF {

				break

			}

			return nil, ErrorReadingPacket

		}

		if Data.PMT != nil {

			for _, Stream := range Data.PMT.ElementaryStreams {

				if Stream.StreamType == astits.StreamTypeAACAudio || Stream.StreamType == astits.StreamTypeAACLATMAudio {

					AudioPID = Stream.ElementaryPID
					AudioPIDFound = true
					break

				}

			}

		}

		if Data.PES != nil && AudioPIDFound && Data.PID == AudioPID {

			if len(Data.PES.Data) > 0 {

				// Extracts ADTS AAC frames from PES data

				Frames := Processor.ParseADTSFrames(Data.PES.Data)
				AACFrames = append(AACFrames, Frames...)

			}

		}

	}

	if len(AACFrames) == 0 {

		return nil, errors.New("no audio data found in segment")

	}

	return AACFrames, nil

}

func (Processor *AudioProcessor) ParseADTSFrames(ADTSData []byte) [][]byte {

	var Frames [][]byte
	Offset := 0

	for Offset < len(ADTSData) {

		if Offset+7 > len(ADTSData) {

			break

		}

		// Parses ADTS header to get frame length

		_, _, FrameLen, _, ErrorParsing := aacparser.ParseADTSHeader(ADTSData[Offset:])

		if ErrorParsing != nil {

			Offset++
			continue

		}

		if Offset+FrameLen > len(ADTSData) {

			break

		}

		Frame := make([]byte, FrameLen)
		copy(Frame, ADTSData[Offset:Offset+FrameLen])
		Frames = append(Frames, Frame)

		Offset += FrameLen

	}

	return Frames

}

func (Processor *AudioProcessor) EncodeAACToOpus(AACFrames [][]byte) ([][]byte, error) {

	var OpusFrames [][]byte
	SamplesPerFrame := FrameSize * Channels
	
	PCMBuffer := make([]int16, 0, SamplesPerFrame*4) // Preallocate reasonable buffer

	for _, Frame := range AACFrames {

		PCMData, ErrorDecoding := Processor.AACDecoder.Decode(Frame)

		if ErrorDecoding != nil { 

			continue // Skip bad frames

		}

		if PCMData == nil {
			
			continue // No data decoded

		}

		PCMBuffer = append(PCMBuffer, PCMData...)

		// Encodes complete frames immediately

		for len(PCMBuffer) >= SamplesPerFrame {

			PCMFrame := PCMBuffer[:SamplesPerFrame]

			OpusData, ErrorEncoding := Processor.OpusEncoder.Encode(PCMFrame, FrameSize, MaxPacketSize)

			if ErrorEncoding != nil {

				return nil, ErrorEncoding

			}

			OpusFrames = append(OpusFrames, OpusData)
			
			PCMBuffer = PCMBuffer[SamplesPerFrame:] // Removes already-processed samples

		}

	}

	// Handles remaining samples with padding if needed

	if len(PCMBuffer) > 0 {

		Padding := make([]int16, SamplesPerFrame-len(PCMBuffer))
		PCMBuffer = append(PCMBuffer, Padding...)

		OpusData, ErrorEncoding := Processor.OpusEncoder.Encode(PCMBuffer, FrameSize, MaxPacketSize)

		if ErrorEncoding != nil {

			return nil, ErrorEncoding

		}

		OpusFrames = append(OpusFrames, OpusData)

	}

	return OpusFrames, nil

}

func (Processor *AudioProcessor) Close() {

	if Processor.AACDecoder != nil {

		Processor.AACDecoder.Close()

	}

}