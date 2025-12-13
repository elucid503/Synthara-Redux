package Structs

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

type Guild struct {

	ID snowflake.ID `json:"id"`

	Queue Queue `json:"queue"`

	Channels Channels `json:"channels"`
	
	Features Features `json:"features"`

	VoiceConnection voice.Conn `json:"-"`
	StreamerMutex sync.Mutex `json:"-"`

}

type Queue struct {

	Previous []Song `json:"previous"`
	Current *Song `json:"current"`
	Next []Song `json:"next"`

}

type Channels struct {

	Voice snowflake.ID `json:"voice"`
	Text snowflake.ID `json:"text"`

}

const (

	RepeatOff = iota
	RepeatOne = iota
	RepeatAll = iota

)

type Features struct {

	Repeat int `json:"repeat"`
	Shuffle bool `json:"shuffle"`
	Autoplay bool `json:"autoplay"`

}

// NewGuild Creates a new Guild instance
func NewGuild(ID snowflake.ID) *Guild {

	return &Guild{

		ID:   ID,

		Queue: Queue{

			Previous: []Song{},
			Current:  nil,
			Next:     []Song{},

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

	}

}

// AddToQueue Adds a song to the end of the queue
func (G *Guild) AddToQueue(Song Song) {

	G.Queue.Next = append(G.Queue.Next, Song)

}

// AdvanceQueue Moves to the next song in the queue
func (G *Guild) AdvanceQueue() bool {

	if len(G.Queue.Next) == 0 {

		return false

	}

	if G.Queue.Current != nil {

		G.Queue.Previous = append(G.Queue.Previous, *G.Queue.Current)

	}

	G.Queue.Current = &G.Queue.Next[0]
	G.Queue.Next = G.Queue.Next[1:]

	return true

}

// HasCurrentSong Checks if there is a currently playing song
func (G *Guild) HasCurrentSong() bool {

	return G.Queue.Current != nil

}

// ClearQueue Clears the entire queue
func (G *Guild) ClearQueue() {

	G.Queue.Previous = []Song{}
	G.Queue.Current = nil
	G.Queue.Next = []Song{}

}

// SetVoiceChannel Sets the voice channel for the guild
func (G *Guild) SetVoiceChannel(ChannelID snowflake.ID) {

	G.Channels.Voice = ChannelID

}

// SetTextChannel Sets the text channel for the guild
func (G *Guild) SetTextChannel(ChannelID snowflake.ID) {

	G.Channels.Text = ChannelID

}

// DisconnectVoice Closes the voice connection if exists
func (G *Guild) DisconnectVoice() error {

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection != nil {

		ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
		defer CancelFunc()

		G.VoiceConnection.Close(ContextToUse)
		G.VoiceConnection = nil

	}

	return nil

}