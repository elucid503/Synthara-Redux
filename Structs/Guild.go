package Structs

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Utils"
	"context"
	"errors"
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

	Previous []Innertube.Song `json:"previous"`
	Current *Innertube.Song `json:"current"`
	Next []Innertube.Song `json:"next"`

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

			Previous: []Innertube.Song{},
			Current:  nil,
			Next:     []Innertube.Song{},

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

// Guild Functions

// Connect Establishes a voice connection to the specified channel. Gateway events are handled.
func (G *Guild) Connect(ToChannelID snowflake.ID) error {

	G.Channels.Voice = ToChannelID

	if G.Channels.Voice == 0 {

		return errors.New("no voice channel set")

	}

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection != nil {

		CloseContext, CloseCancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer CloseCancel()

		G.VoiceConnection.Close(CloseContext) // Close existing connection

	}

	OpenContext, CancelFunc := context.WithTimeout(context.Background(), 10 * time.Second)
	defer CancelFunc()

	VoiceConnection := Globals.DiscordClient.VoiceManager.CreateConn(G.ID)

	ErrorOpening := VoiceConnection.Open(OpenContext, G.Channels.Voice, false, false)

	if ErrorOpening != nil {

		return ErrorOpening

	}

	G.VoiceConnection = VoiceConnection

	return nil;

}
 
// Disconnect Closes the existing voice connection; if none exists, returns an error
func (G *Guild) Disconnect() error {

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection != nil {

		ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
		defer CancelFunc()

		G.VoiceConnection.Close(ContextToUse)
		G.VoiceConnection = nil

		return nil;

	} else {

		return errors.New("no active voice connection to disconnect")

	}

}

// PlayOrAdd sets and plays the current song if no current... otherwise adds; if play, DOES NOT return until the song is finished!
func (G *Guild) PlayOrAdd(Song Innertube.Song) error {
	
	if (G.Queue.HasCurrent()) {

		G.Queue.Add(Song)
		return nil

	}

	G.Queue.Current = &Song

	// Fetch HLS segments for the song

	Segments, ErrorFetchingSegments := Innertube.GetSongAudioSegments(Song.YouTubeID)

	if ErrorFetchingSegments != nil {

		return ErrorFetchingSegments

	}

	if len(Segments) == 0 {

		return errors.New("no audio segments available")

	}

	// Create segment streamer

	Streamer, ErrorCreatingStreamer := Audio.NewSegmentStreamer(0.0, len(Segments))

	if ErrorCreatingStreamer != nil {

		return ErrorCreatingStreamer

	}

	// Create opus provider

	Provider := &Audio.OpusProvider{

		Streamer:   Streamer,
		Segments:   make([]interface{}, len(Segments)),
		Index:      0,

	}

	// Copy segments

	for I, Seg := range Segments {

		Provider.Segments[I] = Seg

	}

	// Set opus frame provider

	G.VoiceConnection.SetOpusFrameProvider(Provider)

	// Set speaking flag

	ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
	defer CancelFunc()

	ErrorSpeaking := G.VoiceConnection.SetSpeaking(ContextToUse, 1)

	if ErrorSpeaking != nil {

		return ErrorSpeaking

	}

	// Start fetching and processing segments in background

	go func() {

		for Index := 0; Index < len(Segments); Index++ {

			if !Streamer.ShouldFetchNext() {

				time.Sleep(100 * time.Millisecond) // 100ms wait before retrying

				Index-- // goes back and retry next time...
				continue

			}

			Segment := Segments[Index]

			SegmentBytes, ErrorFetching := Innertube.GetAudioSegmentBytes(Segment)

			if ErrorFetching != nil {

				Utils.Logger.Error("Error fetching segment: " + ErrorFetching.Error())
				continue

			}

			ErrorProcessing := Streamer.ProcessNextSegment(SegmentBytes)

			if ErrorProcessing != nil {

				Utils.Logger.Error("Error processing segment: " + ErrorProcessing.Error())

			}

		}

	}()

	// Monitor playback; returns when done

	for {

		CurrentIndex, TotalSegments := Streamer.GetProgress()

		if CurrentIndex >= TotalSegments && len(Streamer.OpusFrameChan) == 0 {

			break

		}

		time.Sleep(500 * time.Millisecond)

	}

	return nil
	
}

// Queue Functions

// HasCurrent Checks if there is a current song playing
func (Q *Queue) HasCurrent() bool {

	return Q.Current != nil

}

// Add appends a song to the end of the queue
func (Q *Queue) Add(Song Innertube.Song) {

	Q.Next = append(Q.Next, Song)

}

// Advance moves to the next song in the queue; returns false if there are no more songs
func (Q *Queue) Advance() bool {

	if len(Q.Next) == 0 {

		return false

	}

	if Q.Current != nil {

		Q.Previous = append(Q.Previous, *Q.Current)

	}

	Q.Current = &Q.Next[0]
	Q.Next = Q.Next[1:]

	return true

}

// ClearQueue resets the queue to an empty state
func (Q *Queue) Clear() {

	Q.Previous = []Innertube.Song{}
	Q.Current = nil
	Q.Next = []Innertube.Song{}

}