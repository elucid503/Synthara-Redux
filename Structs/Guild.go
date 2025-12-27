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

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/gorilla/websocket"
)

var GuildStore = make(map[snowflake.ID]*Guild)
var GuildStoreMutex sync.Mutex

type Guild struct {

	ID snowflake.ID `json:"id"`

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
func NewGuild(ID snowflake.ID) *Guild {

	Created := &Guild{

		ID:   ID,

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

func GetGuild(ID snowflake.ID) *Guild {

	GuildStoreMutex.Lock()
	GuildInstance, Exists := GuildStore[ID]
	GuildStoreMutex.Unlock()

	if Exists {

		return GuildInstance
		
	} else {

		return NewGuild(ID)

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

	G.Internal.Disconnecting = true // for other handlers to know

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection != nil {

		// Stops current session before disconnecting to prevent backpressure issues
		
		if G.Queue.PlaybackSession != nil {

			G.Queue.PlaybackSession.Stop()
			G.Queue.PlaybackSession = nil

		}

		if CloseConn {

			ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
			defer CancelFunc()

			// Stop speaking and detach provider before closing connection

			_ = G.VoiceConnection.SetSpeaking(ContextToUse, 0)
			G.VoiceConnection.SetOpusFrameProvider(nil)

			G.VoiceConnection.Close(ContextToUse)
			G.VoiceConnection = nil

		}

		// Removes guild from store to free up memory

		GuildStoreMutex.Lock()
		delete(GuildStore, G.ID)
		GuildStoreMutex.Unlock()

		return nil;

	} else {

		return errors.New("no active voice connection to disconnect")

	}

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

	Playback, ErrorCreatingPlayback := Audio.Play(Segments, SegmentDur, OnFinished)

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