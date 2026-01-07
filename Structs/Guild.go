package Structs

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Utils"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/gorilla/websocket"
)

var GuildStore = make(map[snowflake.ID]*Guild)
var GuildStoreMutex sync.Mutex

type Guild struct {

	ID snowflake.ID `json:"id"`
	Locale discord.Locale `json:"locale"`

	Queue Queue `json:"queue"`

	Channels Channels `json:"channels"`
	
	Features Features `json:"features"`

	VoiceConnection voice.Conn `json:"-"`
	StreamerMutex sync.Mutex `json:"-"`

	Internal GuildInternal `json:"-"`

}

const (

	Event_Initial = "INITIAL_STATE"

	Event_QueueUpdated = "QUEUE_UPDATED"
	Event_StateChanged = "STATE_CHANGED"

	Event_ProgressUpdate = "PROGRESS_UPDATE"

)

type Channels struct {

	Voice snowflake.ID `json:"voice"`
	Text snowflake.ID `json:"text"`

}

type Features struct {

	Repeat int `json:"repeat"`
	Shuffle bool `json:"shuffle"`
	Autoplay bool `json:"autoplay"`

}

type GuildInternal struct {

	Disconnecting bool `json:"disconnecting"`

}

// NewGuild Creates a new Guild instance
func NewGuild(ID snowflake.ID, Locale discord.Locale) *Guild {

	Created := &Guild{

		ID:   ID,
		Locale: Locale,

		Queue: Queue{

			ParentID: ID,

			State:   StateIdle,

			Previous: []*Innertube.Song{},
			Current:  nil,
			Upcoming: []*Innertube.Song{},

			Functions: QueueFunctions{

				State: QueueStateHandler,
				Updated: QueueUpdatedHandler,

			},

			WebSockets: make(map[*websocket.Conn]bool),

		},

		Channels: Channels{

			Voice: 0,
			Text:  0,

		},

		Features: Features{

			Repeat:   RepeatOff,
			Shuffle:  false,
			Autoplay: false,

		},

		VoiceConnection: nil,

		Internal: GuildInternal{

			Disconnecting: false,

		},
		
	}

	// Store the guild

	GuildStoreMutex.Lock()
	GuildStore[ID] = Created
	GuildStoreMutex.Unlock()

	return Created

}

func GetGuild(ID snowflake.ID, Create bool) *Guild {

	GuildStoreMutex.Lock()
	GuildInstance, Exists := GuildStore[ID]
	GuildStoreMutex.Unlock()

	if Exists {

		return GuildInstance
		
	} else if Create {

		GuildInstance, ExistsInCache := Globals.DiscordClient.Caches.GuildCache().Get(ID)

		if !ExistsInCache {

			FetchedGuild, ErrorFetching := Globals.DiscordClient.Rest.GetGuild(ID, false)

			if ErrorFetching == nil {

				GuildInstance = FetchedGuild.Guild

			}

		}

		return NewGuild(ID, discord.Locale(GuildInstance.PreferredLocale))

	} else {

		return nil

	}

}

// Connect Establishes a voice connection to the specified channel. Gateway events are handled.
func (G *Guild) Connect(VoiceChannelID snowflake.ID, TextChannelID snowflake.ID) error {

	G.Channels.Voice = VoiceChannelID
	G.Channels.Text = TextChannelID

	if G.Channels.Voice == 0 {

		return errors.New("no voice channel set")

	}

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection != nil {

		return nil; // Already connected, so we're done
		
	}

	OpenContext, CancelFunc := context.WithTimeout(context.Background(), 10 * time.Second)
	defer CancelFunc()

	VoiceConnection := Globals.DiscordClient.VoiceManager.CreateConn(G.ID)

	ErrorOpening := VoiceConnection.Open(OpenContext, G.Channels.Voice, false, true)

	if ErrorOpening != nil {

		return ErrorOpening

	}

	G.VoiceConnection = VoiceConnection

	return nil;

}
 
// Disconnect Closes the existing voice connection; if none exists, returns an error
func (G *Guild) Disconnect(CloseConn bool) error {

	G.Internal.Disconnecting = true

	return G.Cleanup(CloseConn)

}

// Stops playback, tickers, closes websockets, and the voice connection
func (G *Guild) Cleanup(CloseConn bool) error {

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	Utils.Logger.Info(fmt.Sprintf("Cleaning up guild: %s", G.ID.String()))

	G.Internal.Disconnecting = true // Marks as disconnecting early to prevent re-entrancy

	// Removes immediately from guild store so no new operations re-acquire this guild

	GuildStoreMutex.Lock()
	delete(GuildStore, G.ID)
	GuildStoreMutex.Unlock()

	// Stop playback session if present

	if G.Queue.PlaybackSession != nil {

		// Stop should be safe to call multiple times

		G.Queue.PlaybackSession.Stop()
		G.Queue.PlaybackSession = nil

	}

	// Clears and closes all websockets safely

	G.Queue.SocketMutex.Lock()
	for conn := range G.Queue.WebSockets {

		func(c *websocket.Conn) {

			defer func() {

				if r := recover(); r != nil {

					// noop

				}
				
			}()

			_ = c.Close()

		}(conn)

		delete(G.Queue.WebSockets, conn)

	}

	// Releases map to allow GC of connection objects

	G.Queue.WebSockets = nil
	G.Queue.SocketMutex.Unlock()

	// Clears queue items to free memory

	G.Queue.Current = nil
	G.Queue.Previous = nil
	G.Queue.Upcoming = nil

	// Closes voice connection if requested

	if CloseConn && G.VoiceConnection != nil {

		ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer CancelFunc()

		_ = G.VoiceConnection.SetSpeaking(ContextToUse, 0)

		G.VoiceConnection.SetOpusFrameProvider(nil)

		G.VoiceConnection.Close(ContextToUse)
		G.VoiceConnection = nil

	}

	return nil

}

// Play starts playing the song
func (G *Guild) Play(Song *Innertube.Song) error {

	Segments, SegmentDur, ErrorFetchingSegments := Innertube.GetSongAudioSegments(Song.YouTubeID)

	if ErrorFetchingSegments != nil {

		return ErrorFetchingSegments

	}

	if G.Queue.PlaybackSession != nil {

		G.Queue.PlaybackSession.Stop()
		G.Queue.PlaybackSession = nil

	}

	OnFinished := func() {

		Utils.Logger.Info(fmt.Sprintf("Playback finished for song: %s", Song.Title))

		G.VoiceConnection.SetOpusFrameProvider(nil)
		G.Queue.PlaybackSession = nil

		ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
		defer CancelFunc()

		_ = G.VoiceConnection.SetSpeaking(ContextToUse, 0)

		G.Queue.SetState(StateIdle)

	}

	Playback, ErrorCreatingPlayback := Audio.Play(Segments, SegmentDur, OnFinished, G.Queue.SendToWebsockets)

	if ErrorCreatingPlayback != nil {

		return ErrorCreatingPlayback

	}

	Provider := &Audio.OpusProvider{

		Streamer: Playback.Streamer,

	}

	G.VoiceConnection.SetOpusFrameProvider(Provider)

	ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
	defer CancelFunc()

	ErrorSpeaking := G.VoiceConnection.SetSpeaking(ContextToUse, 1)

	if ErrorSpeaking != nil {

		Playback.Stop()
		return ErrorSpeaking

	}

	G.Queue.SetState(StatePlaying)
	G.Queue.PlaybackSession = Playback

	return nil

}