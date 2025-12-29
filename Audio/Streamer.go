package Audio

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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

	ProgressTicker   *time.Ticker
	ProgressStopChan chan struct{}

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
		ProgressTicker:   nil,
		ProgressStopChan: nil,

		OpusFrameChan: make(chan []byte, 50),
		SeekChan:      make(chan int, 1),
		DoneChan:      make(chan struct{}),

	}

	Streamer.Paused.Store(false)
	Streamer.Stopped.Store(false)

	Streamer.startProgressTicker()

	return Streamer, nil

}

func (Streamer *SegmentStreamer) Pause() {

	Streamer.Paused.Store(true)

}

func (Streamer *SegmentStreamer) Resume() {

	Streamer.Paused.Store(false)

}

// startProgressTicker starts the internal progress ticker on a 20ms interval
func (Streamer *SegmentStreamer) startProgressTicker() {

	Streamer.Mutex.Lock()
	defer Streamer.Mutex.Unlock()
	
	if Streamer.ProgressTicker != nil {

		return // already running

	}

	Streamer.ProgressTicker = time.NewTicker(20 * time.Millisecond)
	Streamer.ProgressStopChan = make(chan struct{})

	go func() {

		for {

			if Streamer.IsPaused() {

				time.Sleep(20 * time.Millisecond)
				continue

			}

			if Streamer.IsStopped() {

				return

			}

			select {
				
				case <-Streamer.ProgressTicker.C:

					Streamer.Mutex.Lock()
					Streamer.Progress += 20
					Streamer.Mutex.Unlock()

				case <-Streamer.ProgressStopChan:

					// Stop signal received; return. The ticker is stopped by stopProgressTicker
					// to avoid races where the ticker pointer is cleared concurrently.
					return

			}

		}

	}()

}

// stopProgressTicker stops the internal progress ticker
func (Streamer *SegmentStreamer) stopProgressTicker() {

	Streamer.Mutex.Lock()
	defer Streamer.Mutex.Unlock()

	if Streamer.ProgressTicker != nil && Streamer.ProgressStopChan != nil {

		// Stop the ticker first to release underlying resources, then signal goroutine
		Streamer.ProgressTicker.Stop()
		close(Streamer.ProgressStopChan)
		Streamer.ProgressStopChan = nil
		Streamer.ProgressTicker = nil

	}

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
	
		// Progress is now incremented by internal ticker, not by frame send
	
		return Frame, true
	}

}

func (Streamer *SegmentStreamer) Stop() {
	Streamer.stopProgressTicker()

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