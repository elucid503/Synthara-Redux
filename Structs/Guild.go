package Structs

import (
	"Synthara-Redux/APIs"
	"Synthara-Redux/APIs/Apple"
	"Synthara-Redux/APIs/Spotify"
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/APIs/YouTube"
	"Synthara-Redux/Audio"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
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

			Previous: []*Tidal.Song{},
			Current:  nil,
			Upcoming: []*Tidal.Song{},

			Suggestions: []*Tidal.Song{},

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
	defer func() {
		if r := recover(); r != nil {
			Utils.Logger.Error("Guild", fmt.Sprintf("Panic in Guild.Connect for guild %s: %v", G.ID.String(), r))
		}
	}()

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

		CloseContext, CloseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer CloseCancel()
		VoiceConnection.Close(CloseContext)

		return ErrorOpening

	}

	G.VoiceConnection = VoiceConnection

	return nil;

}
 
// Disconnect Closes the existing voice connection; if none exists, returns an error
func (G *Guild) Disconnect(CloseConn bool) error {
	defer func() {
		if r := recover(); r != nil {
			Utils.Logger.Error("Guild", fmt.Sprintf("Panic in Guild.Disconnect for guild %s: %v", G.ID.String(), r))
		}
	}()

	G.Internal.Disconnecting = true

	return G.Cleanup(CloseConn)

}

