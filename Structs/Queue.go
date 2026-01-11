package Structs

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"fmt"
	"sync"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/gorilla/websocket"
)

const (

	StateIdle    = iota
	StatePlaying = iota
	StatePaused  = iota

)

const (

	RepeatOff = iota
	RepeatOne = iota
	RepeatAll = iota

)

type Queue struct {

	ParentID snowflake.ID `json:"parent_id"`

	State int `json:"state"`

	Previous []*Innertube.Song `json:"previous"`
	Current  *Innertube.Song   `json:"current"`
	Upcoming []*Innertube.Song `json:"next"`

	Functions      QueueFunctions  `json:"-"`
	
	PlaybackSession *Audio.Playback `json:"-"`

	WebSockets      map[*websocket.Conn]bool `json:"-"`
	SocketMutex     sync.Mutex               `json:"-"`

}

type QueueFunctions struct {

	State func(Queue *Queue, State int) `json:"-"`
	Updated func(Queue *Queue) `json:"-"`

}

func (Q *Queue) SendToWebsockets(Event string, Data interface{}) {

	Q.SocketMutex.Lock()
	defer Q.SocketMutex.Unlock()

	Payload := map[string]interface{}{

		"Event": Event,
		"Data":  Data,

	}

	for Connection := range Q.WebSockets {

		ErrorSending := Connection.WriteJSON(Payload)

		if ErrorSending != nil {

			Connection.Close()
			delete(Q.WebSockets, Connection)

		}

	}

}

// Event-Like Handlers

func QueueStateHandler(Queue *Queue, State int) {

	Utils.Logger.Info(fmt.Sprintf("Queue %s state changed to %d", Queue.ParentID.String(), State))
	Queue.SendToWebsockets(Event_StateChanged, map[string]interface{}{"State": State})

	// Check Queue state and perform actions

	switch State {

		case StateIdle:

			// Idle state; move to next song if available
			// TODO: Repeat/Shuffle and autoplay logic

			Utils.Logger.Info(fmt.Sprintf("Queue %s is now idle; moving on...", Queue.ParentID.String()))

			Advanced := Queue.Next()

			if Advanced {

				Utils.Logger.Info(fmt.Sprintf("Queue %s advanced to next song: %s", Queue.ParentID.String(), Queue.Current.Title))

				Guild := GetGuild(Queue.ParentID, false) // does not create if not found

				State := Innertube.QueueInfo{

					GuildID: Queue.ParentID,

					SongPosition: 0,

					TotalPrevious: len(Guild.Queue.Previous),
					TotalUpcoming: len(Guild.Queue.Upcoming),

					Locale: Guild.Locale.Code(),

				}

				go func() { // we don't need to wait for this...

					_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.MessageCreate{

						Embeds: []discord.Embed{Queue.Current.Embed(State)},
						
					})

					if ErrorSending != nil {

						Utils.Logger.Error(fmt.Sprintf("Error sending message to channel %s for Queue %s: %s", Guild.Channels.Text, Queue.ParentID.String(), ErrorSending.Error()))

					}

				}()

				ErrorPlaying := Guild.Play(Queue.Current)

				if ErrorPlaying != nil {

					Utils.Logger.Error(fmt.Sprintf("Error playing song %s for Queue %s: %s", Queue.Current.Title, Queue.ParentID.String(), ErrorPlaying.Error()))

				}

			} else {

				Utils.Logger.Info(fmt.Sprintf("Queue %s has no more songs to play", Queue.ParentID.String()))

				// Send queue update to notify UI that queue has ended
				Queue.Functions.Updated(Queue)

				// Sends a message indicating the queue has ended

				Guild := GetGuild(Queue.ParentID, false) // does not create if not found
				TextChannelID := Guild.Channels.Text

				go func() { 

					_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(TextChannelID, discord.MessageCreate{

						Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

							Title:       Localizations.Get("Embeds.Notifications.QueueEnded.Title", Guild.Locale.Code()),
							Author:      Localizations.Get("Embeds.Categories.Notifications", Guild.Locale.Code()),
							Description: Localizations.Get("Embeds.Notifications.QueueEnded.Description", Guild.Locale.Code()),

						})},

					})

					if ErrorSending != nil {

						Utils.Logger.Error(fmt.Sprintf("Error sending queue ended message to channel %s for Queue %s: %s", TextChannelID, Queue.ParentID.String(), ErrorSending.Error()))

					}

				}()

				Guild.Cleanup(true)

				Queue.Current = nil;

			}

		case StatePaused:

			if Queue.PlaybackSession != nil {

				Queue.PlaybackSession.Pause()

			}

		case StatePlaying:

			if Queue.PlaybackSession != nil {

				Queue.PlaybackSession.Resume()

			}

	}
	
}

