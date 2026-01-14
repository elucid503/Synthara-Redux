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

	// Recovers from panics (sends on closed channel)

	defer func() {

		if r := recover(); r != nil {

			// noop; return default values

		}

	}()

	for _, Frame := range OpusFrames {

		if Streamer.IsStopped() {

			return 0, false

		}

		// Blocking send; will block until consumer reads or channel is closed

		Streamer.OpusFrameChan <- Frame
		atomic.AddInt64(&Streamer.Progress, 20) // progress increments in ms per frame (20ms)

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

	Frame, OK := <-Streamer.OpusFrameChan

	if !OK {

		return nil, false // Channel closed

	}

	return Frame, true

}

func (Streamer *SegmentStreamer) Stop() {

	if Streamer.Stopped.Swap(true) {

		return // Already stopped

	}

	Streamer.Mutex.Lock()
	defer Streamer.Mutex.Unlock()

	// Safely closes OpusFrameChan so consumers unblock

	defer func() {

		if r := recover(); r != nil {

			// Channel already closed, nop

		}

	}()

	close(Streamer.OpusFrameChan)

	if Streamer.Processor != nil {

		Streamer.Processor.Close()
		Streamer.Processor = nil

	}

}


// Play starts playback and returns when finished or stopped
func Play(Segments []InnertubeStructs.HLSSegment, SegmentDuration int, OnFinished func(), SendToWS func(Event string, Data any), OnStreamingError func()) (*Playback, error) {

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

	// Fetches and processes segments in background

	go func() {

		const MaxConsecutiveFailures = 3
		ConsecutiveFailures := 0

		for Index := 0; Index < len(Segments); {

			if Playback.Stopped.Load() {

				break // Stopped

			}

			if Index >= len(Segments) {

				break // All segments processed

			}

			Segment := Segments[Index]

			SegmentBytes, ErrorFetching := InnertubeFuncs.GetHLSSegment(Segment.URI, &InnertubeFuncs.HLSOptions{})

			// Checks for HTTP errors or empty body

			Failed := false

			if ErrorFetching != nil {

				Utils.Logger.Error("Error fetching segment: " + ErrorFetching.Error())
				Failed = true

			} else if len(SegmentBytes) == 0 {

				Utils.Logger.Error("Empty segment received")
				Failed = true

			}

			if Failed {

				ConsecutiveFailures++

				if ConsecutiveFailures >= MaxConsecutiveFailures {

					Utils.Logger.Error(fmt.Sprintf("Multiple consecutive segment failures (%d), triggering streaming error", ConsecutiveFailures))

					Playback.Stop()

					if OnStreamingError != nil {

						OnStreamingError()

					}

					break

				}

				Index++
				continue

			}

			// Resets consecutive failures on success
			
			ConsecutiveFailures = 0

			// We should send a progress update to any/all connected websockets here
			// We aren't using the events enum since we cant import structs here :(

			go SendToWS("PROGRESS_UPDATE", map[string]interface{}{

				"Progress": Playback.Streamer.Progress,
				"Index":   Playback.Streamer.CurrentIndex,

			})

			// Time for the processing now

			_, _ = Streamer.ProcessNextSegment(SegmentBytes)
			SegmentBytes = nil

			Index++

		}

		// Close channel and trigger finished

		Streamer.Mutex.Lock()

		defer Streamer.Mutex.Unlock()

		defer func() {

			if r := recover(); r != nil {

				// noop

			}
			
		}()

		close(Streamer.OpusFrameChan)

		if !Playback.Stopped.Load() {

			if OnFinished != nil {

				OnFinished()

			}

		}

	}()

	return Playback, nil

}