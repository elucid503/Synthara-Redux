//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mp3 "github.com/hajimehoshi/go-mp3"

	"Synthara-Redux/Utils"
)

// MP4Streamer decodes remote MP3, WAV, or MP4/AAC into PCM frames for the mixer.
type MP4Streamer struct {

	Paused atomic.Bool
	Stopped atomic.Bool

	Progress int64
	BytesStreamed int64
	FramesEmitted int64

	PCMFrameChan chan []int16 // PCMFrameChan carries raw 20ms stereo PCM frames

	CancelFunc context.CancelFunc

	Mutex sync.Mutex

}

// NewMP4Streamer creates a streamer for direct media and remote audio URLs.
func NewMP4Streamer() (*MP4Streamer, error) {

	streamer := &MP4Streamer{

		PCMFrameChan: make(chan []int16, 100),

	}

	streamer.Paused.Store(false)
	streamer.Stopped.Store(false)

	return streamer, nil
}

func (S *MP4Streamer) Pause() {

	S.Paused.Store(true)

}

func (S *MP4Streamer) Resume() {

	S.Paused.Store(false)

}

func (S *MP4Streamer) IsPaused() bool {

	return S.Paused.Load()

}

func (S *MP4Streamer) IsStopped() bool {

	return S.Stopped.Load()

}

// NextPCMFrame returns one buffered PCM frame without blocking. It returns (nil, nil) while paused or if no frames are available, and (nil, io.EOF) if the stream has ended.
func (S *MP4Streamer) NextPCMFrame() ([]int16, error) {

	if S.IsPaused() {

		return nil, nil

	}

	select {

	case Frame, OK := <-S.PCMFrameChan:

		if !OK {

			return nil, io.EOF

		}

		return Frame, nil

	default:

		return nil, nil

	}

}

func (S *MP4Streamer) Stop() {

	if S.Stopped.Swap(true) {

		return // Already stopped

	}

	S.Mutex.Lock()
	defer S.Mutex.Unlock()

	if S.CancelFunc != nil {

		S.CancelFunc()
		S.CancelFunc = nil

	}

	// Safely close PCMFrameChan

	defer func() {

		if r := recover(); r != nil {

			// Channel already closed

		}

	}()

	close(S.PCMFrameChan)

}

// StreamFromURL fetches audio from a URL, auto-detecting MP3, WAV, or MP4/AAC.
func (streamer *MP4Streamer) StreamFromURL(ctx context.Context, url string) error {

	lowerURL := strings.ToLower(url)

	if strings.HasSuffix(lowerURL, ".mp3") {

		return streamer.streamWithFrameCheck(streamer.StreamMP3FromURL(ctx, url))

	}

	if strings.HasSuffix(lowerURL, ".wav") {

		return streamer.streamWithFrameCheck(streamer.StreamWAVFromURL(ctx, url))

	}

	headReq, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)

	if err != nil {

		return fmt.Errorf("failed to create probe request: %w", err)

	}

	probeClient := &http.Client{Timeout: 10 * time.Second}
	headResp, err := probeClient.Do(headReq)

	if err == nil {

		headResp.Body.Close()
		contentType := headResp.Header.Get("Content-Type")

		if strings.Contains(contentType, "audio/mpeg") {

			return streamer.streamWithFrameCheck(streamer.StreamMP3FromURL(ctx, url))

		}

		if isWAVContentType(contentType) {

			return streamer.streamWithFrameCheck(streamer.StreamWAVFromURL(ctx, url))

		}

	}

	client := &http.Client{Timeout: 60 * time.Second}

	info, err := ProbeMP4Stream(ctx, url, client)

	if err != nil {

		return err

	}

	err = streamer.streamMdatChunked(ctx, url, info, client)

	return streamer.streamWithFrameCheck(err)

}

func (streamer *MP4Streamer) streamWithFrameCheck(err error) error {

	if err != nil {
		return err
	}

	if atomic.LoadInt64(&streamer.FramesEmitted) == 0 {
		return errors.New("no audio frames decoded from media URL")
	}

	return nil

}

