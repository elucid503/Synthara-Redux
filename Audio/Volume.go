//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"sync/atomic"
)

// VolumeProcessor applies live gain before the single Opus encode.
type VolumeProcessor struct {

	VolumePercent atomic.Int32

}

func NewVolumeProcessor() (*VolumeProcessor, error) {

	processor := &VolumeProcessor{}
	processor.VolumePercent.Store(100)

	return processor, nil

}

func (volume *VolumeProcessor) SetVolume(percent int) {

	if volume == nil {

		return

	}

	volume.VolumePercent.Store(int32(percent))

}

func (volume *VolumeProcessor) VolumeGain() float32 {

	if volume == nil {

		return 1

	}

	return float32(volume.VolumePercent.Load()) / 100.0

}
