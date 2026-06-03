//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	mp3 "github.com/hajimehoshi/go-mp3"
)

const (
	mp4HeaderProbeSize = 1 << 20
	mp4TailProbeSize   = 4 << 20
)

// MediaSample is one AAC access unit byte range inside an MP4 file.
type MediaSample struct {

	Offset int64
	Size uint32

}

// MP4AudioInfo holds everything needed to stream AAC from an MP4 URL.
type MP4AudioInfo struct {

	ASC []byte

	SampleRate  int
	NumChannels int

	Samples []MediaSample

	Timescale uint32

	MediaDuration uint64
	DurationSec int

}

// ProbeDurationSec returns playback length in seconds for a direct media URL.
func ProbeDurationSec(mediaURL string) int {

	lower := strings.ToLower(mediaURL)

	switch {

		case strings.HasSuffix(lower, ".mp3"):

			return probeMP3Duration(mediaURL)

		case strings.HasSuffix(lower, ".wav"):

			return probeWAVDuration(mediaURL)

		default:

			info, err := fetchMP4Info(context.Background(), mediaURL, &http.Client{Timeout: 20 * time.Second})

			if err != nil {

				return 0

			}

			return info.DurationSeconds()

	}

}

// ProbeMP4Stream loads MP4 audio sample byte ranges for streaming.
func ProbeMP4Stream(ctx context.Context, mediaURL string, client *http.Client) (*MP4AudioInfo, error) {

	return fetchMP4Info(ctx, mediaURL, client)

}

func probeMP3Duration(url string) int {

	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {

		return 0

	}

	resp, err := client.Do(req)

	if err != nil {

		return 0

	}

	defer resp.Body.Close()

	decoder, err := mp3.NewDecoder(resp.Body)

	if err != nil {

		return 0

	}

	length := decoder.Length()

	if length <= 0 {

		return 0

	}

	rate := decoder.SampleRate()

	if rate <= 0 {

		return 0

	}

	return int(length / int64(rate))

}

func probeWAVDuration(url string) int {

	client := &http.Client{Timeout: 15 * time.Second}
	ctx := context.Background()

	header, err := httpRange(ctx, url, client, 0, 65535)

	if err != nil {

		return 0

	}

	format, dataBytes := wavMetadataFromHeader(header)

	if format.SampleRate <= 0 || dataBytes <= 0 {

		return 0

	}

	bytesPerFrame := format.Channels * format.BitsPerSample / 8

	if bytesPerFrame <= 0 {

		return 0

	}

	return int(dataBytes / int64(bytesPerFrame) / int64(format.SampleRate))

}

func fetchMP4Info(ctx context.Context, url string, client *http.Client) (*MP4AudioInfo, error) {

	if client == nil {

		client = &http.Client{Timeout: 60 * time.Second}

	}

	fileSize, acceptsRanges, err := headContentLength(ctx, url, client)

	if err != nil {

		return nil, err

	}

	headerBytes := int64(mp4HeaderProbeSize)

	if fileSize > 0 && headerBytes > fileSize {

		headerBytes = fileSize

	}

	var headerData []byte

	if headerBytes > 0 {

		headerData, err = httpRange(ctx, url, client, 0, headerBytes-1)

		if err != nil {

			return nil, err

		}

	}

	info := &MP4AudioInfo{SampleRate: 48000, NumChannels: 2}

	walkTopLevelAtoms(headerData, 0, info)

	// moov-at-start files (Twitter/CDN) are fully described in the header probe; skip tail unless needed.
	if !info.hasStreamMetadata() && acceptsRanges {

		var tailData []byte
		var tailBase int64
		var tailErr error

		if fileSize > 0 {

			tailBytes := int64(mp4TailProbeSize)

			if tailBytes > fileSize {

				tailBytes = fileSize

			}

			tailBase = fileSize - tailBytes
			tailData, tailErr = httpRange(ctx, url, client, tailBase, fileSize-1)

		} else {

			tailData, tailBase, tailErr = httpSuffixRange(ctx, url, client, mp4TailProbeSize)

		}

		if tailErr == nil {

			walkTopLevelAtoms(tailData, tailBase, info)

		}

	}

	if !info.hasStreamMetadata() && fileSize > 0 && fileSize <= 16<<20 {

		fullData, fullErr := httpRange(ctx, url, client, 0, fileSize-1)

		if fullErr == nil {

			walkTopLevelAtoms(fullData, 0, info)

		}

	}

	if !info.hasStreamMetadata() {

		return nil, errors.New("failed to parse MP4 audio track metadata")

	}

	info.deriveDuration()

	return info, nil

}

func (info *MP4AudioInfo) hasStreamMetadata() bool {

	return len(info.ASC) > 0 && len(info.Samples) > 0

}