// StreamMP3FromURL streams a raw MP3, decodes it to PCM, and encodes to Opus frames. The go-mp3 decoder handles ID3 tags and all MPEG layer 3 variants natively.
func (S *MP4Streamer) StreamMP3FromURL(Ctx context.Context, URL string) error {

	Req, Err := http.NewRequestWithContext(Ctx, "GET", URL, nil)

	if Err != nil {

		return fmt.Errorf("failed to create request: %w", Err)

	}

	// No overall timeout since streaming can be long

	StreamClient := &http.Client{}
	Resp, Err := StreamClient.Do(Req)

	if Err != nil {

		return fmt.Errorf("failed to fetch MP3 stream: %w", Err)

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != http.StatusOK && Resp.StatusCode != http.StatusPartialContent {

		return fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	MP3Dec, Err := mp3.NewDecoder(Resp.Body)

	if Err != nil {

		return fmt.Errorf("failed to initialize MP3 decoder: %w", Err)

	}

	SourceRate := MP3Dec.SampleRate()
	SamplesPerOpusFrame := FrameSize * Channels // 960 * 2 = 1920 int16 values

	SafeSend := func(Frame []int16) bool {

		if S.IsStopped() {

			return false

		}

		defer func() { recover() }() // In case of send on closed channel

		S.PCMFrameChan <- Frame

		atomic.AddInt64(&S.Progress, 20)
		atomic.AddInt64(&S.BytesStreamed, int64(len(Frame)*2))
		atomic.AddInt64(&S.FramesEmitted, 1)

		return true

	}

	PCMBuffer := make([]int16, 0, SamplesPerOpusFrame*8)
	ReadBuf := make([]byte, 16384)

	for {

		if S.IsStopped() {

			break

		}

		N, ReadErr := MP3Dec.Read(ReadBuf)

		if N > 0 {

			// Convert little-endian bytes to int16 samples

			SampleCount := N / 2
			PCMChunk := make([]int16, SampleCount)

			for i := 0; i < SampleCount; i++ {

				PCMChunk[i] = int16(uint16(ReadBuf[i*2]) | uint16(ReadBuf[i*2+1])<<8)

			}

			// Resamples to 48kHz (Qobuz MP3 is 44100Hz, Discord requires 48000Hz)

			if SourceRate != SampleRate {

				PCMChunk = ResamplePCM(PCMChunk, SourceRate, SampleRate, Channels)

			}

			PCMBuffer = append(PCMBuffer, PCMChunk...)

			if Err := S.drainPCMBuffer(SafeSend, &PCMBuffer, SamplesPerOpusFrame); Err != nil {

				return Err

			}

		}

		if ReadErr != nil {

			if ReadErr == io.EOF {

				break

			}

			if Ctx.Err() != nil {

				return nil // context cancelled

			}

			return fmt.Errorf("MP3 decode error: %w", ReadErr)

		}

	}

	if !S.IsStopped() && len(PCMBuffer) > 0 {

		Padding := make([]int16, SamplesPerOpusFrame-len(PCMBuffer))
		PCMBuffer = append(PCMBuffer, Padding...)

		SafeSend(PCMBuffer)

	}

	return nil

}


// streamMdatChunked fetches AAC access units by file offset (supports video+audio MP4).
func (S *MP4Streamer) streamMdatChunked(ctx context.Context, url string, info *MP4AudioInfo, client *http.Client) error {

	decoder, err := NewRawAACDecoder(info.ASC)

	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	defer decoder.Close()

	sendFrame := func(frame []int16) bool {

		if S.IsStopped() {

			return false

		}

		defer func() { recover() }()

		S.PCMFrameChan <- frame

		atomic.AddInt64(&S.Progress, 20)
		atomic.AddInt64(&S.BytesStreamed, int64(len(frame)*2))
		atomic.AddInt64(&S.FramesEmitted, 1)

		return true

	}

	samplesPerFrame := FrameSize * Channels
	pcmBuffer := make([]int16, 0, samplesPerFrame*4)

	sampleIdx := 0
	decodeFailures := 0

	for sampleIdx < len(info.Samples) {

		if S.IsStopped() {

			break

		}

		rangeStart := info.Samples[sampleIdx].Offset
		rangeEnd := rangeStart + int64(info.Samples[sampleIdx].Size)
		batchEnd := sampleIdx + 1

		for batchEnd < len(info.Samples) {

			next := info.Samples[batchEnd]

			if next.Offset != rangeEnd {

				break

			}

			rangeEnd += int64(next.Size)
			batchEnd++

		}

		chunkData, fetchErr := httpRange(ctx, url, client, rangeStart, rangeEnd-1)

		if fetchErr != nil {

			return fmt.Errorf("failed to fetch AAC chunk: %w", fetchErr)

		}

		dataOffset := int64(0)

		for i := sampleIdx; i < batchEnd; i++ {

			if S.IsStopped() {

				return nil

			}

			sample := info.Samples[i]
			size := int64(sample.Size)

			if dataOffset+size > int64(len(chunkData)) {

				break

			}

			aacFrame := stripMp4AACPayload(chunkData[dataOffset : dataOffset+size])
			dataOffset += size

			pcm, decodeErr := decoder.DecodeFrame(aacFrame)

			if decodeErr != nil || len(pcm) == 0 {

				decodeFailures++
				continue

			}

			pcmBuffer = append(pcmBuffer, pcm...)

			if drainErr := S.drainPCMBuffer(sendFrame, &pcmBuffer, samplesPerFrame); drainErr != nil {

				return drainErr

			}

		}

		sampleIdx = batchEnd

	}

	if decodeFailures > 0 {
		Utils.Logger.Warn("Streaming", fmt.Sprintf("MP4 AAC: %d/%d samples failed to decode", decodeFailures, len(info.Samples)))
	}

	if !S.IsStopped() && len(pcmBuffer) > 0 {

		padding := make([]int16, samplesPerFrame-len(pcmBuffer))
		pcmBuffer = append(pcmBuffer, padding...)

		sendFrame(pcmBuffer)

	}

	return nil

}

// MP4Playback wraps the streamer for playback control
type MP4Playback struct {

	Streamer *MP4Streamer
	Volume *VolumeProcessor
	Effects *EffectsProcessor

	Stopped atomic.Bool

}

func (P *MP4Playback) Pause() {

	if P.Streamer != nil {

		P.Streamer.Pause()

	}

}

func (P *MP4Playback) Resume() {

	if P.Streamer != nil {

		P.Streamer.Resume()

	}

}

func (P *MP4Playback) Stop() {

	if P.Stopped.Swap(true) {

		return

	}

	if P.Streamer != nil {

		P.Streamer.Stop()

	}

}

// MP4PCMProvider adapts the streamer's PCM buffer to the mixer's PCMFrameProvider interface.
type MP4PCMProvider struct {

	Streamer *MP4Streamer

}

func (P *MP4PCMProvider) ProvidePCMFrame() ([]int16, error) {

	return P.Streamer.NextPCMFrame()

}

func (P *MP4PCMProvider) Close() {

	P.Streamer.Stop()

}

// PlayMP4 starts playback of an MP4 stream from a URL
func PlayMP4(URL string, OnFinished func(), SendToWS func(Event string, Data any), OnStreamingError func()) (*MP4Playback, error) {

	Streamer, Err := NewMP4Streamer()

	if Err != nil {

		return nil, Err

	}

	Ctx, CancelFunc := context.WithCancel(context.Background())
	Streamer.CancelFunc = CancelFunc

	Playback := &MP4Playback{

		Streamer: Streamer,

	}

	// Fetch and process in background
	go func() {

		Err := Streamer.StreamFromURL(Ctx, URL)

		if Err != nil {

			Utils.Logger.Error("Streaming", fmt.Sprintf("Streaming error: %s", Err.Error()))

			Playback.Stop()

			if OnStreamingError != nil {

				OnStreamingError()

			}

			return

		}

		// Waits for all frames to be consumed

		Streamer.Mutex.Lock()

		func() {

			defer func() {

				if r := recover(); r != nil {

					// noop, channel already closed

				}

			}()

			close(Streamer.PCMFrameChan)
		}()

		ShouldCallFinished := !Playback.Stopped.Load()

		Streamer.Mutex.Unlock()

	DrainPCM: // label for draining loop

		for len(Streamer.PCMFrameChan) > 0 {

			select {

			case <-Ctx.Done():

				break DrainPCM

			case <-time.After(25 * time.Millisecond):

			}

		}

		if ShouldCallFinished {

			if OnFinished != nil {

				OnFinished()

			}

		}

	}()

	return Playback, nil

}