func QueueUpdatedHandler(Queue *Queue) {

	Queue.SendToWebsockets(Event_QueueUpdated, map[string]interface{}{ 

		"Current": Queue.Current,
		"Previous": Queue.Previous,
		"Upcoming": Queue.Upcoming,
		
	})

	if len(Queue.Upcoming) == 0 {

		return

	}

	NextSong := Queue.Upcoming[0]

	// Pre-caches HLS manifest for next song
	
	_, ErrorGettingManifest := Innertube.GetSongManifestURL(NextSong.YouTubeID)

	if ErrorGettingManifest != nil {

		Utils.Logger.Error(fmt.Sprintf("Error caching HLS manifest for song %s: %s", NextSong.Title, ErrorGettingManifest.Error()))

	}
	
}

// Queue Functions

func (Q *Queue) SetState(NewState int) {	

	Q.State = NewState
	go Q.Functions.State(Q, NewState) // done parallel since it may block, and we don't need to wait in this case...

}

// Next moves forward by one song in the upcoming queue; returns false when none exist.
func (Q *Queue) Next() bool {

	return Q.moveTo(1, true)

}

// Previous moves to the most recently played song; returns false when there is no history.
func (Q *Queue) Last() bool {

	return Q.moveTo(-1, true)

}

// Jump moves to the 1-indexed position within the upcoming queue; returns false for invalid positions.
func (Q *Queue) Jump(Index int) bool {

	return Q.moveTo(Index, true)

}

// Remove deletes a song from the upcoming queue at the specified 0-indexed position; returns false for invalid positions.
func (Q *Queue) Remove(Index int) bool {

	if Index < 0 || Index >= len(Q.Upcoming) {

		return false

	}

	Q.Upcoming = append(Q.Upcoming[:Index], Q.Upcoming[Index+1:]...)
	Q.Functions.Updated(Q)

	return true

}

// Move reorders a song in the upcoming queue from one 0-indexed position to another; returns false for invalid positions.
func (Q *Queue) Move(FromIndex int, ToIndex int) bool {

	if FromIndex < 0 || FromIndex >= len(Q.Upcoming) || ToIndex < 0 || ToIndex >= len(Q.Upcoming) {

		return false

	}

	if FromIndex == ToIndex {

		return true

	}

	Song := Q.Upcoming[FromIndex]
	Q.Upcoming = append(Q.Upcoming[:FromIndex], Q.Upcoming[FromIndex+1:]...)

	if ToIndex > FromIndex {

		ToIndex--

	}

	Q.Upcoming = append(Q.Upcoming[:ToIndex], append([]*Innertube.Song{Song}, Q.Upcoming[ToIndex:]...)...)
	Q.Functions.Updated(Q)

	return true

}

// Replay moves to a previously played song at the specified 0-indexed position and starts playback; returns false for invalid positions.
func (Q *Queue) Replay(Index int) bool {

	if Index < 0 || Index >= len(Q.Previous) {

		return false

	}

	// Move current song to front of upcoming

	if Q.Current != nil {

		Q.Upcoming = append([]*Innertube.Song{Q.Current}, Q.Upcoming...)

	}

	// Move songs after target index to front of upcoming (in reverse)

	for i := len(Q.Previous) - 1; i > Index; i-- {

		Q.Upcoming = append([]*Innertube.Song{Q.Previous[i]}, Q.Upcoming...)

	}

	// Set target song as current

	Q.Current = Q.Previous[Index]

	// Trim Previous array

	Q.Previous = Q.Previous[:Index]

	Q.Functions.Updated(Q)

	go Q.Play()
	return true

}