func (info *MP4AudioInfo) DurationSeconds() int {

	if info.DurationSec > 0 {

		return info.DurationSec

	}

	info.deriveDuration()

	return info.DurationSec

}

func (info *MP4AudioInfo) deriveDuration() {

	if info.DurationSec > 0 {
		return
	}

	if info.Timescale > 0 && info.MediaDuration > 0 {

		info.DurationSec = int(float64(info.MediaDuration) / float64(info.Timescale))
		return

	}

	if info.SampleRate > 0 && len(info.Samples) > 0 {

		info.DurationSec = len(info.Samples) * 1024 / info.SampleRate

	}

}

// MP4 atom parsing (audio track only)

type mp4StscEntry struct {

	FirstChunk uint32
	SamplesPerChunk uint32

}

type mp4AudioTrack struct {

	isAudio bool
	hasMp4a bool

	ASC []byte

	SampleRate  int
	NumChannels int

	SampleSizes []uint32
	ChunkOffsets []int64

	Stsc []mp4StscEntry

	Timescale uint32
	MediaDuration uint64

}

func (track *mp4AudioTrack) score() int {

	if !track.isAudio || !track.hasMp4a || len(track.ASC) == 0 {

		return 0

	}

	return len(track.SampleSizes)

}

func (track *mp4AudioTrack) buildSampleTable() []MediaSample {

	if len(track.SampleSizes) == 0 || len(track.ChunkOffsets) == 0 || len(track.Stsc) == 0 {

		return nil

	}

	chunkCount := len(track.ChunkOffsets)
	samplesPerChunk := make([]uint32, chunkCount)

	for chunk := 0; chunk < chunkCount; chunk++ {
		samplesPerChunk[chunk] = samplesPerChunkFor(track.Stsc, uint32(chunk+1))
	}

	samples := make([]MediaSample, 0, len(track.SampleSizes))

	sampleIdx := 0

	for chunk := 0; chunk < chunkCount && sampleIdx < len(track.SampleSizes); chunk++ {

		chunkOffset := track.ChunkOffsets[chunk]
		offsetInChunk := int64(0)

		for s := uint32(0); s < samplesPerChunk[chunk] && sampleIdx < len(track.SampleSizes); s++ {

			size := track.SampleSizes[sampleIdx]

			samples = append(samples, MediaSample{

				Offset: chunkOffset + offsetInChunk,
				Size: size,

			})

			offsetInChunk += int64(size)
			sampleIdx++

		}

	}

	return samples

}

// samplesPerChunkFor returns how many audio samples live in the given 1-based chunk index.
func samplesPerChunkFor(stsc []mp4StscEntry, chunkNum uint32) uint32 {

	if len(stsc) == 0 {

		return 0

	}

	var samples uint32

	for _, entry := range stsc {

		if chunkNum >= entry.FirstChunk {

			samples = entry.SamplesPerChunk

		}

	}

	return samples

}

func walkTopLevelAtoms(data []byte, baseOffset int64, info *MP4AudioInfo) {

	offset := 0

	for offset+8 <= len(data) {

		atomSize, atomType, headerSize, ok := readAtomHeader(data, offset)

		if !ok {

			break

		}

		payloadStart := offset + headerSize
		payloadEnd := offset + atomSize

		if payloadEnd > len(data) {

			break

		}

		if atomType == "moov" {

			mergeAudioTrack(info, parseMoovAudioTrack(data[payloadStart:payloadEnd]))

		}

		offset += atomSize

	}

}

func parseMoovAudioTrack(moov []byte) *mp4AudioTrack {

	best := &mp4AudioTrack{}

	offset := 0

	for offset+8 <= len(moov) {

		atomSize, atomType, headerSize, ok := readAtomHeader(moov, offset)

		if !ok {

			break

		}

		payloadStart := offset + headerSize
		payloadEnd := offset + atomSize

		if payloadEnd > len(moov) {

			break

		}

		if atomType == "trak" {

			candidate := &mp4AudioTrack{}
			parseTrackAtoms(moov[payloadStart:payloadEnd], candidate)

			if candidate.score() > best.score() {

				best = candidate

			}

		}

		offset += atomSize

	}

	if best.score() == 0 {

		return nil

	}

	return best

}

func mergeAudioTrack(info *MP4AudioInfo, track *mp4AudioTrack) {

	if track == nil || track.score() == 0 {
		return
	}

	info.ASC = track.ASC

	if track.SampleRate > 0 {

		info.SampleRate = track.SampleRate

	}

	if track.NumChannels > 0 {

		info.NumChannels = track.NumChannels
	}

	if track.Timescale > 0 {

		info.Timescale = track.Timescale

	}

	if track.MediaDuration > 0 {

		info.MediaDuration = track.MediaDuration

	}

	info.Samples = track.buildSampleTable()

}

