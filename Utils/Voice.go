package Utils

import (
	"Synthara-Redux/Structs"
	Utils "Synthara-Redux/Utils/Audio"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

var (

	Guilds = make(map[snowflake.ID]*Structs.Guild)
	GuildsMutex sync.RWMutex

)

// GetOrCreateGuild Retrieves or creates a guild instance
func GetOrCreateGuild(GuildID snowflake.ID, GuildName string) *Structs.Guild {

	GuildsMutex.Lock()
	defer GuildsMutex.Unlock()

	if Guild, Exists := Guilds[GuildID]; Exists {

		return Guild

	}

	Guild := Structs.NewGuild(GuildID, GuildName)
	Guilds[GuildID] = Guild

	return Guild

}

// GetGuild Retrieves a guild instance
func GetGuild(GuildID snowflake.ID) (*Structs.Guild, bool) {

	GuildsMutex.RLock()
	defer GuildsMutex.RUnlock()

	Guild, Exists := Guilds[GuildID]
	return Guild, Exists

}

// ConnectToVoiceChannel Connects to a voice channel in a guild
func ConnectToVoiceChannel(GuildID snowflake.ID, ChannelID snowflake.ID) error {

	Guild, Exists := GetGuild(GuildID)

	if !Exists {

		return errors.New("guild not found")

	}

	Guild.StreamerMutex.Lock()
	defer Guild.StreamerMutex.Unlock()

	if Guild.VoiceConnection != nil {

		CloseContext, CloseCancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer CloseCancel()

		Guild.VoiceConnection.Close(CloseContext)

	}

	ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 10 * time.Second)
	defer CancelFunc()

	VoiceConnection := DiscordClient.VoiceManager().CreateConn(GuildID)

	ErrorOpening := VoiceConnection.Open(ContextToUse, ChannelID, false, false)

	if ErrorOpening != nil {

		return ErrorOpening

	}

	Guild.VoiceConnection = VoiceConnection
	Guild.SetVoiceChannel(ChannelID)

	Logger.Info("Connected to voice channel in guild " + GuildID.String())

	return nil

}

// OpusProvider Implements OpusFrameProvider interface for SegmentStreamer
type OpusProvider struct {

	Streamer   *Utils.SegmentStreamer
	HTTPClient *http.Client
	Segments   []interface{}
	Index      int

}

func (P *OpusProvider) ProvideOpusFrame() ([]byte, error) {

	Frame, Available := P.Streamer.GetNextFrame()

	if Frame != nil && Available {

		return Frame, nil

	}

	return nil, nil

}

func (P *OpusProvider) Close() {

	P.Streamer.Close()

}

// PlaySongInGuild Plays a song in a guild's voice channel
func PlaySongInGuild(Guild *Structs.Guild, Song Structs.Song) error {

	if Guild.VoiceConnection == nil {

		return errors.New("not connected to voice channel")

	}

	// Fetch HLS segments for the song

	Segments, ErrorFetchingSegments := GetSongAudioSegments(Song.YouTubeID)

	if ErrorFetchingSegments != nil {

		return ErrorFetchingSegments

	}

	if len(Segments) == 0 {

		return errors.New("no audio segments available")

	}

	// Create segment streamer

	Streamer, ErrorCreatingStreamer := Utils.NewSegmentStreamer(0.0, len(Segments))

	if ErrorCreatingStreamer != nil {

		return ErrorCreatingStreamer

	}

	// Create opus provider

	Provider := &OpusProvider{

		Streamer:   Streamer,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Segments:   make([]interface{}, len(Segments)),
		Index:      0,

	}

	// Copy segments

	for I, Seg := range Segments {

		Provider.Segments[I] = Seg

	}

	// Set opus frame provider

	Guild.VoiceConnection.SetOpusFrameProvider(Provider)

	// Set speaking flag

	ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
	defer CancelFunc()

	ErrorSpeaking := Guild.VoiceConnection.SetSpeaking(ContextToUse, 1)

	if ErrorSpeaking != nil {

		return ErrorSpeaking

	}

	// Start fetching and processing segments in background

	go func() {

		HTTPClient := &http.Client{Timeout: 10 * time.Second}

		for Index := 0; Index < len(Segments); Index++ {

			if !Streamer.ShouldFetchNext() {

				time.Sleep(100 * time.Millisecond)
				Index--
				continue

			}

			Segment := Segments[Index]

			Response, ErrorFetching := HTTPClient.Get(Segment.URI)

			if ErrorFetching != nil {

				Logger.Error("Error fetching segment: " + ErrorFetching.Error())
				continue

			}

			SegmentBytes, ErrorReading := io.ReadAll(Response.Body)
			Response.Body.Close()

			if ErrorReading != nil {

				Logger.Error("Error reading segment: " + ErrorReading.Error())
				continue

			}

			ErrorProcessing := Streamer.ProcessNextSegment(SegmentBytes)

			if ErrorProcessing != nil {

				Logger.Error("Error processing segment: " + ErrorProcessing.Error())

			}

		}

	}()

	Logger.Info("Now playing: " + Song.Title)

	// Monitor playback

	for {

		CurrentIndex, TotalSegments := Streamer.GetProgress()

		if CurrentIndex >= TotalSegments && len(Streamer.OpusFrameChan) == 0 {

			break

		}

		time.Sleep(500 * time.Millisecond)

	}

	return nil

}
