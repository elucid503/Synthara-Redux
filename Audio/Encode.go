//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

// drainPCMBuffer emits complete 20ms frames from the decode buffer into PCMFrameChan.
func (S *MP4Streamer) drainPCMBuffer(sendFrame func([]int16) bool, pcmBuffer *[]int16, frameSamples int) error {

	for len(*pcmBuffer) >= frameSamples {

		if S.IsStopped() {

			return nil

		}

		frame := make([]int16, frameSamples)
		copy(frame, (*pcmBuffer)[:frameSamples])
		*pcmBuffer = (*pcmBuffer)[frameSamples:]

		if !sendFrame(frame) {

			return nil

		}

	}

	return nil

}