func parseTrackAtoms(data []byte, track *mp4AudioTrack) {

	offset := 0

	for offset+8 <= len(data) {

		atomSize, atomType, headerSize, ok := readAtomHeader(data, offset)

		if !ok {

			break

		}

		payloadStart := offset + headerSize
		payloadEnd := offset + atomSize

		if payloadEnd > len(data) {

			break

		}

		payload := data[payloadStart:payloadEnd]

		switch atomType {

			case "mdia", "minf", "stbl":

				parseTrackAtoms(payload, track)

			case "hdlr":

				if len(payload) >= 12 {

					handler := string(payload[8:12])
					track.isAudio = handler == "soun"

				}

			case "stsd":

				if len(payload) > 8 {

					parseTrackAtoms(payload[8:], track)

				}

			case "mp4a":

				track.isAudio = true
				track.hasMp4a = true

				if len(payload) >= 28 {

					track.NumChannels = int(payload[16])<<8 | int(payload[17])
					sampleRateFixed := int(payload[24])<<24 | int(payload[25])<<16 | int(payload[26])<<8 | int(payload[27])

					track.SampleRate = sampleRateFixed >> 16 // converts from 16.16 fixed-point to integer

				}

				if len(payload) > 28 {

					parseTrackAtoms(payload[28:], track)

				}

			case "esds":

				if asc := extractASCFromEsds(payload); len(asc) > 0 {

					track.ASC = asc

				}

			case "mdhd":

				parseMDHD(payload, &track.Timescale, &track.MediaDuration)

			case "stsc":

				parseStsc(payload, track)

			case "stco":

				parseStco32(payload, track)

			case "co64":

				parseCo64(payload, track)

			case "stsz":

				parseStsz(payload, track)

		}

		offset += atomSize

	}

}

func parseStsc(data []byte, track *mp4AudioTrack) {

	if len(data) < 8 {

		return

	}

	entryCount := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])

	for i := 0; i < entryCount; i++ {

		off := 8 + i*12

		if off+12 > len(data) {

			break

		}

		track.Stsc = append(track.Stsc, mp4StscEntry{

			FirstChunk: uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3]),
			SamplesPerChunk: uint32(data[off+4])<<24 | uint32(data[off+5])<<16 | uint32(data[off+6])<<8 | uint32(data[off+7]),

		})

	}

}

func parseStco32(data []byte, track *mp4AudioTrack) {

	if len(data) < 12 {
		return
	}

	entryCount := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])

	for i := 0; i < entryCount; i++ {

		off := 8 + i*4

		if off+4 > len(data) {

			break

		}

		track.ChunkOffsets = append(track.ChunkOffsets, int64(uint32(data[off])<<24|uint32(data[off+1])<<16|uint32(data[off+2])<<8|uint32(data[off+3])))

	}

}

func parseCo64(data []byte, track *mp4AudioTrack) {

	if len(data) < 12 {

		return

	}

	entryCount := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])

	for i := 0; i < entryCount; i++ {

		off := 8 + i*8

		if off+8 > len(data) {

			break

		}

		track.ChunkOffsets = append(track.ChunkOffsets, int64(data[off])<<56 | int64(data[off+1])<<48 | int64(data[off+2])<<40 | int64(data[off+3])<<32 | int64(data[off+4])<<24 | int64(data[off+5])<<16 | int64(data[off+6])<<8 | int64(data[off+7]))

	}

}

func parseStsz(data []byte, track *mp4AudioTrack) {

	if len(data) < 12 {

		return

	}

	defaultSize := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
	sampleCount := int(data[8])<<24 | int(data[9])<<16 | int(data[10])<<8 | int(data[11])

	if defaultSize != 0 {

		track.SampleSizes = make([]uint32, sampleCount)

		for i := range track.SampleSizes {

			track.SampleSizes[i] = uint32(defaultSize)

		}

		return

	}

	track.SampleSizes = make([]uint32, 0, sampleCount)

	for i := 0; i < sampleCount; i++ {

		off := 12 + i*4

		if off+4 > len(data) {

			break

		}

		track.SampleSizes = append(track.SampleSizes, uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3]))

	}

}