// ClearQueue resets the queue to an empty state
func (Q *Queue) Clear() {

	Q.Current = nil

	Q.Previous = []*Innertube.Song{}
	Q.Upcoming = []*Innertube.Song{}

	Q.Functions.Updated(Q)

}

// Add appends a song to the end of the queue OR current
func (Q *Queue) Add(Song *Innertube.Song, Requestor string) int {

	Song.Internal.Requestor = Requestor

	Pos := len(Q.Upcoming)

	if Q.Current == nil {

		Q.Current = Song

	} else {

		Q.Upcoming = append(Q.Upcoming, Song)
		Pos++ // Position in UPCOMING queue is 1-based

	}

	Q.Functions.Updated(Q)

	return Pos
	
}

// Play delegates playback of the current song to the Guild; returns false on failure.
func (Q *Queue) Play() bool {

	if Q.Current == nil {

		return false

	}

	Guild := GetGuild(Q.ParentID, false) // does not create if not found

	if Guild == nil {

		return false

	}

	ErrorPlaying := Guild.Play(Q.Current)

	if ErrorPlaying != nil {

		Utils.Logger.Error(fmt.Sprintf("Error playing song %s for Queue %s: %s", Q.Current.Title, Q.ParentID.String(), ErrorPlaying.Error()))
		return false

	}

	return true

}

// moveTo performs the queue movement; optionally starts playback when ShouldPlay is true. Positive indices navigate upcoming songs (1-indexed), negative indices navigate previous songs (-1 is most recent).
func (Q *Queue) moveTo(Index int, ShouldPlay bool) bool {

	if Index == 0 {

		return false // Index 0 is invalid

	}

	// Handle negative indexing for previous songs

	if Index < 0 {

		AbsIndex := -Index // Convert to positive for array indexing

		if AbsIndex > len(Q.Previous) {

			return false // Out of bounds

		}

		// Calculate how many songs to move back

		TargetIndex := len(Q.Previous) - AbsIndex

		// Move current song to front of upcoming

		if Q.Current != nil {

			Q.Upcoming = append([]*Innertube.Song{Q.Current}, Q.Upcoming...)

		}

		// Move songs between target and end of Previous to front of Upcoming

		if AbsIndex > 1 {

			MovedSongs := make([]*Innertube.Song, AbsIndex-1)
			copy(MovedSongs, Q.Previous[TargetIndex+1:])

			// Reverse order since we're moving backwards

			for i := len(MovedSongs) - 1; i >= 0; i-- {

				Q.Upcoming = append([]*Innertube.Song{MovedSongs[i]}, Q.Upcoming...)

			}

		}

		// Set target song as current

		Q.Current = Q.Previous[TargetIndex]

		// Trim Previous array

		Q.Previous = Q.Previous[:TargetIndex]

		if !ShouldPlay {

			return true

		}

		Q.Functions.Updated(Q)

		go Q.Play()
		return true

	}

	// Handle positive indexing for upcoming songs

	if Index < 1 || Index > len(Q.Upcoming) {

		return false

	}

	if Q.Current != nil {

		Q.Previous = append(Q.Previous, Q.Current)

	}

	if Index > 1 {

		Skipped := make([]*Innertube.Song, Index-1)
		copy(Skipped, Q.Upcoming[:Index-1])
		Q.Previous = append(Q.Previous, Skipped...)

	}

	Q.Current = Q.Upcoming[Index-1]
	Remaining := make([]*Innertube.Song, len(Q.Upcoming[Index:]))
	copy(Remaining, Q.Upcoming[Index:])

	Q.Upcoming = Remaining

	if !ShouldPlay {

		return true

	}

	Q.Functions.Updated(Q)

	go Q.Play() // same reason for goroutine as above
	return true

}