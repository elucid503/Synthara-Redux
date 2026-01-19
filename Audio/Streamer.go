//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"Synthara-Redux/Utils"

	"layeh.com/gopus"
)

// MP4Streamer handles streaming MP4/AAC files from Tidal
type MP4Streamer struct {
	
	Processor *MP4Processor

	Paused  atomic.Bool
	Stopped atomic.Bool

	Progress      int64 // Progress in milliseconds
	BytesStreamed int64 // Total bytes streamed

	OpusFrameChan chan []byte

	CancelFunc context.CancelFunc

	Mutex sync.Mutex

}

// NewMP4Streamer creates a new streamer for MP4 files
func NewMP4Streamer() (*MP4Streamer, error) {

	Processor, Err := NewMP4Processor()

	if Err != nil {

		return nil, Err

	}

	Streamer := &MP4Streamer{

		Processor:     Processor,
		Progress:      0,

		OpusFrameChan: make(chan []byte, 100),

	}

	Streamer.Paused.Store(false)
	Streamer.Stopped.Store(false)

	return Streamer, nil
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

func (S *MP4Streamer) GetNextFrame() ([]byte, bool) {

	if S.IsPaused() {

		return nil, true

	}

	Frame, OK := <-S.OpusFrameChan

	if !OK {

		return nil, false // Channel closed

	}

	return Frame, true

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

	// Safely close OpusFrameChan

	defer func() {

		if r := recover(); r != nil {

			// Channel already closed

		}

	}()

	close(S.OpusFrameChan)

	if S.Processor != nil {

		S.Processor.Close()
		S.Processor = nil

	}

}

// StreamFromURL fetches and processes audio from a direct MP4 URL using chunked streaming
func (S *MP4Streamer) StreamFromURL(Ctx context.Context, URL string) error {

	Client := &http.Client{Timeout: 60 * time.Second}

	// First, fetch just the header to parse moov atom (usually first 500KB-1MB)

	HeaderSize := int64(1024 * 1024) // 1MB should be enough for moov
	HeaderReq, Err := http.NewRequest("GET", URL, nil)
	
	if Err != nil {

		return fmt.Errorf("failed to create header request: %w", Err)

	}

	HeaderReq.Header.Set("Range", fmt.Sprintf("bytes=0-%d", HeaderSize-1))

	HeaderResp, Err := Client.Do(HeaderReq)
	if Err != nil {

		return fmt.Errorf("failed to fetch MP4 header: %w", Err)

	}

	defer HeaderResp.Body.Close()

	if HeaderResp.StatusCode != http.StatusOK && HeaderResp.StatusCode != http.StatusPartialContent {

		return fmt.Errorf("HTTP %d", HeaderResp.StatusCode)

	}

	HeaderData, Err := io.ReadAll(HeaderResp.Body)
	if Err != nil {

		return fmt.Errorf("failed to read header: %w", Err)

	}

	// Parse MP4 header to extract metadata

	Info := &MP4AudioInfo{

		SampleRate:  44100,
		NumChannels: 2,

	}

	S.parseMP4Header(HeaderData, Info)

	if Info.ASC == nil || len(Info.SampleSizes) == 0 {

		return errors.New("failed to parse MP4 metadata - moov atom not found")

	}

	// Now stream the mdat content in chunks

	return S.streamMdatChunked(Ctx, URL, Info, Client)

}

// ProcessMP4 extracts AAC from MP4 and encodes to Opus with streaming output
func (S *MP4Streamer) ProcessMP4(Data []byte) error {

	// Parse MP4 to extract AAC frames with proper boundaries

	AACFrames, ASC, _, _, Err := S.Processor.ParseMP4(Data)
	
	if Err != nil {

		return fmt.Errorf("failed to parse MP4: %w", Err)
	}


	// Create decoder for raw AAC

	Decoder, Err := NewRawAACDecoder(ASC)

	if Err != nil {

		return fmt.Errorf("failed to create decoder: %w", Err)

	}

	defer Decoder.Close()

	// Safe send helper that handles closed channel

	SafeSend := func(Data []byte) bool {

		if S.IsStopped() {

			return false

		}

		defer func() {

			recover() // Ignore panic from closed channel

		}()

		S.OpusFrameChan <- Data

		atomic.AddInt64(&S.Progress, 20)
		atomic.AddInt64(&S.BytesStreamed, int64(len(Data)))

		return true

	}

	// Process frames in batches for lower latency

	SamplesPerOpusFrame := FrameSize * Channels
	PCMBuffer := make([]int16, 0, SamplesPerOpusFrame*4)
	
	for _, AACFrame := range AACFrames {

		if S.IsStopped() {

			break

		}
		
		// Decode single AAC frame (already resampled to 48kHz stereo)

		PCM, Err := Decoder.DecodeFrame(AACFrame)

		if Err != nil || len(PCM) == 0 {

			continue

		}
		
		PCMBuffer = append(PCMBuffer, PCM...)
		
		// Encode and send complete Opus frames immediately

		for len(PCMBuffer) >= SamplesPerOpusFrame {

			if S.IsStopped() {

				return nil

			}

			OpusData, Err := S.Processor.OpusEncoder.Encode(PCMBuffer[:SamplesPerOpusFrame], FrameSize, MaxPacketSize)

			if Err != nil {

				return Err

			}
			
			if !SafeSend(OpusData) {

				return nil

			}
			
			PCMBuffer = PCMBuffer[SamplesPerOpusFrame:]

		}

	}

	// Handle remaining samples with padding

	if len(PCMBuffer) > 0 && !S.IsStopped() {

		Padding := make([]int16, SamplesPerOpusFrame-len(PCMBuffer))
		PCMBuffer = append(PCMBuffer, Padding...)
		
		OpusData, Err := S.Processor.OpusEncoder.Encode(PCMBuffer, FrameSize, MaxPacketSize)

		if Err == nil {

			SafeSend(OpusData)

		}
		
	}

	return nil
}

// MP4Processor handles AAC extraction and encoding
type MP4Processor struct {
	
	AACDecoder  *FDKAACDecoder
	OpusEncoder *gopus.Encoder

}

func NewMP4Processor() (*MP4Processor, error) {

	AACDecoder, Err := NewFDKAACDecoder()

	if Err != nil {

		return nil, Err

	}

	OpusEncoder, Err := gopus.NewEncoder(SampleRate, Channels, gopus.Audio)

	if Err != nil {

		AACDecoder.Close()
		return nil, Err

	}

	OpusEncoder.SetBitrate(128000) // 128 kb/s

	return &MP4Processor{

		AACDecoder:  AACDecoder,
		OpusEncoder: OpusEncoder,

	}, nil

}

// MP4AudioInfo holds parsed audio information from MP4

type MP4AudioInfo struct {

	ASC         []byte   // AudioSpecificConfig

	SampleSizes []uint32 // Size of each AAC frame
	SampleRate  int

	NumChannels int
	
	MdatOffset  int64    // Offset to mdat payload
	MdatSize    int64    // Size of mdat payload

}

// ParseMP4 parses an MP4 file and extracts AAC frames with proper boundaries
// Returns: AAC frames, AudioSpecificConfig, SampleRate, NumChannels, error
func (P *MP4Processor) ParseMP4(Data []byte) ([][]byte, []byte, int, int, error) {

	Reader := bytes.NewReader(Data)
	
	Info := &MP4AudioInfo{

		SampleRate:  44100,
		NumChannels: 2,

	}
	
	var MdatData []byte
	
	// First pass: find all atoms

	for {

		StartPos, _ := Reader.Seek(0, io.SeekCurrent)
		
		Header := make([]byte, 8)
		_, Err := Reader.Read(Header)

		if Err == io.EOF {

			break

		}

		if Err != nil {

			return nil, nil, 0, 0, Err
			
		}
		
		AtomSize := int64(Header[0])<<24 | int64(Header[1])<<16 | int64(Header[2])<<8 | int64(Header[3])
		AtomType := string(Header[4:8])
		
		// Handles extended size

		if AtomSize == 1 {

			ExtHeader := make([]byte, 8)
			Reader.Read(ExtHeader)

			AtomSize = int64(ExtHeader[0])<<56 | int64(ExtHeader[1])<<48 | int64(ExtHeader[2])<<40 | int64(ExtHeader[3])<<32 | int64(ExtHeader[4])<<24 | int64(ExtHeader[5])<<16 | int64(ExtHeader[6])<<8 | int64(ExtHeader[7])
		
		}
		
		if AtomSize < 8 {
			
			break

		}
		
		DataSize := AtomSize - 8
		
		switch AtomType {

			// see below for comments on what each one is

			case "mdat":

				Info.MdatOffset = StartPos + 8
				Info.MdatSize = DataSize
				MdatData = make([]byte, DataSize)
				io.ReadFull(Reader, MdatData)
				
			case "moov":

				MoovData := make([]byte, DataSize)

				io.ReadFull(Reader, MoovData)
				parseMP4Moov(MoovData, Info)
				
			default:

				Reader.Seek(DataSize, io.SeekCurrent)

		}

	}
	
	if len(MdatData) == 0 {
		return nil, nil, 0, 0, errors.New("no mdat atom found")
	}
	
	// Split mdat into AAC frames using sample sizes
	var AACFrames [][]byte
	
	if len(Info.SampleSizes) > 0 {

		// Use sample size table to extract frames
		Offset := 0

		for _, Size := range Info.SampleSizes {

			if Offset+int(Size) > len(MdatData) {
				break
			}

			Frame := make([]byte, Size)
			copy(Frame, MdatData[Offset:Offset+int(Size)])

			AACFrames = append(AACFrames, Frame)
			Offset += int(Size)

		}

	} else {

		// No sample size table, so we return entire mdat as single frame
		AACFrames = append(AACFrames, MdatData)

	}
	
	if len(AACFrames) == 0 {

		return nil, nil, 0, 0, errors.New("no AAC frames extracted")

	}
	
	return AACFrames, Info.ASC, Info.SampleRate, Info.NumChannels, nil

}

// parseMP4Moov parses moov atom to extract audio configuration
func parseMP4Moov(Data []byte, Info *MP4AudioInfo) {
	
	// Parse nested atoms in moov
	parseMP4Atoms(Data, Info, 0)

}

// parseMP4Atoms recursively parses MP4 atoms
func parseMP4Atoms(Data []byte, Info *MP4AudioInfo, Depth int) {
	
	Offset := 0
	
	for Offset < len(Data)-8 {
		
		AtomSize := int(Data[Offset])<<24 | int(Data[Offset+1])<<16 | int(Data[Offset+2])<<8 | int(Data[Offset+3])
		AtomType := string(Data[Offset+4 : Offset+8])
		
		if AtomSize < 8 || Offset+AtomSize > len(Data) {
			break
		}
		
		AtomData := Data[Offset+8 : Offset+AtomSize]
		
		switch AtomType {
		case "trak", "mdia", "minf", "stbl":
			// Container atoms - parse recursively
			parseMP4Atoms(AtomData, Info, Depth+1)
			
		case "stsd":
			// Sample description - contains mp4a
			if len(AtomData) > 8 {
				parseMP4Atoms(AtomData[8:], Info, Depth+1) // Skip version/flags and entry count
			}
			
		case "mp4a":
			// Audio sample entry
			if len(AtomData) >= 28 {
				Info.NumChannels = int(AtomData[16])<<8 | int(AtomData[17])
				SampleRateFixed := int(AtomData[24])<<24 | int(AtomData[25])<<16 | int(AtomData[26])<<8 | int(AtomData[27])
				Info.SampleRate = SampleRateFixed >> 16
				
				// Parse nested atoms (esds is inside mp4a)
				if len(AtomData) > 28 {
					parseMP4Atoms(AtomData[28:], Info, Depth+1)
				}
			}
			
		case "esds":
			// Elementary stream descriptor - contains ASC
			if len(AtomData) > 4 {
				Info.ASC = extractASCFromEsds(AtomData)
			}
			
		case "stsz":
			// Sample size table
			if len(AtomData) >= 12 {
				// Version (1) + Flags (3) + Sample Size (4) + Sample Count (4)
				DefaultSize := int(AtomData[4])<<24 | int(AtomData[5])<<16 | int(AtomData[6])<<8 | int(AtomData[7])
				SampleCount := int(AtomData[8])<<24 | int(AtomData[9])<<16 | int(AtomData[10])<<8 | int(AtomData[11])
				
				if DefaultSize != 0 {
					// All samples have the same size
					Info.SampleSizes = make([]uint32, SampleCount)
					for i := 0; i < SampleCount; i++ {
						Info.SampleSizes[i] = uint32(DefaultSize)
					}
				} else {
					// Variable sample sizes
					Info.SampleSizes = make([]uint32, 0, SampleCount)
					for i := 0; i < SampleCount && 12+i*4+4 <= len(AtomData); i++ {
						Idx := 12 + i*4
						Size := uint32(AtomData[Idx])<<24 | uint32(AtomData[Idx+1])<<16 | uint32(AtomData[Idx+2])<<8 | uint32(AtomData[Idx+3])
						Info.SampleSizes = append(Info.SampleSizes, Size)
					}
				}
			}
		}
		
		Offset += AtomSize
	}
}

// DecodeAACFrames decodes individual AAC frames using FDK-AAC
func (P *MP4Processor) DecodeAACFrames(Frames [][]byte, ASC []byte) ([]int16, error) {
	
	if len(Frames) == 0 {
		return nil, errors.New("no AAC frames to decode")
	}
	
	// Use the raw AAC decoder with ASC
	var AllPCM []int16
	
	// Create decoder for raw AAC
	Decoder, Err := NewRawAACDecoder(ASC)
	if Err != nil {
		return nil, Err
	}
	defer Decoder.Close()
	
	// Decode each frame
	for _, Frame := range Frames {
		PCM, Err := Decoder.DecodeFrame(Frame)
		if Err != nil {
			continue // Skip bad frames
		}
		AllPCM = append(AllPCM, PCM...)
	}
	
	if len(AllPCM) == 0 {
		return nil, errors.New("no PCM data decoded")
	}
	
	return AllPCM, nil
}

// EncodePCMToOpus encodes PCM audio data to Opus frames
func (P *MP4Processor) EncodePCMToOpus(PCMData []int16) ([][]byte, error) {

	var OpusFrames [][]byte
	SamplesPerFrame := FrameSize * Channels

	// Process complete frames
	for i := 0; i+SamplesPerFrame <= len(PCMData); i += SamplesPerFrame {
		
		PCMFrame := PCMData[i : i+SamplesPerFrame]
		
		OpusData, Err := P.OpusEncoder.Encode(PCMFrame, FrameSize, MaxPacketSize)
		if Err != nil {
			return nil, Err
		}
		
		OpusFrames = append(OpusFrames, OpusData)
	}

	// Handle remaining samples with padding
	Remaining := len(PCMData) % SamplesPerFrame
	if Remaining > 0 {
		
		StartIdx := len(PCMData) - Remaining
		PCMFrame := make([]int16, SamplesPerFrame)
		copy(PCMFrame, PCMData[StartIdx:])
		
		OpusData, Err := P.OpusEncoder.Encode(PCMFrame, FrameSize, MaxPacketSize)
		if Err != nil {
			return nil, Err
		}
		
		OpusFrames = append(OpusFrames, OpusData)
	}

	return OpusFrames, nil
}

// ExtractAACFromMP4 extracts raw AAC frames from MP4 container
func (P *MP4Processor) ExtractAACFromMP4(Data []byte) ([]byte, []byte, int, int, error) {

	Reader := bytes.NewReader(Data)
	
	// Parse MP4 atoms to find mdat and audio config

	SampleRate := 44100
	NumChannels := 2
	
	var AudioData []byte
	var AudioSpecificConfig []byte
	
	for {

		// Read atom header (size + type)

		Header := make([]byte, 8)

		_, Err := Reader.Read(Header)

		if Err == io.EOF { break }
		if Err != nil { return nil, nil, 0, 0, Err}
		
		AtomSize := int64(Header[0])<<24 | int64(Header[1])<<16 | int64(Header[2])<<8 | int64(Header[3])
		AtomType := string(Header[4:8])
		
		if AtomSize < 8 { break } // 8 bytes minimum is valid 
		
		DataSize := AtomSize - 8
		
		if AtomType == "mdat" {

			// This contains the actual audio data

			AudioData = make([]byte, DataSize)

			_, Err := io.ReadFull(Reader, AudioData)

			if Err != nil {

				return nil, nil, 0, 0, fmt.Errorf("failed to read mdat: %w", Err)

			}
			
		} else if AtomType == "moov" {

			// Parse moov for audio configuration

			MoovData := make([]byte, DataSize)

			_, Err := io.ReadFull(Reader, MoovData)

			if Err != nil {

				continue

			}

			// Try to extract sample rate and ASC from moov atom

			ExtractedRate, ExtractedChannels, ASC := parseMoovForAudioConfig(MoovData)

			if ExtractedRate > 0 {

				SampleRate = ExtractedRate

			}

			if ExtractedChannels > 0 {

				NumChannels = ExtractedChannels

			}

			if len(ASC) > 0 {

				AudioSpecificConfig = ASC

			}

		} else {

			// Skips other atoms

			Reader.Seek(DataSize, io.SeekCurrent)

		}

	}
	
	if len(AudioData) == 0 {

		return nil, nil, 0, 0, errors.New("no audio data found in MP4")

	}
	
	return AudioData, AudioSpecificConfig, SampleRate, NumChannels, nil

}

// parseMoovForAudioConfig attempts to extract audio configuration from moov atom
func parseMoovForAudioConfig(Data []byte) (int, int, []byte) {
	
	SampleRate := 0
	Channels := 0

	var ASC []byte
	
	// Search for 'esds' atom which contains AudioSpecificConfig

	for i := 0; i < len(Data)-4; i++ {

		if Data[i] == 'e' && Data[i+1] == 's' && Data[i+2] == 'd' && Data[i+3] == 's' {

			// Found esds atom, parse it to find ASC

			ESDSData := Data[i+4:]
			ASC = extractASCFromEsds(ESDSData)

			break

		}

	}
	
	// Search for 'mp4a' atom which contains audio config

	for i := 0; i < len(Data)-28; i++ {
		if Data[i] == 'm' && Data[i+1] == 'p' && Data[i+2] == '4' && Data[i+3] == 'a' {

			// Found mp4a atom, extract sample rate and channels
			// Offset 16-17: channels, Offset 24-27: sample rate

			if i+28 <= len(Data) {

				Channels = int(Data[i+16])<<8 | int(Data[i+17])

				SampleRate = int(Data[i+24])<<24 | int(Data[i+25])<<16 | int(Data[i+26])<<8 | int(Data[i+27])
				SampleRate = SampleRate >> 16 // Sample rate is in 16.16 fixed point

			}

			break

		}

	}
	
	return SampleRate, Channels, ASC

}

// extractASCFromEsds extracts AudioSpecificConfig from esds atom data
func extractASCFromEsds(Data []byte) []byte {

	// esds format is complex with variable length descriptors
	// We're looking for the DecSpecificInfo descriptor (tag 0x05), which contains the AudioSpecificConfig
	
	if len(Data) < 10 {

		return nil

	}
	
	// Skip version and flags (4 bytes)

	Pos := 4
	
	for Pos < len(Data)-2 {

		Tag := Data[Pos]
		Pos++
		
		// Read length (variable length encoding)

		Length := 0

		for Pos < len(Data) && (Data[Pos]&0x80) != 0 {

			Length = (Length << 7) | int(Data[Pos]&0x7F)
			Pos++

		}

		if Pos < len(Data) {

			Length = (Length << 7) | int(Data[Pos]&0x7F)
			Pos++

		}
		
		if Tag == 0x05 { // DecSpecificInfoTag - this is the ASC

			if Pos+Length <= len(Data) && Length > 0 {

				asc := make([]byte, Length)
				copy(asc, Data[Pos:Pos+Length])

				return asc

			}
		}
		
		// For ES_Descriptor (0x03) and DecoderConfigDescriptor (0x04), 
		// we need to skip headers and continue parsing

		if Tag == 0x03 {

			// Skips ES_ID (2 bytes) and flags (1 byte)

			Pos += 3
			continue

		}
		if Tag == 0x04 {

			// Skips objectTypeIndication (1), streamType (1), bufferSize (3), maxBitrate (4), avgBitrate (4)

			Pos += 13
			continue

		}
		
		// Skip unknown tags

		Pos += Length

	}
	
	return nil

}

// DecodeAndEncodeAAC decodes raw AAC data and encodes to Opus
func (P *MP4Processor) DecodeAndEncodeAAC(AACData []byte, ASC []byte, SourceSampleRate int, NumChannels int) ([][]byte, error) {

	var OpusFrames [][]byte
	SamplesPerFrame := FrameSize * Channels
	
	PCMBuffer := make([]int16, 0, SamplesPerFrame*10)

	// For raw AAC in MP4, we need to decode the entire stream
	// The data is already in raw AAC format (without ADTS headers)
	
	var PCMData []int16
	var Err error

	// If we have ASC (AudioSpecificConfig), use raw decoding

	if len(ASC) > 0 {

		PCMData, Err = P.AACDecoder.DecodeRaw(AACData, ASC)

	} else {

		// Tries ADTS decoding as fallback
		PCMData, Err = P.decodeADTS(AACData)

	}
	
	if Err != nil {

		// Tries ADTS as last resort
		PCMData, Err = P.decodeADTS(AACData)

		if Err != nil {

			return nil, fmt.Errorf("failed to decode AAC: %w", Err)

		}

	}

	// Resamples if needed (most Tidal content is 44100Hz, Discord needs 48000Hz)

	if SourceSampleRate != SampleRate && SourceSampleRate > 0 {

		PCMData = ResamplePCM(PCMData, SourceSampleRate, SampleRate, NumChannels)

	}

	PCMBuffer = append(PCMBuffer, PCMData...)

	// Encode complete frames

	for len(PCMBuffer) >= SamplesPerFrame {

		PCMFrame := PCMBuffer[:SamplesPerFrame]

		OpusData, Err := P.OpusEncoder.Encode(PCMFrame, FrameSize, MaxPacketSize)

		if Err != nil {

			return nil, Err

		}

		OpusFrames = append(OpusFrames, OpusData)
		PCMBuffer = PCMBuffer[SamplesPerFrame:]

	}

	// Handle remaining samples with padding

	if len(PCMBuffer) > 0 {

		Padding := make([]int16, SamplesPerFrame-len(PCMBuffer))
		PCMBuffer = append(PCMBuffer, Padding...)

		OpusData, Err := P.OpusEncoder.Encode(PCMBuffer, FrameSize, MaxPacketSize)

		if Err != nil {

			return nil, Err

		}

		OpusFrames = append(OpusFrames, OpusData)

	}

	return OpusFrames, nil

}

// decodeADTS decodes ADTS-framed AAC data
func (P *MP4Processor) decodeADTS(Data []byte) ([]int16, error) {
	
	var AllPCM []int16
	Offset := 0

	for Offset < len(Data) {

		// Check for ADTS sync word (0xFF 0xFx)
		if Offset+7 > len(Data) {

			break

		}
		
		if Data[Offset] != 0xFF || (Data[Offset+1]&0xF0) != 0xF0 {

			Offset++
			continue

		}

		// Parse ADTS header
		ProtectionAbsent := (Data[Offset+1] & 0x01) != 0
		HeaderSize := 7

		if !ProtectionAbsent {

			HeaderSize = 9

		}
		
		FrameLength := int(Data[Offset+3]&0x03)<<11 | int(Data[Offset+4])<<3 | int(Data[Offset+5]>>5)
		
		if FrameLength < HeaderSize || Offset+FrameLength > len(Data) {

			Offset++
			continue

		}

		// Decode this frame

		Frame := Data[Offset : Offset+FrameLength]
		PCM, Err := P.AACDecoder.Decode(Frame)
		
		if Err == nil && PCM != nil {

			AllPCM = append(AllPCM, PCM...)

		}

		Offset += FrameLength

	}

	if len(AllPCM) == 0 {

		return nil, errors.New("no PCM data decoded")

	}

	return AllPCM, nil

}

// parseMP4Header parses the MP4 header to extract moov metadata
func (S *MP4Streamer) parseMP4Header(Data []byte, Info *MP4AudioInfo) {

	Reader := bytes.NewReader(Data)

	for {

		StartPos, _ := Reader.Seek(0, io.SeekCurrent)

		Header := make([]byte, 8)
		_, Err := Reader.Read(Header)

		if Err == io.EOF { break } // done

		if Err != nil { return } // errorhybTF% 

		AtomSize := int64(Header[0])<<24 | int64(Header[1])<<16 | int64(Header[2])<<8 | int64(Header[3])
		AtomType := string(Header[4:8])

		if AtomSize == 1 {

			ExtHeader := make([]byte, 8)
			Reader.Read(ExtHeader)
			
			AtomSize = int64(ExtHeader[0])<<56 | int64(ExtHeader[1])<<48 | int64(ExtHeader[2])<<40 | int64(ExtHeader[3])<<32 | int64(ExtHeader[4])<<24 | int64(ExtHeader[5])<<16 | int64(ExtHeader[6])<<8 | int64(ExtHeader[7])

		}

		if AtomSize < 8 {

			break

		}

		DataSize := AtomSize - 8

		switch AtomType {

		case "mdat":

			// mdat atom contains raw audio data

			Info.MdatOffset = StartPos + 8
			Info.MdatSize = DataSize
			Reader.Seek(DataSize, io.SeekCurrent)

		case "moov":

			// moov atom contains metadata

			MoovData := make([]byte, DataSize)

			io.ReadFull(Reader, MoovData)
			parseMP4Moov(MoovData, Info)

		default:

			// skips other atoms

			Reader.Seek(DataSize, io.SeekCurrent)

		}

	}

}

// streamMdatChunked streams mdat content in chunks and processes incrementally
func (S *MP4Streamer) streamMdatChunked(Ctx context.Context, URL string, Info *MP4AudioInfo, Client *http.Client) error {

	// Create decoder for raw AAC

	Decoder, Err := NewRawAACDecoder(Info.ASC)

	if Err != nil {

		return fmt.Errorf("failed to create decoder: %w", Err)

	}

	defer Decoder.Close()

	// Safe send helper

	SafeSend := func(Data []byte) bool {

		if S.IsStopped() {

			return false

		}

		defer func() {

			recover()

		}()

		S.OpusFrameChan <- Data

		atomic.AddInt64(&S.Progress, 20)
		atomic.AddInt64(&S.BytesStreamed, int64(len(Data)))

		return true

	}

	// Process frames in chunks

	SamplesPerOpusFrame := FrameSize * Channels
	PCMBuffer := make([]int16, 0, SamplesPerOpusFrame*4)

	CurrentOffset := Info.MdatOffset
	FrameIndex := 0
	ChunkSize := int64(256 * 1024) // 256KB chunks

	for FrameIndex < len(Info.SampleSizes) {

		if S.IsStopped() {

			break

		}

		// Calculate how many frames we can fit in this chunk

		ChunkEnd := CurrentOffset + ChunkSize
		FramesInChunk := []int{}
		ChunkActualSize := int64(0)

		for FrameIndex < len(Info.SampleSizes) {

			AACFrameSize := int64(Info.SampleSizes[FrameIndex])
			if CurrentOffset+ChunkActualSize+AACFrameSize > ChunkEnd {

				break

			}

			FramesInChunk = append(FramesInChunk, FrameIndex)
			ChunkActualSize += AACFrameSize
			FrameIndex++

		}

		if len(FramesInChunk) == 0 {

			break

		}

		// Fetch this chunk using range request

		RangeStart := CurrentOffset
		RangeEnd := CurrentOffset + ChunkActualSize - 1

		Req, Err := http.NewRequest("GET", URL, nil)
		if Err != nil {

			return fmt.Errorf("failed to create chunk request: %w", Err)

		}

		Req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", RangeStart, RangeEnd))

		Req = Req.WithContext(Ctx)

		if S.IsStopped() {

			return nil

		}

		Resp, Err := Client.Do(Req)

		if Err != nil {

			return fmt.Errorf("failed to fetch chunk: %w", Err)

		}

		if Resp.StatusCode != http.StatusOK && Resp.StatusCode != http.StatusPartialContent {

			Resp.Body.Close()
			return fmt.Errorf("chunk request failed: HTTP %d", Resp.StatusCode)

		}

		ChunkData, Err := io.ReadAll(Resp.Body)
		Resp.Body.Close()

		if Err != nil {

			return fmt.Errorf("failed to read chunk: %w", Err)

		}

		// Process each AAC frame in this chunk

		ChunkOffset := int64(0)

		for _, FrameIdx := range FramesInChunk {

			if S.IsStopped() {

				return nil

			}

			AACFrameSize := int64(Info.SampleSizes[FrameIdx])
			if ChunkOffset+AACFrameSize > int64(len(ChunkData)) {

				break

			}

			AACFrame := ChunkData[ChunkOffset : ChunkOffset+AACFrameSize]
			ChunkOffset += AACFrameSize

			// Decode AAC frame to PCM

			PCM, Err := Decoder.DecodeFrame(AACFrame)

			if Err != nil || len(PCM) == 0 {

				continue

			}

			PCMBuffer = append(PCMBuffer, PCM...)

			// Encode complete Opus frames immediately

			for len(PCMBuffer) >= SamplesPerOpusFrame {

				if S.IsStopped() {

					return nil

				}

				OpusData, Err := S.Processor.OpusEncoder.Encode(PCMBuffer[:SamplesPerOpusFrame], FrameSize, MaxPacketSize)

				if Err != nil {

					return Err

				}

				if !SafeSend(OpusData) {

					return nil

				}

				PCMBuffer = PCMBuffer[SamplesPerOpusFrame:]

			}

		}

		CurrentOffset += ChunkActualSize

	}

	// Handle remaining samples

	if len(PCMBuffer) > 0 && !S.IsStopped() {

		Padding := make([]int16, SamplesPerOpusFrame-len(PCMBuffer))
		PCMBuffer = append(PCMBuffer, Padding...)

		OpusData, Err := S.Processor.OpusEncoder.Encode(PCMBuffer, FrameSize, MaxPacketSize)
		if Err == nil {

			SafeSend(OpusData)

		}

	}

	return nil

}

func (P *MP4Processor) Close() {

	if P.AACDecoder != nil {

		P.AACDecoder.Close()

	}

}

// MP4Playback wraps the streamer for playback control
type MP4Playback struct {
	
	Streamer *MP4Streamer
	Stopped  atomic.Bool

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

// MP4OpusProvider implements the OpusFrameProvider interface
type MP4OpusProvider struct {

	Streamer *MP4Streamer

}

func (P *MP4OpusProvider) ProvideOpusFrame() ([]byte, error) {
	
	Frame, Available := P.Streamer.GetNextFrame()

	if Frame != nil && Available {

		return Frame, nil

	}

	return nil, nil

}

func (P *MP4OpusProvider) Close() {

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

			close(Streamer.OpusFrameChan)
		}()

		ShouldCallFinished := !Playback.Stopped.Load()

		Streamer.Mutex.Unlock()

		if ShouldCallFinished {

			if OnFinished != nil {

				OnFinished()

			}

		}

	}()

	return Playback, nil

}