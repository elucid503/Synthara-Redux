package Tidal

import (
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type Song struct {

	TidalID int64 `json:"tidal_id"`

	Title   string   `json:"title"`
	
	Artists []string `json:"artists"`

	Album   string `json:"album"`
	AlbumID int64  `json:"album_id"`

	Duration SongDuration `json:"duration"`

	Cover string `json:"cover"`

	MixID string `json:"mix_id"` // For AutoPlay (TRACK_MIX)

	Internal SongInternal `json:"-"`
		
}

type SongDuration struct {

	Seconds   int    `json:"seconds"`
	Formatted string `json:"formatted"`

}

type QueueInfo struct {

	Playing bool `json:"playing"`
	
	GuildID snowflake.ID `json:"guild_id"`

	SongPosition  int `json:"song_position"`
	TotalUpcoming int `json:"total_upcoming"`
	TotalPrevious int `json:"total_previous"`

	Locale string `json:"locale"`
	
}

type SongInternal struct {

	Requestor string `json:"requestor"`

	Suggested bool `json:"suggested"`

	Playlist PlaylistMeta `json:"playlist"`

}

type PlaylistMeta struct {

	Platform string `json:"platform"`

	Index int `json:"index"`
	Total int `json:"total"`

	Name string `json:"name"`
	ID   string `json:"id"`

}

// FormatDuration converts seconds to "M:SS" format
func FormatDuration(Seconds int) string {

	Minutes := Seconds / 60
	Secs := Seconds % 60

	return fmt.Sprintf("%d:%02d", Minutes, Secs)

}

// TrackToSong converts a Tidal Track to a Song
func TrackToSong(Track Track) Song {

	Artists := make([]string, 0, len(Track.Artists))
	for _, Artist := range Track.Artists {
		Artists = append(Artists, Artist.Name)
	}

	Cover := ""
	if Track.Album.Cover != "" {
		Cover = fmt.Sprintf("https://resources.tidal.com/images/%s/640x640.jpg", 
			ReplaceHyphens(Track.Album.Cover))
	}

	return Song{
		TidalID:  Track.ID,
		Title:    Track.Title,
		Artists:  Artists,
		Album:    Track.Album.Title,
		AlbumID:  Track.Album.ID,
		Cover:    Cover,
		MixID:    Track.Mixes.TrackMix,
		Duration: SongDuration{
			Seconds:   Track.Duration,
			Formatted: FormatDuration(Track.Duration),
		},
	}
}

// InfoToSong converts a Tidal Info to a Song
func InfoToSong(Info Info) Song {

	Artists := make([]string, 0, len(Info.Artists))

	for _, Artist := range Info.Artists {

		Artists = append(Artists, Artist.Name)
		
	}

	Cover := ""

	if Info.Album.Cover != "" {

		Cover = fmt.Sprintf("https://resources.tidal.com/images/%s/640x640.jpg", ReplaceHyphens(Info.Album.Cover))

	}

	return Song{

		TidalID:  Info.ID,
		
		Title:    Info.Title,
		
		Artists:  Artists,

		Album:    Info.Album.Title,
		AlbumID:  Info.Album.ID,

		Cover:    Cover,

		MixID:    Info.Mixes.TrackMix,

		Duration: SongDuration{

			Seconds:   Info.Duration,
			Formatted: FormatDuration(Info.Duration),
			
		},

	}

}

func ReplaceHyphens(s string) string {

	result := ""

	for _, c := range s {

		if c == '-' {

			result += "/"

		} else {

			result += string(c)

		}

	}
	
	return result

}

func (S *Song) Embed(State QueueInfo) discord.Embed {

	Locale := State.Locale

	if Locale == "" {

		Locale = Localizations.Default

	}

	Embed := discord.NewEmbedBuilder()

	Embed.SetTitle(S.Title)

	AuthorName := Localizations.Get("Embeds.NowPlaying.AuthorNowPlaying", Locale)
	AddedState := Localizations.Get("Embeds.NowPlaying.StatePlayedBy", Locale)

	if State.SongPosition > 0 { 

		SongWord := Localizations.Pluralize("Song", State.SongPosition, Locale)
		AuthorName = Localizations.GetFormat("Embeds.NowPlaying.AuthorSongsAway", Locale, State.SongPosition, SongWord)
		AddedState = Localizations.Get("Embeds.NowPlaying.StateEnqueuedBy", Locale)

	}

	Embed.SetAuthor(AuthorName, "", "")

	// Joins artist names using ", "

	ArtistNames := ""

	for i, Artist := range S.Artists {

		ArtistNames += Artist

		if i < (len(S.Artists) - 1) {

			ArtistNames += ", "

		}

	}

	Page := fmt.Sprintf("%s/Queues/%s", os.Getenv("DOMAIN"), State.GuildID.String()) 
	Embed.SetURL(Page)

	Embed.SetThumbnail(S.Cover)

	Description := Localizations.GetFormat("Embeds.NowPlaying.DescriptionOnAlbum", Locale, S.Album)

	if (S.Internal.Playlist.Index >= 0) && (S.Internal.Playlist.Total > 0) {

		Description += "\n" + Localizations.GetFormat("Embeds.NowPlaying.DescriptionInPlaylist", Locale, S.Internal.Playlist.Index + 1, S.Internal.Playlist.Total, S.Internal.Playlist.Name)
		
	}

	if S.Internal.Suggested {

		Description += "\n" + Localizations.Get("Embeds.NowPlaying.SuggestedSong", Locale)

	}

	Embed.SetDescription(Description)

	Embed.AddField(Localizations.Get("Embeds.NowPlaying.FieldArtists", Locale), ArtistNames, true)
	Embed.AddField(Localizations.Get("Embeds.NowPlaying.FieldDuration", Locale), Localizations.GetFormat("Embeds.NowPlaying.DurationFormat", Locale, S.Duration.Formatted), true)
	Embed.AddField(AddedState, S.Internal.Requestor, true)
	
	// Color 

	DominantColor, ColorFetchError := Utils.GetDominantColorHex(S.Cover)

	if ColorFetchError != nil {

		Utils.Logger.Warn(fmt.Sprintf("Failed to get dominant color for song embed: %s", ColorFetchError.Error()))
		
	}

	Embed.SetColor(DominantColor)

	return Embed.Build()

}

func (S *Song) Buttons(State QueueInfo) []discord.InteractiveComponent {

	// Different buttons for now playing vs queued songs

	Buttons := []discord.InteractiveComponent{}

	if State.SongPosition == 0 {

		PlayPauseIcon := Icons.Play
		PlayPauseID := "Play"

		if (State.Playing) {

			PlayPauseIcon = Icons.Pause
			PlayPauseID = "Pause"

		}

		// Now playing buttons

		LastButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Last", "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipBack)),

		})

		Buttons = append(Buttons, LastButton)

		LyricsButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Lyrics", "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.ChatBubbles)),

		})

		Buttons = append(Buttons, LyricsButton)

		PlayPauseButton := discord.NewButton(discord.ButtonStyleSecondary, "", PlayPauseID, "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(PlayPauseIcon)),

		})

		Buttons = append(Buttons, PlayPauseButton)

		QueueButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Queue", "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.Albums)),

		})

		Buttons = append(Buttons, QueueButton)

		NextButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Next", "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipForward)),

		})

		Buttons = append(Buttons, NextButton)

	} else {

		// Queued song buttons

		RemoveButton := discord.NewButton(discord.ButtonStyleDanger, Localizations.Get("Buttons.RemoveSong", State.Locale), fmt.Sprintf("RemoveSong:%d", S.TidalID), "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.Trash)),

		})

		Buttons = append(Buttons, RemoveButton)

		JumpToButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Buttons.JumpToSong", State.Locale), fmt.Sprintf("JumpToSong:%d", S.TidalID), "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.Play)),

		})

		Buttons = append(Buttons, JumpToButton)

	}

	return Buttons

}