func parseMDHD(data []byte, timescale *uint32, duration *uint64) {

	if len(data) < 20 {

		return

	}

	if data[0] == 0 {

		*timescale = uint32(data[12])<<24 | uint32(data[13])<<16 | uint32(data[14])<<8 | uint32(data[15])
		*duration = uint64(uint32(data[16])<<24 | uint32(data[17])<<16 | uint32(data[18])<<8 | uint32(data[19]))

		return

	}

	if len(data) < 32 {

		return

	}

	*timescale = uint32(data[20])<<24 | uint32(data[21])<<16 | uint32(data[22])<<8 | uint32(data[23])
	*duration = uint64(data[24])<<56 | uint64(data[25])<<48 | uint64(data[26])<<40 | uint64(data[27])<<32 | uint64(data[28])<<24 | uint64(data[29])<<16 | uint64(data[30])<<8 | uint64(data[31])

}

func extractASCFromEsds(data []byte) []byte {

	if len(data) < 10 {

		return nil

	}

	pos := 4

	for pos < len(data)-2 {

		tag := data[pos]
		pos++

		length := 0

		for pos < len(data) && (data[pos]&0x80) != 0 {

			length = (length << 7) | int(data[pos]&0x7F)
			pos++

		}

		if pos < len(data) {

			length = (length << 7) | int(data[pos]&0x7F)
			pos++

		}

		if tag == 0x05 {

			if pos+length <= len(data) && length > 0 {

				asc := make([]byte, length)
				copy(asc, data[pos:pos+length])

				return asc

			}

		}

		if tag == 0x03 {

			pos += 3
			continue

		}

		if tag == 0x04 {

			pos += 13
			continue

		}

		pos += length

	}

	return nil

}

// HTTP helpers

func headContentLength(ctx context.Context, url string, client *http.Client) (fileSize int64, acceptsRanges bool, err error) {

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)

	if err != nil {

		return 0, false, err

	}

	resp, err := client.Do(req)

	if err != nil {

		return 0, false, err

	}

	defer resp.Body.Close()

	if resp.ContentLength > 0 {

		fileSize = resp.ContentLength

	}

	acceptsRanges = strings.Contains(strings.ToLower(resp.Header.Get("Accept-Ranges")), "bytes")

	return fileSize, acceptsRanges, nil

}

func httpRange(ctx context.Context, url string, client *http.Client, start, end int64) ([]byte, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {

		return nil, err

	}

	if end >= start {

		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	}

	resp, err := client.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {

		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)

	}

	return io.ReadAll(resp.Body)

}

func httpSuffixRange(ctx context.Context, url string, client *http.Client, tailBytes int) ([]byte, int64, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {

		return nil, 0, err

	}

	req.Header.Set("Range", fmt.Sprintf("bytes=-%d", tailBytes))

	resp, err := client.Do(req)

	if err != nil {

		return nil, 0, err

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {

		return nil, 0, fmt.Errorf("HTTP %d", resp.StatusCode)

	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, 0, err

	}

	base := int64(0)

	if cr := resp.Header.Get("Content-Range"); cr != "" {

		rangePart, _, _ := strings.Cut(strings.TrimPrefix(cr, "bytes "), "/")

		if startPart, _, ok := strings.Cut(rangePart, "-"); ok {

			if parsed, parseErr := strconv.ParseInt(startPart, 10, 64); parseErr == nil {

				base = parsed

			}

		}

	}

	return data, base, nil

}

// stripMp4AACPayload removes MP4-style length prefixes when present; raw access units pass through.
func stripMp4AACPayload(data []byte) []byte {

	if len(data) < 6 {

		return data

	}

	// ADTS sync word
	if data[0] == 0xFF && (data[1]&0xF0) == 0xF0 {

		return data

	}

	length32 := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3]) // 4-byte big-endian length prefix (standard mp4a layout)

	if length32 > 8 && length32 <= len(data)-4 {

		return data[4 : 4+length32]

	}

	length16 := int(data[0])<<8 | int(data[1]) // 2-byte big-endian length prefix

	if length16 > 8 && length16 <= len(data)-2 {

		return data[2 : 2+length16]

	}

	return data

}

func readAtomHeader(data []byte, offset int) (size int, atomType string, headerSize int, ok bool) {

	if offset+8 > len(data) {

		return 0, "", 0, false

	}

	size = int(data[offset])<<24 | int(data[offset+1])<<16 | int(data[offset+2])<<8 | int(data[offset+3])
	atomType = string(data[offset+4 : offset+8])

	headerSize = 8

	if size == 1 {

		if offset+16 > len(data) {

			return 0, "", 0, false

		}

		size = int(data[offset+8])<<56 | int(data[offset+9])<<48 | int(data[offset+10])<<40 | int(data[offset+11])<<32 | int(data[offset+12])<<24 | int(data[offset+13])<<16 | int(data[offset+14])<<8 | int(data[offset+15])

		headerSize = 16

	}

	if size < headerSize {

		return 0, "", 0, false

	}

	return size, atomType, headerSize, true

}
