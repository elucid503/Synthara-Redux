package Structs

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"fmt"
	"math/rand"
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

	Previous []*Tidal.Song `json:"previous"`
	Current  *Tidal.Song   `json:"current"`
	Upcoming []*Tidal.Song `json:"next"`

	Suggestions []*Tidal.Song `json:"suggestions"`

	Functions      QueueFunctions  `json:"-"`
	
	PlaybackSession *Audio.MP4Playback `json:"-"`

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

	defer func() {

		if r := recover(); r != nil {

			Utils.Logger.Error("Queue", fmt.Sprintf("Panic in QueueStateHandler for queue %s: %v", Queue.ParentID.String(), r))
		}
		
	}()

	Utils.Logger.Info("Queue", fmt.Sprintf("Queue %s state changed to %d", Queue.ParentID.String(), State))
	Queue.SendToWebsockets(Event_StateChanged, map[string]interface{}{"State": State})

	// Check Queue state and perform actions

	switch State {

		case StateIdle:

			// Idle state; move to next song if available

			Utils.Logger.Info("Queue", fmt.Sprintf("Queue %s is now idle; moving on...", Queue.ParentID.String()))

			Guild := GetGuild(Queue.ParentID, false)

			if Guild == nil {

				return

			}

			// Handle Repeat One - replay current song
			if Guild.Features.Repeat == RepeatOne && Queue.Current != nil {

				Utils.Logger.Info("Queue", fmt.Sprintf("Queue %s repeating current song: %s", Queue.ParentID.String(), Queue.Current.Title))

				go Queue.Play()
				return

			}

			// Move current song to Previous before calling Next(), for AutoPlay seed
			if Queue.Current != nil {

				Queue.Previous = append(Queue.Previous, Queue.Current)
				Queue.Current = nil

			}

			Advanced := Queue.Next(false) // Notified below

			if Advanced {

				Utils.Logger.Info("Queue", fmt.Sprintf("Queue %s advanced to next song: %s", Queue.ParentID.String(), Queue.Current.Title))

				Queue.SendNowPlayingMessage()

				ErrorPlaying := Guild.Play(Queue.Current)

				if ErrorPlaying != nil {

					Utils.Logger.Error("Playback", fmt.Sprintf("Error playing song %s for Queue %s: %s", Queue.Current.Title, Queue.ParentID.String(), ErrorPlaying.Error()))

				}

			} else {

				Utils.Logger.Info("Queue", fmt.Sprintf("Queue %s has no more songs to play", Queue.ParentID.String()))

				// Send queue update to notify UI that queue has ended
				Queue.Functions.Updated(Queue)

				// Sends a message indicating the queue has ended

				TextChannelID := Guild.Channels.Text

				go func() { 

					AutoPlayButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Buttons.AutoPlay", Guild.Locale.Code()), "AutoPlay", "", 0).WithEmoji(discord.ComponentEmoji{

						ID: snowflake.MustParse(Icons.GetID(Icons.Sparkles)),

					})
					
					DisconnectButton := discord.NewButton(discord.ButtonStyleDanger, Localizations.Get("Buttons.Disconnect", Guild.Locale.Code()), "Disconnect", "", 0).WithEmoji(discord.ComponentEmoji{

						ID: snowflake.MustParse(Icons.GetID(Icons.Call)),

					})

					_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(TextChannelID, discord.NewMessageCreateBuilder().
						AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

							Title:       Localizations.Get("Embeds.Notifications.QueueEnded.Title", Guild.Locale.Code()),
							Author:      Localizations.Get("Embeds.Categories.Notifications", Guild.Locale.Code()),
							Description: Localizations.Get("Embeds.Notifications.QueueEnded.Description", Guild.Locale.Code()),

						})).
						AddActionRow(AutoPlayButton, DisconnectButton).
						Build())

					if ErrorSending != nil {

						Utils.Logger.Error("Command", fmt.Sprintf("Error sending queue ended message to channel %s for Queue %s: %s", TextChannelID, Queue.ParentID.String(), ErrorSending.Error()))

					}

				}()

				// Start inactivity timer instead of immediate cleanup
				Guild.StartInactivityTimer()

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

	defer func() {

		if r := recover(); r != nil {

			Utils.Logger.Error("Queue", fmt.Sprintf("Panic in QueueUpdatedHandler for queue %s: %v", Queue.ParentID.String(), r))
		}

	}()

	Queue.SendToWebsockets(Event_QueueUpdated, map[string]interface{}{ 

		"Current": Queue.Current,
		"Previous": Queue.Previous,
		"Upcoming": Queue.Upcoming,
		"Suggestions": Queue.Suggestions,
		
	})
	
	Guild := GetGuild(Queue.ParentID, false)

	if Guild != nil && Guild.Features.Autoplay {

		// Regenerate if we're running low on suggestions (fewer than 2) and queue is not empty

		if len(Queue.Suggestions) < 2 && (Queue.Current != nil || len(Queue.Previous) > 0) {

			go Queue.RegenerateSuggestions()

		}

	}

	if len(Queue.Upcoming) == 0 {

		return

	}

	NextSong := Queue.Upcoming[0]

	// Pre-cache streaming URL for next song
	
	_, ErrorGettingStream := Tidal.GetStreamURL(NextSong.TidalID)

	if ErrorGettingStream != nil {

		Utils.Logger.Error("Streaming", fmt.Sprintf("Error caching stream URL for song %s: %s", NextSong.Title, ErrorGettingStream.Error()))

	}
	
}

