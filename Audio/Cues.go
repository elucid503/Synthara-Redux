//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
)

// CueKind selects a voice-command feedback sound.
type CueKind int

const (

	CueWake CueKind = iota
	CueEnd // 1

)

const (

	cueWakePath = "./Assets/Wake.pcm"
	cueEndPath  = "./Assets/End.pcm"

	playbackDuckGain = 0.15 // music at 15% while capture active

)

// PlaybackDuckGain is the music volume multiplier while voice command capture is active.
func PlaybackDuckGain() float32 {

	return playbackDuckGain

}

// CueDuration returns how long a cue plays at 20ms per frame.
func CueDuration(kind CueKind) time.Duration {

	N := len(cueFrames(kind))

	if N == 0 {

		return 0

	}

	return time.Duration(N) * 20 * time.Millisecond

}

var (
	cueFramesWake [][]int16
	cueFramesEnd  [][]int16

	cueLoadOnce sync.Once

	cueLoadErr error
)

func loadVoiceCues() {

	cueLoadOnce.Do(func() {

		cueFramesWake, cueLoadErr = loadPCMFrames(cueWakePath)
		if cueLoadErr != nil {

			return

		}

		cueFramesEnd, cueLoadErr = loadPCMFrames(cueEndPath)

	})

}

func loadPCMFrames(path string) ([][]int16, error) {

	Data, Err := os.ReadFile(path)

	if Err != nil {

		return nil, fmt.Errorf("read cue %s: %w", path, Err)

	}

	SamplesPerFrame := FrameSize * Channels
	BytesPerFrame := SamplesPerFrame * 2

	if len(Data) < BytesPerFrame {

		return nil, fmt.Errorf("cue %s: too short", path)

	}

	FrameCount := len(Data) / BytesPerFrame
	Frames := make([][]int16, FrameCount)

	for I := 0; I < FrameCount; I++ {

		Off := I * BytesPerFrame
		Chunk := Data[Off : Off+BytesPerFrame]
		Frame := make([]int16, SamplesPerFrame)

		for J := 0; J < SamplesPerFrame; J++ {

			Frame[J] = int16(binary.LittleEndian.Uint16(Chunk[J*2:]))

		}

		Frames[I] = Frame

	}

	return Frames, nil

}

func cueFrames(kind CueKind) [][]int16 {

	loadVoiceCues()

	if cueLoadErr != nil {

		return nil

	}

	switch kind {

	case CueWake:

		return cueFramesWake

	case CueEnd:

		return cueFramesEnd

	default:

		return nil

	}

}
