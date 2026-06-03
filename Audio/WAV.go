//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

type wavFormat struct {

	SampleRate int
	Channels int
	BitsPerSample int

}

func (streamer *MP4Streamer) StreamWAVFromURL(ctx context.Context, url string) error {

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {

		return fmt.Errorf("failed to create WAV request: %w", err)

	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {

		return fmt.Errorf("failed to fetch WAV stream: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {

		return fmt.Errorf("HTTP %d", resp.StatusCode)

	}

	format, dataReader, err := openWAVStream(resp.Body)

	if err != nil {

		return err

	}

	if format.BitsPerSample != 16 {

		return fmt.Errorf("unsupported WAV bit depth: %d", format.BitsPerSample)

	}

	samplesPerFrame := FrameSize * Channels
	sendFrame := func(frame []int16) bool {

		if streamer.IsStopped() {

			return false

		}

		defer func() { recover() }()

		streamer.PCMFrameChan <- frame

		atomic.AddInt64(&streamer.Progress, 20)
		atomic.AddInt64(&streamer.BytesStreamed, int64(len(frame)*2))
		atomic.AddInt64(&streamer.FramesEmitted, 1)

		return true

	}

	pcmBuffer := make([]int16, 0, samplesPerFrame*8)
	readBuf := make([]byte, 16384)

	for {

		if streamer.IsStopped() {

			break

		}

		n, readErr := dataReader.Read(readBuf)

		if n > 0 {

			sampleCount := n / 2
			chunk := make([]int16, sampleCount)

			for i := 0; i < sampleCount; i++ {

				chunk[i] = int16(binary.LittleEndian.Uint16(readBuf[i*2:]))

			}

			if format.Channels == 1 {

				stereo := make([]int16, len(chunk)*2)

				for i, sample := range chunk {

					stereo[i*2] = sample
					stereo[i*2+1] = sample

				}

				chunk = stereo

			} else if format.Channels > 2 {

				stereo := make([]int16, (len(chunk)/format.Channels)*2)

				for i := 0; i < len(stereo)/2; i++ {

					stereo[i*2] = chunk[i*format.Channels]
					stereo[i*2 + 1] = chunk[i*format.Channels+1]

				}

				chunk = stereo

			}

			if format.SampleRate != SampleRate {
				chunk = ResamplePCM(chunk, format.SampleRate, SampleRate, Channels)
			}

			pcmBuffer = append(pcmBuffer, chunk...)

			if err := streamer.drainPCMBuffer(sendFrame, &pcmBuffer, samplesPerFrame); err != nil {
				return err
			}

		}

		if readErr != nil {

			if readErr == io.EOF {

				break

			}

			if ctx.Err() != nil {

				return nil

			}

			return fmt.Errorf("WAV read error: %w", readErr)

		}

	}

	if !streamer.IsStopped() && len(pcmBuffer) > 0 {

		padding := make([]int16, samplesPerFrame-len(pcmBuffer))
		pcmBuffer = append(pcmBuffer, padding...)

		sendFrame(pcmBuffer)

	}

	return nil

}

func openWAVStream(body io.Reader) (wavFormat, io.Reader, error) {

	var riff [12]byte

	if _, err := io.ReadFull(body, riff[:]); err != nil {

		return wavFormat{}, nil, fmt.Errorf("invalid WAV header: %w", err)

	}

	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {

		return wavFormat{}, nil, fmt.Errorf("not a RIFF WAVE file")

	}

	var format wavFormat
	var pcmData []byte

	for {

		var chunkHeader [8]byte

		if _, err := io.ReadFull(body, chunkHeader[:]); err != nil {

			if err == io.EOF {
				break
			}

			return wavFormat{}, nil, fmt.Errorf("WAV chunk header: %w", err)

		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {

		case "fmt ":

			if chunkSize < 16 {
				return wavFormat{}, nil, fmt.Errorf("WAV fmt chunk too small")
			}

			fmtData := make([]byte, chunkSize)

			if _, err := io.ReadFull(body, fmtData); err != nil {

				return wavFormat{}, nil, err

			}

			format.Channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			format.SampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			format.BitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))

		case "data":

			pcmData = make([]byte, chunkSize)

			if _, err := io.ReadFull(body, pcmData); err != nil {

				return wavFormat{}, nil, err

			}

		default:

			if _, err := io.CopyN(io.Discard, body, int64(chunkSize)); err != nil {

				return wavFormat{}, nil, err

			}

		}

		if len(pcmData) > 0 {

			break

		}

	}

	if format.SampleRate == 0 || len(pcmData) == 0 {

		return wavFormat{}, nil, fmt.Errorf("WAV missing fmt or data chunk")

	}

	return format, &wavDataReader{data: pcmData, pos: 0}, nil

}

type wavDataReader struct {

	data []byte
	pos int

}

func (reader *wavDataReader) Read(buf []byte) (int, error) {

	if reader.pos >= len(reader.data) {

		return 0, io.EOF

	}

	n := copy(buf, reader.data[reader.pos:])
	reader.pos += n

	return n, nil

}

// wavMetadataFromHeader reads fmt/data chunk sizes from the start of a WAV file.
func wavMetadataFromHeader(data []byte) (wavFormat, int64) {

	var format wavFormat
	var dataSize int64

	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {

		return format, 0

	}

	offset := 12

	for offset+8 <= len(data) {

		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))

		payloadStart := offset + 8
		payloadEnd := payloadStart + chunkSize

		if payloadEnd > len(data) {

			break

		}

		switch chunkID {

		case "fmt ":

			if chunkSize >= 16 {

				format.Channels = int(binary.LittleEndian.Uint16(data[payloadStart+2 : payloadStart+4]))
				format.SampleRate = int(binary.LittleEndian.Uint32(data[payloadStart+4 : payloadStart+8]))
				format.BitsPerSample = int(binary.LittleEndian.Uint16(data[payloadStart+14 : payloadStart+16]))

			}

		case "data":

			dataSize = int64(chunkSize)

		}

		if dataSize > 0 && format.SampleRate > 0 {

			break

		}

		offset = payloadEnd

		if chunkSize % 2 == 1 {

			offset++

		}

	}

	return format, dataSize

}

func isWAVContentType(contentType string) bool {

	lower := strings.ToLower(contentType)

	return strings.Contains(lower, "audio/wav") || strings.Contains(lower, "audio/wave") || strings.Contains(lower, "audio/x-wav")

}