// Queue Functions

func (Q *Queue) SetState(NewState int) {	

	Q.State = NewState
	go Q.Functions.State(Q, NewState) // done parallel since it may block, and we don't need to wait in this case...

}

// Next moves forward by one song in the upcoming queue; returns false when none exist.
func (Q *Queue) Next(Notify bool) bool {

	Guild := GetGuild(Q.ParentID, false)

	// Check if we need to use AutoPlay when upcoming is empty

	if Guild != nil && len(Q.Upcoming) == 0 && Guild.Features.Autoplay {

		// Regenerate suggestions if none exist

		if len(Q.Suggestions) == 0 {

			Q.RegenerateSuggestions()

		}

		// Pull from suggestions if available

		if len(Q.Suggestions) > 0 {

			NextSuggestion := Q.Suggestions[0]
			Q.Suggestions = Q.Suggestions[1:]

			Q.Upcoming = append(Q.Upcoming, NextSuggestion)

		} else {

			Utils.Logger.Warn("AutoPlay", fmt.Sprintf("Failed to generate suggestions for Queue %s", Q.ParentID.String()))

		}

	}

	Resp := Q.moveTo(1, true)

	if (Notify && Q.Current != nil) {

		Q.SendNowPlayingMessage()

	}

	return Resp

}

// Previous moves to the most recently played song; returns false when there is no history.
func (Q *Queue) Last() bool {

	Success := Q.moveTo(-1, true)

	if Success {

		Q.SendNowPlayingMessage()

	}

	return Success

}

// Jump moves to the 1-indexed position within the upcoming queue; returns false for invalid positions.
func (Q *Queue) Jump(Index int) bool {

	Success := Q.moveTo(Index, true)

	if Success {

		Q.SendNowPlayingMessage()

	}

	return Success

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

	Q.Upcoming = append(Q.Upcoming[:ToIndex], append([]*Tidal.Song{Song}, Q.Upcoming[ToIndex:]...)...)
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

		Q.Upcoming = append([]*Tidal.Song{Q.Current}, Q.Upcoming...)

	}

	// Move songs after target index to front of upcoming (in reverse)

	for i := len(Q.Previous) - 1; i > Index; i-- {

		Q.Upcoming = append([]*Tidal.Song{Q.Previous[i]}, Q.Upcoming...)

	}

	// Set target song as current

	Q.Current = Q.Previous[Index]

	// Trim Previous array

	Q.Previous = Q.Previous[:Index]

	Q.Functions.Updated(Q)

	Q.SendNowPlayingMessage()

	go Q.Play()
	return true

}

