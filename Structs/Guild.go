package Structs

import (
	"Synthara-Redux/APIs"
	"Synthara-Redux/APIs/Apple"
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/APIs/Spotify"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
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

	Event_Error = "ERROR"

)

type Channels struct {

	Voice snowflake.ID `json:"voice"`
	Text snowflake.ID `json:"text"`

}

type Features struct {

	Repeat int `json:"repeat"`
	Shuffle bool `json:"shuffle"`
	Autoplay bool `json:"autoplay"`
	Locked bool `json:"locked"`

}

type GuildInternal struct {

	Disconnecting bool `json:"disconnecting"`

	InactivityTimer *time.Timer `json:"-"`
	InactivityMutex sync.Mutex   `json:"-"`

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

			Suggestions: []*Innertube.Song{},

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
			Locked:   false,

		},

		VoiceConnection: nil,

		Internal: GuildInternal{

			Disconnecting: false,

			InactivityTimer: nil,

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

// Stops playback, closes websockets, and the voice connection
func (G *Guild) Cleanup(CloseConn bool) error {

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	Utils.Logger.Info(fmt.Sprintf("Cleaning up guild: %s", G.ID.String()))

	G.Internal.Disconnecting = true // Marks as disconnecting early to prevent re-entrancy

	// Stop inactivity timer if present
	G.StopInactivityTimer()

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
	G.Queue.Suggestions = nil

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

// StartInactivityTimer starts or resets the inactivity timer
func (G *Guild) StartInactivityTimer() {

	G.Internal.InactivityMutex.Lock()
	defer G.Internal.InactivityMutex.Unlock()

	// Stop existing timer if present
	if G.Internal.InactivityTimer != nil {

		G.Internal.InactivityTimer.Stop()

	}

	// Determine timeout duration based on AutoPlay setting
	Duration := 1 * time.Hour

	if G.Features.Autoplay {

		Duration = 3 * time.Hour

	}

	Utils.Logger.Info(fmt.Sprintf("Starting inactivity timer for guild %s with duration: %s", G.ID.String(), Duration.String()))

	// Create new timer
	G.Internal.InactivityTimer = time.AfterFunc(Duration, func() {

		Utils.Logger.Info(fmt.Sprintf("Inactivity timer expired for guild %s, disconnecting...", G.ID.String()))

		// Get guild locale for translations
		Locale := G.Locale.Code()

		// Determine duration string for message
		DurationKey := "Common.OneHour"

		if G.Features.Autoplay {

			DurationKey = "Common.ThreeHours"

		}

		// Send notification message before disconnecting
		go func() {

			DisconnectButton := discord.NewButton(discord.ButtonStylePrimary, Localizations.Get("Buttons.Reconnect", Locale), "Reconnect", "", 0).WithEmoji(discord.ComponentEmoji{

				ID: snowflake.MustParse(Icons.GetID(Icons.Call)),

			})

			_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
				AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.InactivityDisconnect.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
					Description: Localizations.GetFormat("Embeds.Notifications.InactivityDisconnect.Description", Locale, Localizations.Get(DurationKey, Locale)),

				})).
				AddActionRow(DisconnectButton).
				Build())

			if ErrorSending != nil {

				Utils.Logger.Error(fmt.Sprintf("Error sending inactivity disconnect message to guild %s: %s", G.ID.String(), ErrorSending.Error()))

			}

		}()

		G.Disconnect(true)

	})

}

// ResetInactivityTimer resets the inactivity timer if it exists
func (G *Guild) ResetInactivityTimer() {

	G.Internal.InactivityMutex.Lock()
	defer G.Internal.InactivityMutex.Unlock()

	if G.Internal.InactivityTimer != nil {

		Utils.Logger.Info(fmt.Sprintf("Resetting inactivity timer for guild %s", G.ID.String()))

		G.Internal.InactivityTimer.Stop()

		// Determine timeout duration based on AutoPlay setting
		Duration := 1 * time.Hour

		if G.Features.Autoplay {

			Duration = 3 * time.Hour

		}

		G.Internal.InactivityTimer.Reset(Duration)

	}

}

// StopInactivityTimer stops and clears the inactivity timer
func (G *Guild) StopInactivityTimer() {

	G.Internal.InactivityMutex.Lock()
	defer G.Internal.InactivityMutex.Unlock()

	if G.Internal.InactivityTimer != nil {

		Utils.Logger.Info(fmt.Sprintf("Stopping inactivity timer for guild %s", G.ID.String()))

		G.Internal.InactivityTimer.Stop()
		G.Internal.InactivityTimer = nil

	}

}

