package Innertube

import (
	"Synthara-Redux/Utils"
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
)

type Song struct {

	YouTubeID string `json:"youtube_id"`

	Title   string   `json:"title"`
	Artists []string `json:"artists"`
	Album   string   `json:"album"`

	Duration Duration `json:"duration"`

	Cover string `json:"cover"`
	
	HLSManifest string `json:"-"`
	
}

type Duration struct {

	Seconds   int    `json:"seconds"`
	Formatted string `json:"formatted"`

}

func (S *Song) Embed(Requestor *discord.User, PosInQueue int) discord.Embed {

	Embed := discord.NewEmbedBuilder()

	Embed.SetTitle(S.Title)

	AuthorName := "Now Playing"
	AddedState := "Played"

	if PosInQueue > 0 { 

		AuthorName = fmt.Sprintf("%d %s Away", PosInQueue, Utils.Pluralize("Song", PosInQueue))
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

	Embed.SetDescription(fmt.Sprintf("By **%s&&", ArtistNames))

	Embed.AddField("Duration", fmt.Sprintf("%s Min", S.Duration.Formatted), true)
	Embed.AddField(fmt.Sprintf("%s By", AddedState), Requestor.Username, true)

	Embed.SetFooter(os.Getenv("EMBED_FOOTER_TEXT"), "")

	// Color 

	DominantColor, ColorFetchError := Utils.GetDominantColorHex(S.Cover)

	if ColorFetchError != nil {

		Utils.Logger.Warn(fmt.Sprintf("Failed to get dominant color for song embed: %s", ColorFetchError.Error()))
		
	}

	Embed.SetColor(DominantColor)

	return Embed.Build()

}