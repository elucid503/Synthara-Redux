package Innertube

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

	YouTubeID string `json:"youtube_id"`

	Title   string   `json:"title"`
	
	Artists []string `json:"artists"`

	Album   string   `json:"album"`
	AlbumID string   `json:"album_id"`

	Duration Duration `json:"duration"`

	Cover string `json:"cover"`

	Internal SongInternal `json:"-"`
		
}

type Duration struct {

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

	Playlist PlaylistMeta `json:"playlist"`

}

type PlaylistMeta struct {

	Platform string `json:"platform"`

	Index int `json:"index"`
	Total int `json:"total"`

	Name string `json:"name"`
	ID  string `json:"id"`

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

		RemoveButton := discord.NewButton(discord.ButtonStyleDanger, Localizations.Get("Buttons.RemoveSong", State.Locale), fmt.Sprintf("RemoveSong:%s", S.YouTubeID), "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.Trash)),

		})

		Buttons = append(Buttons, RemoveButton)

		JumpToButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Buttons.JumpToSong", State.Locale), fmt.Sprintf("JumpToSong:%s", S.YouTubeID), "", 0).WithEmoji(discord.ComponentEmoji{

			ID: snowflake.MustParse(Icons.GetID(Icons.Play)),

		})

		Buttons = append(Buttons, JumpToButton)

	}

	return Buttons

}