// RouteURI takes a Synthara-Redux URI string and handles adding/playing the content. Returns the song, its position in the queue, and any error
func (G *Guild) HandleURI(URI string, Requestor string) (*Innertube.Song, int, error) {

	Type, ID, ErrorParsing := APIs.ParseURI(URI)

	if ErrorParsing != nil {

		return nil, -1, ErrorParsing

	}

	var PosAdded int
	var SongFound *Innertube.Song

	switch Type {

		case APIs.URITypeNone:

			// Search for songs using the query

			SearchResults := Innertube.SearchForSongs(ID)

			if len(SearchResults) == 0 {

				return nil, -1, errors.New("no search results found")

			}

			SongFound = &SearchResults[0]

			PosAdded = G.Queue.Add(SongFound, Requestor)


		case APIs.URITypeSong:

			Song, SongFetchErr := Innertube.GetSong(ID)

			if SongFetchErr != nil {

				return nil, -1, SongFetchErr

			}

			SongFound = &Song

			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeVideo:

			// Can do the same thing as a song probably

			Song, SongFetchErr := Innertube.GetSong(ID)

			if SongFetchErr != nil {

				return nil, -1, SongFetchErr

			}

			SongFound = &Song

			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeAlbum:

			AlbumSongs, AlbumFetchErr := Innertube.GetAlbumSongs(ID)

			if AlbumFetchErr != nil {

				return nil, -1, AlbumFetchErr

			}

			if len(AlbumSongs) == 0 {

				return nil, -1, errors.New("album contains no songs")

			}

			SongFound = &AlbumSongs[0]

			for i, Song := range AlbumSongs {

				Pos := G.Queue.Add(&Song, Requestor)

				if i == 0 {

					PosAdded = Pos // records Pos of first song added

				}

			}

		case APIs.URITypeArtist:

			ArtistSongs, ArtistFetchErr := Innertube.GetArtistSongs(ID)

			if ArtistFetchErr != nil {

				return nil, -1, ArtistFetchErr

			}

			if len(ArtistSongs) == 0 {

				return nil, -1, errors.New("artist contains no songs")

			}

			SongFound = &ArtistSongs[0]

			for i, Song := range ArtistSongs {

				Pos := G.Queue.Add(&Song, Requestor)

				if i == 0 {

					PosAdded = Pos 
					
				}

			}

		case APIs.URITypePlaylist:

			PlaylistSongs, PlaylistFetchErr := Innertube.GetPlaylistSongs(ID)

			if PlaylistFetchErr != nil {

				return nil, -1, PlaylistFetchErr

			}

			if len(PlaylistSongs) == 0 {

				return nil, -1, errors.New("playlist contains no songs")

			}

			SongFound = &PlaylistSongs[0]

			for i, Song := range PlaylistSongs {

				Pos := G.Queue.Add(&Song, Requestor)

				if i == 0 {

					PosAdded = Pos

				}

			}

		case APIs.URITypeSPSong:

			ResolvedSong, _, SpotifyFetchErr := Spotify.SpotifyIDToSong(ID)

			if SpotifyFetchErr != nil {

				return nil, -1, SpotifyFetchErr

			}

			SongFound = &ResolvedSong

			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeSPAlbum:

			FirstSong, SpotifyAlbum, FirstSongError := Spotify.SpotifyAlbumToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest 

				AllOtherSongs, _, OtherFetchError := Spotify.SpotifyAlbumToAllSongs(SpotifyAlbum, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error(fmt.Sprintf("Error fetching rest of Spotify album songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Title", G.Locale.Code(), SpotifyAlbum.Name),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Description", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code())),

				})).Build())

			}()

		case APIs.URITypeSPPlaylist:

			FirstSong, SpotifyPlaylist, FirstSongError := Spotify.SpotifyPlaylistToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest. same as album

				AllOtherSongs, _, OtherFetchError := Spotify.SpotifyPlaylistToAllSongs(SpotifyPlaylist, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error(fmt.Sprintf("Error fetching rest of Spotify playlist songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
					AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Title", G.Locale.Code(), SpotifyPlaylist.Name),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Description", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code())),

					})).Build())

			}()

		case APIs.URITypeAMSong:

			ResolvedSong, _, AppleMusicFetchErr := Apple.AppleMusicIDToSong(ID)

			if AppleMusicFetchErr != nil {

				return nil, -1, AppleMusicFetchErr

			}

			SongFound = &ResolvedSong

			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeAMAlbum:

			FirstSong, AppleMusicAlbum, FirstSongError := Apple.AppleMusicAlbumToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest

				AllOtherSongs, _, OtherFetchError := Apple.AppleMusicAlbumToAllSongs(AppleMusicAlbum, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error(fmt.Sprintf("Error fetching rest of Apple Music album songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Title", G.Locale.Code(), AppleMusicAlbum.Attributes.Name),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Description", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code())),

				})).Build())

			}()

		case APIs.URITypeAMPlaylist:

			FirstSong, AppleMusicPlaylist, FirstSongError := Apple.AppleMusicPlaylistToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest

				AllOtherSongs, _, OtherFetchError := Apple.AppleMusicPlaylistToAllSongs(AppleMusicPlaylist, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error(fmt.Sprintf("Error fetching rest of Apple Music playlist songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
					AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Title", G.Locale.Code(), AppleMusicPlaylist.Attributes.Name),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedToQueue.Description", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code())),

					})).Build())

			}()

	}
		
	if (PosAdded == 0 && G.Queue.State != StatePlaying) {

		go func() { // done as to not block

			PlayError := G.Play(SongFound)

			if PlayError != nil {

				Utils.Logger.Error(fmt.Sprintf("Error playing song: %s", PlayError.Error()))

			}

		}()

	}

	return SongFound, PosAdded, nil

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

	OnStreamingError := func() {

		Utils.Logger.Error(fmt.Sprintf("Streaming error for song: %s, disconnecting guild", Song.Title))

		Locale := G.Locale.Code()

		go func() {

			_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
				AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.StreamingError.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Error", Locale),
					Description: Localizations.Get("Embeds.Notifications.StreamingError.Description", Locale),
					Color:       0xFFB3BA,

				})).
				Build())

			if ErrorSending != nil {

				Utils.Logger.Error(fmt.Sprintf("Error sending streaming error message to guild %s: %s", G.ID.String(), ErrorSending.Error()))

			}

		}()

		// Disconnect and clean up
		G.Disconnect(true)

	}

	Playback, ErrorCreatingPlayback := Audio.Play(Segments, SegmentDur, OnFinished, G.Queue.SendToWebsockets, OnStreamingError)

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

	// Stop inactivity timer when playback starts
	G.StopInactivityTimer()

	return nil

}