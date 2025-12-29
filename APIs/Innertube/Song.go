package Innertube

import (
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

	Duration Duration `json:"duration"`

	Cover string `json:"cover"`

	Internal SongInternal `json:"-"`
		
}

type Duration struct {

	Seconds   int    `json:"seconds"`
	Formatted string `json:"formatted"`

}

type QueueInfo struct {

	GuildID snowflake.ID `json:"guild_id"`

	SongPosition int `json:"song_position"`
	TotalUpcoming  int `json:"total_upcoming"`
	TotalPrevious int `json:"total_previous"`
	
}

type SongInternal struct {

	Requestor string `json:"requestor"`

}

func (S *Song) Embed(State QueueInfo) discord.Embed {

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

	Page := fmt.Sprintf("%s/Queues/%s", os.Getenv("DOMAIN"), State.GuildID.String()) 
	Embed.SetURL(Page)

	Embed.SetThumbnail(S.Cover)

	Description := fmt.Sprintf("On **%s**", S.Album)

	Embed.SetDescription(Description)

	Embed.AddField("Artists", ArtistNames, true)
	Embed.AddField("Duration", fmt.Sprintf("%s Min", S.Duration.Formatted), true)
	Embed.AddField(fmt.Sprintf("%s By", AddedState), S.Internal.Requestor, true)
	
	// Embed.SetFooter(fmt.Sprintf("%d %s in Queue • %d %s Played", State.TotalUpcoming, Utils.Pluralize("Song", State.TotalUpcoming), State.TotalPrevious, Utils.Pluralize("Song", State.TotalPrevious)), "") // no footer IconURL
		
	// Color 

	DominantColor, ColorFetchError := Utils.GetDominantColorHex(S.Cover)

	if ColorFetchError != nil {

		Utils.Logger.Warn(fmt.Sprintf("Failed to get dominant color for song embed: %s", ColorFetchError.Error()))
		
	}

	Embed.SetColor(DominantColor)

	return Embed.Build()

}