// ClearQueue resets the queue to an empty state
func (Q *Queue) Clear() {

	Q.Current = nil

	Q.Previous = []*Tidal.Song{}
	Q.Upcoming = []*Tidal.Song{}

	Q.Functions.Updated(Q)

}

// Add appends a song to the end of the queue OR current
func (Q *Queue) Add(Song *Tidal.Song, Requestor string) int {

	Song.Internal.Requestor = Requestor

	Pos := len(Q.Upcoming)

	if Q.Current == nil {

		Q.Current = Song

	} else {

		Q.Upcoming = append(Q.Upcoming, Song)
		Pos++ // Position in UPCOMING queue is 1-based

	}

	go Q.Functions.Updated(Q)

	return Pos
	
}

// Play delegates playback of the current song to the Guild; returns false on failure.
func (Q *Queue) Play() bool {

	// Protect against panics that might occur in underlying libraries
	defer func() {

		if r := recover(); r != nil {

			Utils.Logger.Error("Playback", fmt.Sprintf("Panic recovered in Queue.Play for Queue %s: %v", Q.ParentID.String(), r))

			// Set state back to idle on panic
			Guild := GetGuild(Q.ParentID, false)

			if Guild != nil {

				Guild.StreamerMutex.Lock()
				Guild.Queue.SetState(StateIdle)
				Guild.StreamerMutex.Unlock()

			}

		}

	}()

	if Q.Current == nil {

		return false

	}

	Guild := GetGuild(Q.ParentID, false) // does not create if not found

	if Guild == nil {

		return false

	}

	ErrorPlaying := Guild.Play(Q.Current)

	if ErrorPlaying != nil {

		Utils.Logger.Error("Playback", fmt.Sprintf("Error playing song %s for Queue %s: %s", Q.Current.Title, Q.ParentID.String(), ErrorPlaying.Error()))
		
		// Set state back to idle on error
		
		Guild.StreamerMutex.Lock()
		Q.SetState(StateIdle)
		Guild.StreamerMutex.Unlock()
		
		return false

	}

	return true

}

// SendNowPlayingMessage sends a "now playing" embed to the guild's text channel
func (Q *Queue) SendNowPlayingMessage() {

	if Q.Current == nil {

		return

	}

	Guild := GetGuild(Q.ParentID, false)

	if Guild == nil {

		return

	}

	State := Tidal.QueueInfo{

		Playing: true,

		GuildID: Q.ParentID,
		SongPosition: 0,

		TotalPrevious: len(Q.Previous),
		TotalUpcoming: len(Q.Upcoming),

		Locale: Guild.Locale.Code(),
		
	}

	go func() {

		_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
			AddEmbeds(Q.Current.Embed(State)).
			AddActionRow(Q.Current.Buttons(State)...).
			Build())

		if ErrorSending != nil {

			Utils.Logger.Error("Command", fmt.Sprintf("Error sending now playing message to channel %s for Queue %s: %s", Guild.Channels.Text, Q.ParentID.String(), ErrorSending.Error()))
		
		}

	}()

}

