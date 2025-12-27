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
	Progress int64

	OpusFrameChan chan []byte
	SeekChan      chan int
	DoneChan      chan struct{}

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

		Progress: 0,

		OpusFrameChan: make(chan []byte, 50),
		SeekChan:      make(chan int, 1),
		DoneChan:      make(chan struct{}),

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

func (Streamer *SegmentStreamer) ProcessNextSegment(SegmentBytes []byte) (int, bool) {

	if Streamer.IsStopped() {

		return 0, false

	}

	OpusFrames, ErrorProcessing := Streamer.Processor.ProcessSegment(SegmentBytes)

	if ErrorProcessing != nil {

		Utils.Logger.Error(fmt.Sprintf("Error processing segment %d/%d: %s", Streamer.CurrentIndex, Streamer.TotalSegments, ErrorProcessing.Error()))
		return 0, false

	}

	for _, Frame := range OpusFrames {

		if Streamer.IsStopped() {

			return 0, false

		}

		select {

			case <-Streamer.DoneChan:
				return 0, false

			case Index := <-Streamer.SeekChan:

				return Index, true

			default:

		}

		select {

			case <-Streamer.DoneChan:
				return 0, false

			case Index := <-Streamer.SeekChan:

				return Index, true

			case Streamer.OpusFrameChan <- Frame: // Blocking send
				
		}

	}

	Streamer.Mutex.Lock()
	Streamer.CurrentIndex++
	Streamer.Mutex.Unlock()

	return 0, false

}

func (Streamer *SegmentStreamer) GetNextFrame() ([]byte, bool) {

	if Streamer.IsPaused() {

		return nil, true

	}

	select {
	case <-Streamer.DoneChan:
		return nil, false
	case Frame, OK := <-Streamer.OpusFrameChan:
		if !OK {

			return nil, false // channel closed
	
		}
	
		// Each frame is 20ms, so we increment progress by 20
	
		Streamer.Mutex.Lock()
		Streamer.Progress += 20
		Streamer.Mutex.Unlock()
	
		return Frame, true
	}

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

			// Channel already closed, nop

		}

	}()

	close(Streamer.DoneChan)

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

		Streamer:        Streamer,
		Segments:        Segments,
		SegmentDuration: SegmentDuration,
		Stopped:         atomic.Bool{},

	}

	Playback.Stopped.Store(false)

	// Fetch and process segments in background

	go func() {

		for Index := 0; Index < len(Segments); {

			select {

				case NewIndex := <-Streamer.SeekChan:

					Index = NewIndex
					continue

				default:
					
			}

			if Playback.Stopped.Load() {

				break // Stopped

			}

			if Index >= len(Segments) {

				break // All segments processed

			}

			Segment := Segments[Index]

			SegmentBytes, ErrorFetching := InnertubeFuncs.GetHLSSegment(Segment.URI, &InnertubeFuncs.HLSOptions{})

			if ErrorFetching != nil {

				Utils.Logger.Error("Error fetching segment: " + ErrorFetching.Error())

				Index++
				continue

			}

			NewIndex, Seeked := Streamer.ProcessNextSegment(SegmentBytes)
			SegmentBytes = nil

			if Seeked { // go to new index

				Index = NewIndex
				continue

			}

			Index++

		}

		// Close channel and trigger finished

		Streamer.Mutex.Lock()

		close(Streamer.OpusFrameChan)

		if !Playback.Stopped.Load() {

			if OnFinished != nil {

				OnFinished()

			}

		}

		Streamer.Mutex.Unlock()

	}()

	return Playback, nil

}