// Stops playback, closes websockets, and the voice connection
func (G *Guild) Cleanup(CloseConn bool) error {

	Utils.Logger.Info("Guild", fmt.Sprintf("Cleanup requested for guild: %s", G.ID.String()))

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	Utils.Logger.Info("Guild", fmt.Sprintf("Cleaning up guild: %s", G.ID.String()))

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

		// Store reference and nil it immediately to prevent race conditions
		VoiceConn := G.VoiceConnection
		G.VoiceConnection = nil

		// Safely close the connection with error recovery
		_ = VoiceConn.SetSpeaking(ContextToUse, 0)

		VoiceConn.SetOpusFrameProvider(nil)

		// Recover from any panics during close

		func() {

			defer func() {

				if r := recover(); r != nil {

					Utils.Logger.Error("Guild", fmt.Sprintf("Panic during voice connection close for guild %s: %v", G.ID.String(), r))

				}

			}()

			VoiceConn.Close(ContextToUse)

		}()

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

	Utils.Logger.Info("Guild", fmt.Sprintf("Starting inactivity timer for guild %s with duration: %s", G.ID.String(), Duration.String()))

	// Create new timer
	G.Internal.InactivityTimer = time.AfterFunc(Duration, func() {

		Utils.Logger.Info("Guild", fmt.Sprintf("Inactivity timer expired for guild %s, disconnecting...", G.ID.String()))

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

				Utils.Logger.Error("Command", fmt.Sprintf("Error sending inactivity disconnect message to guild %s: %s", G.ID.String(), ErrorSending.Error()))

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

		Utils.Logger.Info("Guild", fmt.Sprintf("Resetting inactivity timer for guild %s", G.ID.String()))

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

		Utils.Logger.Info("Guild", fmt.Sprintf("Stopping inactivity timer for guild %s", G.ID.String()))

		G.Internal.InactivityTimer.Stop()
		G.Internal.InactivityTimer = nil

	}

}

// RouteURI takes a Synthara-Redux URI string and handles adding/playing the content. Returns the song, its position in the queue, and any error
func (G *Guild) HandleURI(URI string, Requestor string) (*Tidal.Song, int, error) {

	Type, ID, ErrorParsing := APIs.ParseURI(URI)

	if ErrorParsing != nil {

		return nil, -1, ErrorParsing

	}

	var PosAdded int
	var SongFound *Tidal.Song

	switch Type {

		// Plain-text search

		case APIs.URITypeNone:

			SearchResults, SearchErr := Tidal.SearchSongs(ID)

			if SearchErr != nil || len(SearchResults) == 0 {

				return nil, -1, errors.New("no search results found")

			}

			SongFound = &SearchResults[0]

			PosAdded = G.Queue.Add(SongFound, Requestor)

		// Tidal (Default)

		case APIs.URITypeTidalSong:

			// Internal Tidal song ID - fetch directly by ID

			TidalID, ParseErr := strconv.ParseInt(ID, 10, 64)
			
			if ParseErr != nil {

				return nil, -1, fmt.Errorf("invalid Tidal ID: %s", ID)

			}

			FetchedSong, FetchErr := Tidal.GetSong(TidalID)
			
			if FetchErr != nil {

				return nil, -1, FetchErr

			}

			SongFound = &FetchedSong
			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeTidalAlbum:

			// Tidal album - fetch all tracks

			AlbumID, ParseErr := strconv.ParseInt(ID, 10, 64)
			
			if ParseErr != nil {

				return nil, -1, fmt.Errorf("invalid Tidal album ID: %s", ID)

			}

			AlbumTracks, FetchErr := Tidal.FetchAlbumTracks(AlbumID)
			
			if FetchErr != nil || len(AlbumTracks) == 0 {

				return nil, -1, errors.New("could not fetch album tracks")

			}

			SongFound = &AlbumTracks[0]
			PosAdded = G.Queue.Add(SongFound, Requestor)

			// Add rest of tracks in background

			if len(AlbumTracks) > 1 {

				go func() {

					for _, Song := range AlbumTracks[1:] {

						SongCopy := Song
						G.Queue.Add(&SongCopy, Requestor)

					}

					Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromAlbum", G.Locale.Code(), len(AlbumTracks), Localizations.Pluralize("Song", len(AlbumTracks), G.Locale.Code()), AlbumTracks[0].Album),

					})).Build())

				}()
				
			}

		case APIs.URITypeFavorites:

			User, UserErr := GetUser(ID) // ID from URI is UserID
			if UserErr != nil {
				return nil, -1, UserErr
			}

			FavURIs := User.GetTopFavorites(10)
			if len(FavURIs) == 0 {
				return nil, -1, errors.New("no favorites found")
			}

			Songs := []Tidal.Song{}
			for _, favURI := range FavURIs {
				// Parse and resolve each favorite
				Type, ResID, pErr := APIs.ParseURI(favURI)
				if pErr == nil && Type == APIs.URITypeTidalSong {
					tid, _ := strconv.ParseInt(ResID, 10, 64)
					if s, err := Tidal.GetSong(tid); err == nil {
						Songs = append(Songs, s)
					}
				}
			}

			if len(Songs) == 0 {
				return nil, -1, errors.New("could not resolve favorites")
			}

			// Set Playlist Metadata
			PlaylistMeta := Tidal.PlaylistMeta{
				Name:     "Favorites",
				Platform: "System",
				Total:    len(Songs),
				ID:       "favorites:" + ID,
			}

			for i := range Songs {
				Songs[i].Internal.Playlist = PlaylistMeta
				Songs[i].Internal.Playlist.Index = i
			}

			SongFound = &Songs[0]
			PosAdded = G.Queue.Add(SongFound, Requestor)

			if len(Songs) > 1 {
				go func() {
					for _, Song := range Songs[1:] {
						SongCopy := Song
						G.Queue.Add(&SongCopy, Requestor)
					}
					// Optional: Send notification about "Playlist" added
				}()
			}

		case APIs.URITypeSuggestions:

			User, UserErr := GetUser(ID)
			if UserErr != nil {
				return nil, -1, UserErr
			}

			if User.MostRecentMix == "" {
				return nil, -1, errors.New("no recent mix found")
			}

			Songs, FetchErr := Tidal.FetchMixItems(User.MostRecentMix)
			if FetchErr != nil || len(Songs) == 0 {
				return nil, -1, errors.New("could not fetch suggestions")
			}

			if len(Songs) > 10 {
				Songs = Songs[:10]
			}

			// Set Playlist Metadata
			PlaylistMeta := Tidal.PlaylistMeta{
				Name:     "Suggestions",
				Platform: "System",
				Total:    len(Songs),
				ID:       "suggestions:" + ID,
			}

			for i := range Songs {
				Songs[i].Internal.Playlist = PlaylistMeta
				Songs[i].Internal.Playlist.Index = i
			}

			SongFound = &Songs[0]
			PosAdded = G.Queue.Add(SongFound, Requestor)

			if len(Songs) > 1 {
				go func() {
					for _, Song := range Songs[1:] {
						SongCopy := Song
						G.Queue.Add(&SongCopy, Requestor)
					}
				}()
			}


		case APIs.URITypeTidalPlaylist:

			// Tidal playlist - fetch all tracks

			PlaylistID, ParseErr := strconv.ParseInt(ID, 10, 64)

			if ParseErr != nil {

				return nil, -1, fmt.Errorf("invalid Tidal playlist ID: %s", ID)

			}

			PlaylistTracks, FetchErr := Tidal.FetchPlaylistTracks(PlaylistID)

			if FetchErr != nil || len(PlaylistTracks) == 0 {

				return nil, -1, errors.New("could not fetch playlist tracks")

			}

			SongFound = &PlaylistTracks[0]
			PosAdded = G.Queue.Add(SongFound, Requestor)

			// Add rest of tracks in background

			if len(PlaylistTracks) > 1 {

				go func() {

					for _, Song := range PlaylistTracks[1:] {

						SongCopy := Song

						G.Queue.Add(&SongCopy, Requestor)

					}

					Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromPlaylist", G.Locale.Code(), len(PlaylistTracks), Localizations.Pluralize("Song", len(PlaylistTracks), G.Locale.Code()), "Playlist"),

					})).Build())

				}()

			}

		// YouTube

		case APIs.URITypeYouTubeVideo:

			ResolvedSong, _, YouTubeFetchErr := YouTube.VideoIDToSong(ID)

			if YouTubeFetchErr != nil {

				return nil, -1, YouTubeFetchErr

			}

			SongFound = &ResolvedSong

			PosAdded = G.Queue.Add(SongFound, Requestor)

		case APIs.URITypeYouTubePlaylist:

			FirstSong, YouTubePlaylist, FirstSongError := YouTube.PlaylistIDToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest of playlist

				AllOtherSongs, FailedCount, _, OtherFetchError := YouTube.PlaylistIDToAllSongs(YouTubePlaylist, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error("YouTube Fetch", fmt.Sprintf("Error fetching rest of YouTube playlist songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				TotalVideos := len(YouTubePlaylist.Videos)
				SuccessCount := len(AllOtherSongs) + 1 // +1 for first song

				PlaylistTitle := YouTubePlaylist.Title

				if PlaylistTitle == "" {

					PlaylistTitle = Localizations.Get("Common.Playlist", G.Locale.Code())

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromPlaylist", G.Locale.Code(), SuccessCount, Localizations.Pluralize("Song", SuccessCount, G.Locale.Code()), PlaylistTitle),

				})).Build())

				// If some songs failed, send an additional notification
				if FailedCount > 0 {

					Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.SomePlaylistSongsFailed.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.SomePlaylistSongsFailed.Description", G.Locale.Code(), FailedCount, TotalVideos, Localizations.Pluralize("Song", TotalVideos, G.Locale.Code()), PlaylistTitle),

					})).Build())

				}

			}()

		
		// YouTube Music

		case APIs.URITypeYTMusicAlbum:

			FirstSong, YouTubeMusicAlbum, FirstSongError := YouTube.MusicAlbumIDToFirstSong(ID)

			if FirstSongError != nil {

				return nil, -1, FirstSongError

			}

			SongFound = &FirstSong

			PosAdded = G.Queue.Add(&FirstSong, Requestor)

			go func() {

				// Add rest of album

				AllOtherSongs, FailedCount, _, OtherFetchError := YouTube.MusicAlbumIDToAllSongs(YouTubeMusicAlbum, true) // ignores first

				if OtherFetchError != nil {

					Utils.Logger.Error("YouTube Fetch", fmt.Sprintf("Error fetching rest of YouTube Music album songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				// Calculate total videos (including first one)
				TotalVideos := len(YouTubeMusicAlbum.Videos)
				SuccessCount := len(AllOtherSongs) + 1 // +1 for first song

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromAlbum", G.Locale.Code(), SuccessCount, Localizations.Pluralize("Song", SuccessCount, G.Locale.Code()), YouTubeMusicAlbum.Title),

				})).Build())

				// If some songs failed, send an additional notification
				if FailedCount > 0 {

					Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.SomePlaylistSongsFailed.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.SomePlaylistSongsFailed.Description", G.Locale.Code(), FailedCount, TotalVideos, Localizations.Pluralize("Song", TotalVideos, G.Locale.Code()), YouTubeMusicAlbum.Title),

					})).Build())

				}

			}()

		case APIs.URITypeYTMusicArtist:

			ArtistSongs, ArtistFetchErr := YouTube.MusicArtistIDToSongs(ID)

			if ArtistFetchErr != nil || len(ArtistSongs) == 0 {

				return nil, -1, errors.New("could not fetch artist songs")

			}

			SongFound = &ArtistSongs[0]

			PosAdded = G.Queue.Add(SongFound, Requestor)

			// Add rest of artist songs in background

			if len(ArtistSongs) > 1 {

				go func() {

					for _, Song := range ArtistSongs[1:] {

						SongCopy := Song
						G.Queue.Add(&SongCopy, Requestor)

					}

					Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromArtist", G.Locale.Code(), len(ArtistSongs), Localizations.Pluralize("Song", len(ArtistSongs), G.Locale.Code()), ArtistSongs[0].Artists[0]),

					})).Build())

				}()

			}
		
		// Spotify
		
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

					Utils.Logger.Error("Spotify Fetch", fmt.Sprintf("Error fetching rest of Spotify album songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromAlbum", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code()), SpotifyAlbum.Name),

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

					Utils.Logger.Error("Spotify Fetch", fmt.Sprintf("Error fetching rest of Spotify playlist songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
					AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromPlaylist", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code()), SpotifyPlaylist.Name),

					})).Build())

			}()

		// Apple Music

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

					Utils.Logger.Error("Apple Music Fetch", fmt.Sprintf("Error fetching rest of Apple Music album songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
					Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
					Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromAlbum", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code()), AppleMusicAlbum.Attributes.Name),

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

					Utils.Logger.Error("Apple Music Fetch", fmt.Sprintf("Error fetching rest of Apple Music playlist songs: %s", OtherFetchError.Error()))
					return

				}

				for _, Song := range AllOtherSongs {

					G.Queue.Add(&Song, Requestor)

				}

				Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
					AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Embeds.Notifications.AddedAdditionalSongs.Title", G.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", G.Locale.Code()),
						Description: Localizations.GetFormat("Embeds.Notifications.AddedAdditionalSongs.FromPlaylist", G.Locale.Code(), len(AllOtherSongs)+1, Localizations.Pluralize("Song", len(AllOtherSongs)+1, G.Locale.Code()), AppleMusicPlaylist.Attributes.Name),

					})).Build())

			}()

	}

	// Auto-plays if first in queue and not already playing
		
	if (PosAdded == 0 && G.Queue.State != StatePlaying) {

		go func() { // done as to not block

			PlayError := G.Play(SongFound)

			if PlayError != nil {

				Utils.Logger.Error("Playback", fmt.Sprintf("Error playing song: %s", PlayError.Error()))

			}

		}()

	}

	return SongFound, PosAdded, nil

}

