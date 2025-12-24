package Audio

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"Synthara-Redux/Utils"

	InnertubeFuncs "github.com/elucid503/Overture-Play/Public"
	InnertubeStructs "github.com/elucid503/Overture-Play/Structs"
)

type SegmentStreamer struct {

	Processor *AudioProcessor

	CurrentIndex  int
	TotalSegments int

	Paused  atomic.Bool
	Stopped atomic.Bool

	SegmentDuration int

	OpusFrameChan chan []byte

	Mutex sync.Mutex

}

func NewSegmentStreamer(SegmentDuration int, TotalSegments int) (*SegmentStreamer, error) {

	Processor, ErrorCreatingProcessor := NewAudioProcessor()

	if ErrorCreatingProcessor != nil {

		return nil, ErrorCreatingProcessor

	}

	Streamer := &SegmentStreamer{

		Processor:       Processor,

		CurrentIndex:    0,

		TotalSegments:   TotalSegments,
		SegmentDuration: SegmentDuration,

		OpusFrameChan: make(chan []byte, 50),

	}

	Streamer.Paused.Store(false)
	Streamer.Stopped.Store(false)

	return Streamer, nil

}

func (Streamer *SegmentStreamer) Pause() {

	Streamer.Paused.Store(true)

}

func (Streamer *SegmentStreamer) Resume() {

	Streamer.Paused.Store(false)

}

func (Streamer *SegmentStreamer) IsPaused() bool {

	return Streamer.Paused.Load()

}

func (Streamer *SegmentStreamer) IsStopped() bool {

	return Streamer.Stopped.Load()

}

func (Streamer *SegmentStreamer) ProcessNextSegment(SegmentBytes []byte) {

	if Streamer.IsStopped() {

		return

	}

	OpusFrames, ErrorProcessing := Streamer.Processor.ProcessSegment(SegmentBytes)

	if ErrorProcessing != nil {

		Utils.Logger.Error(fmt.Sprintf("Error processing segment %d/%d: %s", Streamer.CurrentIndex, Streamer.TotalSegments, ErrorProcessing.Error()))
		return

	}

	for _, Frame := range OpusFrames {

		if Streamer.IsStopped() {

			return

		}

		Streamer.OpusFrameChan <- Frame // Blocking send

	}

	Streamer.Mutex.Lock()
	Streamer.CurrentIndex++
	Streamer.Mutex.Unlock()

}

func (Streamer *SegmentStreamer) GetNextFrame() ([]byte, bool) {

	if Streamer.IsPaused() {

		return nil, true

	}

	Frame, OK := <-Streamer.OpusFrameChan

	if !OK {

		return nil, false // channel closed

	}

	return Frame, true

}

func (Streamer *SegmentStreamer) Stop() {

	if Streamer.Stopped.Swap(true) {

		return // Already stopped

	}

	Streamer.Mutex.Lock()
	defer Streamer.Mutex.Unlock()

	// Safely close channel if not already closed

	defer func() {

		if r := recover(); r != nil {

			// Channel already closed, ignore panic

		}

	}()

	close(Streamer.OpusFrameChan)

	if Streamer.Processor != nil {

		Streamer.Processor.Close()
		Streamer.Processor = nil

	}

}


// Play starts playback and returns when finished or stopped
func Play(Segments []InnertubeStructs.HLSSegment, SegmentDuration int, OnFinished func()) (*Playback, error) {

	if len(Segments) == 0 {

		return nil, errors.New("no audio segments available")

	}

	Streamer, ErrorCreatingStreamer := NewSegmentStreamer(SegmentDuration, len(Segments))

	if ErrorCreatingStreamer != nil {

		return nil, ErrorCreatingStreamer

	}

	Playback := &Playback{

		Streamer: Streamer,
		Stopped: atomic.Bool{},

	}

	Playback.Stopped.Store(false)

	// Fetch and process segments in background

	go func() {

		for Index := 0; Index < len(Segments); Index++ {

			if Playback.Stopped.Load() {

				break // Stopped

			}

			Segment := Segments[Index]

			SegmentBytes, ErrorFetching := InnertubeFuncs.GetHLSSegment(Segment.URI, &InnertubeFuncs.HLSOptions{})

			if ErrorFetching != nil {

				Utils.Logger.Error("Error fetching segment: " + ErrorFetching.Error())
				continue

			}

			Streamer.ProcessNextSegment(SegmentBytes)
			SegmentBytes = nil

		}

		// Close channel and trigger finished

		Streamer.Mutex.Lock()

		if !Playback.Stopped.Load() {

			close(Streamer.OpusFrameChan)

			if OnFinished != nil {

				OnFinished()

			}

		}

		Streamer.Mutex.Unlock()

	}()

	return Playback, nil

}