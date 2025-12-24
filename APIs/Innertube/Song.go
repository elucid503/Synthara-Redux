package Innertube

import (
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
)

type Song struct {

	YouTubeID string `json:"youtube_id"`

	Title   string   `json:"title"`
	Artists []string `json:"artists"`
	Album   string   `json:"album"`

	Duration Duration `json:"duration"`

	Cover string `json:"cover"`
		
}

type Duration struct {

	Seconds   int    `json:"seconds"`
	Formatted string `json:"formatted"`

}

type QueueInfo struct {

	SongPosition int `json:"song_position"`
	TotalUpcoming  int `json:"total_upcoming"`
	TotalPrevious int `json:"total_previous"`
	
	TimePlaying int `json:"time_previous"`

}

func (S *Song) Embed(Requestor *discord.User, State QueueInfo) discord.Embed {

	Embed := discord.NewEmbedBuilder()

	Embed.SetTitle(S.Title)

	AuthorName := "Now Playing"
	AddedState := "Played"

	if State.SongPosition > 0 { 

		AuthorName = fmt.Sprintf("%d %s Away", State.SongPosition, Utils.Pluralize("Song", State.SongPosition))
		AddedState = "Enqueued"

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

	Embed.SetThumbnail(S.Cover)

	Embed.SetDescription(fmt.Sprintf("By **%s**", ArtistNames))

	Embed.AddField("Duration", fmt.Sprintf("%s Min", S.Duration.Formatted), true)
	Embed.AddField(fmt.Sprintf("%s By", AddedState), Requestor.Username, true)

	TotalSongs := State.TotalPrevious + State.TotalUpcoming + 1
	CurrentPosition := State.SongPosition + 1
	
	if State.TimePlaying > 0 { // only shows time playing if greater than 0

		Embed.SetFooter(fmt.Sprintf("Song %d of %d • Playing for %s Min", CurrentPosition, TotalSongs, FormatDuration(State.TimePlaying)), "")

	} else {

		Embed.SetFooter(fmt.Sprintf("Song %d of %d • Connected", CurrentPosition, TotalSongs), "")
		
	}

	// Color 

	DominantColor, ColorFetchError := Utils.GetDominantColorHex(S.Cover)

	if ColorFetchError != nil {

		Utils.Logger.Warn(fmt.Sprintf("Failed to get dominant color for song embed: %s", ColorFetchError.Error()))
		
	}

	Embed.SetColor(DominantColor)

	return Embed.Build()

}