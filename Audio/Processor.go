//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"bytes"
	"context"
	"errors"
	"io"

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

				// Extract ADTS AAC frames from PES data
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

		// Parse ADTS header to get frame length
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

	var AllPCMData []int16

	// Decode all AAC frames to PCM
	for _, Frame := range AACFrames {

		PCMData, ErrorDecoding := Processor.AACDecoder.Decode(Frame)

		if ErrorDecoding != nil {

			continue // Skip bad frames

		}

		if PCMData != nil {

			AllPCMData = append(AllPCMData, PCMData...)

		}

	}

	if len(AllPCMData) == 0 {

		return nil, errors.New("no PCM data decoded")

	}

	// Encode PCM to Opus in chunks
	var OpusFrames [][]byte
	SamplesPerFrame := FrameSize * Channels

	for Offset := 0; Offset < len(AllPCMData); Offset += SamplesPerFrame {

		End := Offset + SamplesPerFrame

		if End > len(AllPCMData) {

			// Pad last frame with silence
			Padding := make([]int16, End-len(AllPCMData))
			AllPCMData = append(AllPCMData, Padding...)

		}

		PCMFrame := AllPCMData[Offset:End]

		// Encode to Opus
		OpusData, ErrorEncoding := Processor.OpusEncoder.Encode(PCMFrame, FrameSize, MaxPacketSize)

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

type SegmentStreamer struct {

	Processor       *AudioProcessor

	CurrentIndex    int
	TotalSegments   int

	SegmentDuration float64

	OpusFrameChan   chan []byte

	ErrorChan       chan error
	StopChan        chan struct{}

}

func NewSegmentStreamer(SegmentDuration float64, TotalSegments int) (*SegmentStreamer, error) {

	Processor, ErrorCreatingProcessor := NewAudioProcessor()

	if ErrorCreatingProcessor != nil {

		return nil, ErrorCreatingProcessor

	}

	return &SegmentStreamer{

		Processor:       Processor,
		CurrentIndex:    0,
		TotalSegments:   TotalSegments,
		SegmentDuration: SegmentDuration,
		OpusFrameChan:   make(chan []byte, 100),
		ErrorChan:       make(chan error, 10),
		StopChan:        make(chan struct{}),

	}, nil

}

func (Streamer *SegmentStreamer) ProcessNextSegment(SegmentBytes []byte) error {

	OpusFrames, ErrorProcessing := Streamer.Processor.ProcessSegment(SegmentBytes)

	if ErrorProcessing != nil {

		return ErrorProcessing

	}

	for _, Frame := range OpusFrames {

		select {

		case Streamer.OpusFrameChan <- Frame:

		case <-Streamer.StopChan:

			return errors.New("stream stopped")

		}

	}

	Streamer.CurrentIndex++

	return nil

}

func (Streamer *SegmentStreamer) GetNextFrame() ([]byte, bool) {

	select {

	case Frame := <-Streamer.OpusFrameChan:

		return Frame, true

	case <-Streamer.StopChan:

		return nil, false

	default:

		return nil, true

	}

}

func (Streamer *SegmentStreamer) ShouldFetchNext() bool {

	return Streamer.CurrentIndex < Streamer.TotalSegments && len(Streamer.OpusFrameChan) < 50 // fetches next if less than 50 frames are buffered; 20ms/frame = 1s buffer

}

func (Streamer *SegmentStreamer) GetProgress() (int, int) {

	return Streamer.CurrentIndex, Streamer.TotalSegments

}

func (Streamer *SegmentStreamer) Stop() {

	close(Streamer.StopChan)
	Streamer.Processor.Close()

}

func (Streamer *SegmentStreamer) Close() {

	Streamer.Stop()
	close(Streamer.OpusFrameChan)
	close(Streamer.ErrorChan)

}