// shuffleUpcoming uses Fisher-Yates algorithm to shuffle the upcoming queue in-place
func (Q *Queue) ShuffleUpcoming() {

	if len(Q.Upcoming) <= 1 {

		return

	}

	// Fisher-Yates shuffle
	for i := len(Q.Upcoming) - 1; i > 0; i-- {

		j := rand.Intn(i + 1)
		Q.Upcoming[i], Q.Upcoming[j] = Q.Upcoming[j], Q.Upcoming[i]

	}

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

			Q.Upcoming = append([]*Tidal.Song{Q.Current}, Q.Upcoming...)

		}

		// Move songs between target and end of Previous to front of Upcoming

		if AbsIndex > 1 {

			MovedSongs := make([]*Tidal.Song, AbsIndex-1)
			copy(MovedSongs, Q.Previous[TargetIndex+1:])

			// Reverse order since we're moving backwards

			for i := len(MovedSongs) - 1; i >= 0; i-- {

				Q.Upcoming = append([]*Tidal.Song{MovedSongs[i]}, Q.Upcoming...)

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

	// Checking features

	Guild := GetGuild(Q.ParentID, false)

	if Q.Current != nil {

		Q.Previous = append(Q.Previous, Q.Current)

		// Handles Repeat All - re-enqueue the current song after it's moved to previous

		if Guild != nil && Guild.Features.Repeat == RepeatAll {

			Q.Upcoming = append(Q.Upcoming, Q.Current)

		}

	}

	// Sets new current song

	Q.Current = Q.Upcoming[Index-1]
	Remaining := make([]*Tidal.Song, len(Q.Upcoming[Index:])) // shift by Index-1

	copy(Remaining, Q.Upcoming[Index:])

	Q.Upcoming = Remaining

	if Guild != nil && Guild.Features.Shuffle {

		Q.ShuffleUpcoming()

	}

	if !ShouldPlay {

		return true

	}

	Q.Functions.Updated(Q)

	go Q.Play() // same reason for goroutine as above
	
	return true

}

// RegenerateSuggestions fetches new song suggestions based on the last played song using Tidal mix
func (Q *Queue) RegenerateSuggestions() {

	Guild := GetGuild(Q.ParentID, false)

	if Guild == nil {

		Utils.Logger.Warn("AutoPlay", fmt.Sprintf("RegenerateSuggestions called but Guild is nil for Queue %s", Q.ParentID.String()))
		return

	}

	if !Guild.Features.Autoplay {

		Utils.Logger.Warn("AutoPlay", fmt.Sprintf("RegenerateSuggestions called but AutoPlay is disabled for Queue %s", Q.ParentID.String()))
		return
		
	}

	// Determine seed song (last in Previous queue, then Current)

	var SeedSong *Tidal.Song

	if len(Q.Previous) > 0 {

		SeedSong = Q.Previous[len(Q.Previous)-1]

	} else if Q.Current != nil {

		SeedSong = Q.Current

	} else {

		Utils.Logger.Warn("AutoPlay", fmt.Sprintf("No seed song available for Queue %s (Previous: %d, Current: %v)", Q.ParentID.String(), len(Q.Previous), Q.Current != nil))
		return // No seed available

	}

	Utils.Logger.Info("AutoPlay", fmt.Sprintf("Regenerating suggestions for Queue %s using seed: %s", Q.ParentID.String(), SeedSong.Title))

	// Get mix ID from current song if available, otherwise fetch it

	MixID := SeedSong.MixID
	
	if MixID == "" {

		var Err error

		MixID, Err = Tidal.FetchTrackMix(SeedSong.TidalID)

		if Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Error fetching track mix for Queue %s: %s", Q.ParentID.String(), Err.Error()))
			return

		}

	}

	// Fetch mix items (similar songs)

	SimilarSongs, ErrorFetching := Tidal.FetchMixItems(MixID)

	if ErrorFetching != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Error fetching mix items for Queue %s: %s", Q.ParentID.String(), ErrorFetching.Error()))
		return

	}

	// Filter out the seed song

	Filtered := make([]Tidal.Song, 0, len(SimilarSongs))

	for _, Song := range SimilarSongs {

		if Song.TidalID != SeedSong.TidalID {

			Filtered = append(Filtered, Song)

		}

	}

	// Randomize and take up to 5 suggestions

	MaxSuggestions := 5
	
	if len(Filtered) > MaxSuggestions {

		// Shuffle using Fisher-Yates and take first MaxSuggestions

		for i := len(Filtered) - 1; i > 0; i-- {

			j := rand.Intn(i + 1)
			Filtered[i], Filtered[j] = Filtered[j], Filtered[i]
			
		}

		Filtered = Filtered[:MaxSuggestions]

	}

	// Convert to pointers and mark as suggested

	Q.Suggestions = make([]*Tidal.Song, 0, len(Filtered))

	for i := range Filtered {

		Song := &Filtered[i]
		Song.Internal.Suggested = true
		Song.Internal.Requestor = "AutoPlay"

		Q.Suggestions = append(Q.Suggestions, Song)

	}

	Utils.Logger.Info("AutoPlay", fmt.Sprintf("Generated %d suggestions for Queue %s", len(Q.Suggestions), Q.ParentID.String()))

}