// Play starts playing the song using Tidal streaming
func (G *Guild) Play(Song *Tidal.Song) error {

	if Song == nil {

		return fmt.Errorf("cannot play nil song")
		
	}

	defer func() {

		if r := recover(); r != nil {

			SongTitle := "unknown"

			if Song != nil {

				SongTitle = Song.Title

			}

			Utils.Logger.Error("Playback", fmt.Sprintf("Panic in Guild.Play for song %s", SongTitle))

			buf := make([]byte, 1<<16) // 64kb buffer
			runtime.Stack(buf, true)
			
			Utils.Logger.Error("Playback", fmt.Sprintf("Stack trace: %s", string(buf)))

			G.Queue.SetState(StateIdle)
			G.Queue.PlaybackSession = nil

		}

	}()

	// Get stream URL from Tidal
	StreamURL, ErrorFetchingStream := Tidal.GetStreamURL(Song.TidalID)

	if ErrorFetchingStream != nil {

		return ErrorFetchingStream

	}

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.VoiceConnection == nil {

		return fmt.Errorf("voice connection closed")
		
	}

	if G.Queue.PlaybackSession != nil {

		G.Queue.PlaybackSession.Stop()
		G.Queue.PlaybackSession = nil

	}

	var Playback *Audio.MP4Playback

	OnFinished := func() {

		defer func() {

			if r := recover(); r != nil {

				Utils.Logger.Error("Playback", fmt.Sprintf("Panic in OnFinished: %v", r))
			}
			
		}()

		Utils.Logger.Info("Playback", fmt.Sprintf("Playback finished for song: %s", Song.Title))

		G.StreamerMutex.Lock()

		if G.VoiceConnection == nil {
			
			G.StreamerMutex.Unlock()
			return 
			
		}

		// Double-check we aren't interfering with a new session
		if Playback != nil && G.Queue.PlaybackSession != Playback {
			
			G.StreamerMutex.Unlock()
			return
			
		}

		G.Queue.PlaybackSession = nil
		G.Queue.SetState(StateIdle)

		// Capture connection to use outside lock to prevent deadlocks if connection is zombie
		VoiceConnection := G.VoiceConnection
		G.StreamerMutex.Unlock()

		if VoiceConnection != nil {

			VoiceConnection.SetOpusFrameProvider(nil)

			ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
			defer CancelFunc()

			_ = VoiceConnection.SetSpeaking(ContextToUse, 0)

		}

	}

	OnStreamingError := func() {

		Utils.Logger.Error("Streaming", fmt.Sprintf("Streaming error for song: %s, disconnecting guild", Song.Title))

		Locale := G.Locale.Code()

		go func() {

			_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(G.Channels.Text, discord.NewMessageCreateBuilder().
				AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.StreamingError.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Error", Locale),
					Description: Localizations.Get("Embeds.Notifications.StreamingError.Description", Locale),
					Color:       Utils.ERROR,

				})).
				Build())

			if ErrorSending != nil {

				Utils.Logger.Error("Command", fmt.Sprintf("Error sending streaming error message to guild %s: %s", G.ID.String(), ErrorSending.Error()))

			}

		}()

		// Disconnect and clean up
		G.Disconnect(true)

	}

	var ErrorCreatingPlayback error
	Playback, ErrorCreatingPlayback = Audio.PlayMP4(StreamURL, OnFinished, G.Queue.SendToWebsockets, OnStreamingError)

	if ErrorCreatingPlayback != nil {

		return ErrorCreatingPlayback

	}

	Provider := &Audio.MP4OpusProvider{

		Streamer: Playback.Streamer,

	}

	if G.VoiceConnection == nil {

		Playback.Stop()
		return fmt.Errorf("voice connection closed before setting provider")

	}

	G.VoiceConnection.SetOpusFrameProvider(Provider)

	if G.VoiceConnection == nil {

		Playback.Stop()
		return fmt.Errorf("voice connection closed after setting provider")

	}

	ContextToUse, CancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
	defer CancelFunc()

	ErrorSpeaking := G.VoiceConnection.SetSpeaking(ContextToUse, 1)

	if ErrorSpeaking != nil {

		Playback.Stop()
		return ErrorSpeaking

	}

	G.Queue.SetState(StatePlaying)
	G.Queue.PlaybackSession = Playback

	// Start progress update ticker

    go func() {
        Playback := Playback

        if Playback == nil {
            return
        }

        Ticker := time.NewTicker(5 * time.Second)
        defer Ticker.Stop()

        for range Ticker.C {

            G.StreamerMutex.Lock()

            if G.Queue.PlaybackSession != Playback || Playback.Stopped.Load() || G.Internal.Disconnecting {

                G.StreamerMutex.Unlock()
                return
				
            }

            if Playback.Streamer == nil {
                G.StreamerMutex.Unlock()
                continue
            }

            Progress := Playback.Streamer.Progress

            G.StreamerMutex.Unlock()

            G.Queue.SendToWebsockets(Event_ProgressUpdate, map[string]any{"Progress": Progress})

        }
        
    }()

	// Stop inactivity timer when playback starts (unless autoplay is enabled)

	if !G.Features.Autoplay {

		G.StopInactivityTimer()

	}

	return nil

}