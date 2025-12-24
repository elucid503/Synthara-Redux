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

}

const (

	StateIdle = iota
	StatePlaying = iota
	StatePaused = iota

)

type Queue struct {

	ParentID snowflake.ID `json:"parent_id"`

	State int `json:"state"`

	Previous []Innertube.Song `json:"previous"`
	Current *Innertube.Song `json:"current"`
	Upcoming []Innertube.Song `json:"next"`

	Functions QueueFunctions `json:"-"`
	CurrentStreamer *Audio.SegmentStreamer `json:"-"`

}

type Channels struct {

	Voice snowflake.ID `json:"voice"`
	Text snowflake.ID `json:"text"`

}

type QueueFunctions struct {

	Added func(Queue *Queue, Song Innertube.Song) `json:"-"`
	State func(Queue *Queue, State int) `json:"-"`
	Updated func(Queue *Queue) `json:"-"`

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

	Created := &Guild{

		ID:   ID,

		Queue: Queue{

			ParentID: ID,

			State:   StateIdle,

			Previous: []Innertube.Song{},
			Current:  nil,
			Upcoming: []Innertube.Song{},

			Functions: QueueFunctions{

				Added: QueueAddedHandler,
				State: QueueStateHandler,
				Updated: QueueUpdatedHandler,

			},

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

	// Store the guild

	GuildStoreMutex.Lock()
	GuildStore[ID] = Created
	GuildStoreMutex.Unlock()

	return Created

}

func GetOrCreateGuild(ID snowflake.ID) *Guild {

	GuildStoreMutex.Lock()
	GuildInstance, Exists := GuildStore[ID]
	GuildStoreMutex.Unlock()

	if Exists {

		return GuildInstance
		
	} else {

		return NewGuild(ID)

	}

}

// Event-Like Handlers

func QueueAddedHandler(Queue *Queue, Song Innertube.Song) {

	Utils.Logger.Info(fmt.Sprintf("Song %s was enqueued for Queue %s", Song.Title, Queue.ParentID.String()))
	
}

func QueueStateHandler(Queue *Queue, State int) {

	Utils.Logger.Info(fmt.Sprintf("Queue %s state changed to %d", Queue.ParentID.String(), State))

	// Check Queue state and perform actions

	switch State {

		case StateIdle:

			// Idle state; move to next song if available
			// TODO: Repeat/Shuffle and autoplay logic

			Utils.Logger.Info(fmt.Sprintf("Queue %s is now idle; moving on...", Queue.ParentID.String()))

			Advanced := Queue.Advance()

			if Advanced {

				Utils.Logger.Info(fmt.Sprintf("Queue %s advanced to next song: %s", Queue.ParentID.String(), Queue.Current.Title))

				Guild := GetOrCreateGuild(Queue.ParentID)

				ErrorPlaying := Guild.Play(*Queue.Current)

				if ErrorPlaying != nil {

					Utils.Logger.Error(fmt.Sprintf("Error playing song %s for Queue %s: %s", Queue.Current.Title, Queue.ParentID.String(), ErrorPlaying.Error()))

				}

			} else {

				Utils.Logger.Info(fmt.Sprintf("Queue %s has no more songs to play", Queue.ParentID.String()))

				Queue.Current = nil;

			}

		case StatePaused:

			if Queue.CurrentStreamer != nil {

				Queue.CurrentStreamer.Pause()

			}

		case StatePlaying:

			if Queue.CurrentStreamer != nil {

				Queue.CurrentStreamer.Resume()

			}

	}
	
}

func QueuePauseStateHandler(Queue *Queue, IsPaused bool) {

	if Queue.CurrentStreamer == nil {
		return
	}

	if IsPaused {

		Utils.Logger.Info(fmt.Sprintf("Queue %s paused", Queue.ParentID.String()))
		Queue.CurrentStreamer.Pause()

	} else {

		Utils.Logger.Info(fmt.Sprintf("Queue %s resumed", Queue.ParentID.String()))
		Queue.CurrentStreamer.Resume()

	}

}

func QueueUpdatedHandler(Queue *Queue) {

	if len(Queue.Upcoming) == 0 {

		return

	}

	NextSong := Queue.Upcoming[0]

	// Pre-caches HLS manifest for next song
	
	Utils.Logger.Info(fmt.Sprintf("Queue %s updated; caching HLS manifest for song: %s", Queue.ParentID.String(), NextSong.Title))

	_, ErrorGettingManifest := Innertube.GetSongManifestURL(NextSong.YouTubeID)

	if ErrorGettingManifest != nil {

		Utils.Logger.Error(fmt.Sprintf("Error caching HLS manifest for song %s: %s", NextSong.Title, ErrorGettingManifest.Error()))

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

// Play sets and plays the current song; DOES NOT return until the song is finished!
func (G *Guild) Play(Song Innertube.Song) (error) {
	
	// Fetch HLS segments for the song

	Segments, SegmentDur, ErrorFetchingSegments := Innertube.GetSongAudioSegments(Song.YouTubeID)

	if ErrorFetchingSegments != nil {

		return ErrorFetchingSegments

	}

	if len(Segments) == 0 {

		return errors.New("no audio segments available")

	}

	// Create segment streamer

	Streamer, ErrorCreatingStreamer := Audio.NewSegmentStreamer(float64(SegmentDur), len(Segments))

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

	G.Queue.ChangeState(StatePlaying) // Now is playing

	// Store streamer in queue for event handlers

	G.Queue.CurrentStreamer = Streamer

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

			if ErrorProcessing != nil && ErrorProcessing.Error() != "stream stopped" { // ignores stream stopped errors

				Utils.Logger.Error("Error processing segment: " + ErrorProcessing.Error())

			}

		}

	}()

	// Monitor playback; returns when done

	for {

		CurrentIndex, TotalSegments := Streamer.GetProgress()

		if CurrentIndex >= TotalSegments && len(Streamer.OpusFrameChan) == 0 {

			G.Queue.ChangeState(StateIdle) // No longer playing
			break

		}

		time.Sleep(250 * time.Millisecond) // Checks every 250ms

	}

	return nil
	
}

func (Q *Queue) ChangeState(NewState int) {	

	Q.State = NewState
	go Q.Functions.State(Q, NewState) // done parallel since it may block, and we don't need to wait in this case...

}

// playCurrent starts playback of the current song; returns false on failure.
func (Q *Queue) playCurrent() bool {

	if Q.Current == nil {

		return false

	}

	if Q.CurrentStreamer != nil {

		Q.CurrentStreamer.Stop() // ensures previous stream halts and stopChan is closed
		Q.CurrentStreamer = nil

	}

	Guild := GetOrCreateGuild(Q.ParentID)

	ErrorPlaying := Guild.Play(*Q.Current)

	if ErrorPlaying != nil {

		Utils.Logger.Error(fmt.Sprintf("Error playing song %s for Queue %s: %s", Q.Current.Title, Q.ParentID.String(), ErrorPlaying.Error()))
		return false

	}

	return true

}

// HasCurrent Checks if there is a current song playing
func (Q *Queue) HasCurrent() bool {

	return Q.Current != nil

}

// IsEmpty Checks if the queue is empty (no current song and no next songs)
func (Q *Queue) IsEmpty() bool {

	return Q.Current == nil && len(Q.Upcoming) == 0

}

// Add appends a song to the end of the queue OR current
func (Q *Queue) Add(Song Innertube.Song) int {

	Pos := len(Q.Upcoming)

	if Q.Current == nil {

		Q.Current = &Song

	} else {

		Q.Upcoming = append(Q.Upcoming, Song)
		Pos++ // Position in UPCOMING queue is 1-based

	}

	Q.Functions.Added(Q, Song)
	Q.Functions.Updated(Q)

	return Pos
	
}

// Advance moves to the next song in the queue; returns false if there are no more songs
func (Q *Queue) Advance() bool {

	return Q.jump(1, false)

}

// Next moves forward by one song in the upcoming queue; returns false when none exist.
func (Q *Queue) Next() bool {

	return Q.jump(1, true)

}

// Previous moves to the most recently played song; returns false when there is no history.
func (Q *Queue) Last() bool {

	if len(Q.Previous) == 0 {

		return false

	}

	if Q.Current != nil {

		Q.Upcoming = append([]Innertube.Song{*Q.Current}, Q.Upcoming...)

	}

	PrevIndex := len(Q.Previous) - 1
	PrevSong := Q.Previous[PrevIndex]
	Q.Previous = Q.Previous[:PrevIndex]
	Q.Current = &PrevSong

	Q.Functions.Updated(Q)

	go Q.playCurrent() // done in goroutine to avoid blocking
	return true

}

// Jump moves to the 1-indexed position within the upcoming queue; returns false for invalid positions.
func (Q *Queue) Jump(Index int) bool {

	return Q.jump(Index, true)

}

// jump performs the queue movement; optionally starts playback when shouldPlay is true.
func (Q *Queue) jump(Index int, shouldPlay bool) bool {

	if Index < 1 || Index > len(Q.Upcoming) {

		return false

	}

	if Q.Current != nil {

		Q.Previous = append(Q.Previous, *Q.Current)

	}

	if Index > 1 {

		Skipped := make([]Innertube.Song, Index-1)
		copy(Skipped, Q.Upcoming[:Index-1])
		Q.Previous = append(Q.Previous, Skipped...)

	}

	Target := Q.Upcoming[Index-1]
	Remaining := make([]Innertube.Song, len(Q.Upcoming[Index:]))
	copy(Remaining, Q.Upcoming[Index:])

	Q.Current = &Target
	Q.Upcoming = Remaining

	if !shouldPlay {

		return true

	}

	Q.Functions.Updated(Q)

	go Q.playCurrent() // same reason for goroutine as above
	return true

}

// ClearQueue resets the queue to an empty state
func (Q *Queue) Clear() {

	Q.Previous = []Innertube.Song{}
	Q.Current = nil
	Q.Upcoming = []Innertube.Song{}

	Q.Functions.Updated(Q)

}

func (Q *Queue) GetTimePlaying() int {

	// Returns current duration + all previous durations in seconds

	TotalSeconds := 0

	if Q.Current != nil {

		TotalSeconds += int(Q.CurrentStreamer.GetCurrentTime()) // converts float64 to int

	}

	for _, PreviousSong := range Q.Previous {

		TotalSeconds += PreviousSong.Duration.Seconds

	}

	return TotalSeconds

}