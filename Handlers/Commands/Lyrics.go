package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

type LyricsAPIResponse struct {

	Type     string `json:"type"`

	Metadata struct {

		Source string `json:"source"`
		SongWriters []string `json:"songWriters"`
		Language string `json:"language"`

	} `json:"metadata"`

	Lyrics []struct {

		Time     int    `json:"time"`
		Duration int    `json:"duration"`

		Text     string `json:"text"`

		Element  struct {

			SongPart string `json:"songPart"`
			Singer   string `json:"singer"`

		} `json:"element"`

	} `json:"lyrics"`

}

// Simple chunker for long strings
func ChunkString(S string, Size int) []string {

	if len(S) <= Size {

		return []string{S}

	}

	var Chunks []string

	for i := 0; i < len(S); i += Size {

		End := i + Size

		if End > len(S) {

			End = len(S)
			
		}

		Chunks = append(Chunks, S[i:End])

	}

	return Chunks
}

func LyricsFetcher(Title string, Artist string) (*LyricsAPIResponse, error) {

	Params := url.Values{}
	Params.Set("title", Title)

	if Artist != "" {

		Params.Set("artist", Artist)

	}

	Params.Set("source", "apple,lyricsplus,musixmatch,spotify,musixmatch-word")

	ReqURL := fmt.Sprintf("https://lyricsplus.prjktla.workers.dev/v2/lyrics/get?%s", Params.Encode())

	Ctx, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	Req, Err := http.NewRequestWithContext(Ctx, "GET", ReqURL, nil)

	if Err != nil {

		return nil, Err

	}

	Resp, Err := http.DefaultClient.Do(Req)

	if Err != nil {

		return nil, Err

	}
	
	defer Resp.Body.Close()

	if Resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("bad status: %d", Resp.StatusCode)

	}

	var APIResp LyricsAPIResponse

	Decoder := json.NewDecoder(Resp.Body)
	
	if Err := Decoder.Decode(&APIResp); Err != nil {

		return nil, Err

	}
	
	if len(APIResp.Lyrics) == 0 {

		return nil, fmt.Errorf("no lyrics found")

	}

	return &APIResp, nil

}

type LyricsResponse struct {

	Embeds []discord.Embed
	Buttons []discord.InteractiveComponent
	
}

func BuildLyricsResponse(GuildID snowflake.ID, Locale string) (*LyricsResponse, error) {

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || Guild.Queue.Current == nil {

		return nil, fmt.Errorf("no song playing")

	}

	Song := Guild.Queue.Current

	// Page for web lyrics view

	Page := fmt.Sprintf("%s/Queues/%s?View=Lyrics", os.Getenv("DOMAIN"), GuildID.String())

	// Cleans title similar to frontend

	Regex := regexp.MustCompile(`\s*\(.*?\)`)
	Cleaned := strings.TrimSpace(Regex.ReplaceAllString(Song.Title, ""))

	// Try cleaned title first, fall back to original title like the frontend

	Artist := ""

	if len(Song.Artists) > 0 {

		Artist = Song.Artists[0]

	}

	APIRespPtr, Err := LyricsFetcher(Cleaned, Artist)

	if Err != nil && Cleaned != Song.Title {

		// Retries with original title

		APIRespPtr, Err = LyricsFetcher(Song.Title, Artist)

	}

	if Err != nil || APIRespPtr == nil {

		return nil, fmt.Errorf("lyrics not found")

	}

	APIResp := *APIRespPtr

	// Builds plain-text lyrics grouped by song part

	var Parts []string
	PrevPart := ""

	for _, l := range APIResp.Lyrics {

		Part := l.Element.SongPart

		if Part != "" && Part != PrevPart {

			// separates sections

			Parts = append(Parts, "")
			Parts = append(Parts, strings.ToUpper(Part))

			PrevPart = Part

		}

		if l.Text != "" {

			Parts = append(Parts, fmt.Sprintf("> %s", l.Text)) // blockquote style

		}

	}

	LyricsText := strings.TrimSpace(strings.Join(Parts, "\n"))

	SongColor, _ := Utils.GetDominantColorHex(Song.Cover)

	// Determine which view link to use (word vs line)

	ViewKey := "Embeds.Lyrics.ViewLine"

	if strings.ToLower(APIResp.Type) == "word" {

		ViewKey = "Embeds.Lyrics.ViewWord"

	}

	ViewLabel := Localizations.Get(ViewKey, Locale)

	// Builds a single embed with truncation if necessary

	MaxDesc := 4000
	Desc := ""

	Writers := ""

	if len(APIResp.Metadata.SongWriters) > 0 {

		Writers = Localizations.GetFormat("Embeds.Lyrics.WrittenBy", Locale, strings.Join(APIResp.Metadata.SongWriters, ", ")) + "\n\n"

	} 

	Truncated := Localizations.Get("Embeds.Lyrics.Truncated", Locale)

	if len(Writers)+len(LyricsText) <= MaxDesc {

		Desc = Writers + LyricsText

	} else {

		suffix := "..." + "\n\n" + Truncated + "\n\n"
		Limit := MaxDesc - len(suffix)

		if Limit < 0 {

			Limit = 0

		}

		Prefix := Writers + LyricsText

		if len(Prefix) > Limit {

			Prefix = Prefix[:Limit]

		}

		Desc = strings.TrimSpace(Prefix) + suffix

	}

	Embed := discord.NewEmbedBuilder()

	Embed.SetURL(Page)
	Embed.SetTitle(Song.Title)
	Embed.SetThumbnail(Song.Cover)
	Embed.SetDescription(Desc)
	Embed.SetColor(SongColor)
	Embed.SetAuthor(Localizations.Get("Embeds.Lyrics.Title", Locale), "", "")

	return &LyricsResponse{
		Embeds: []discord.Embed{Embed.Build()},
		Buttons: []discord.InteractiveComponent{discord.NewButton(discord.ButtonStyleLink, ViewLabel, "", Page, snowflake.ID(0))},
	}, nil

}

func Lyrics(Event *events.ApplicationCommandInteractionCreate) {

	// Defer response since this may take a minute

	Event.DeferCreateMessage(false)

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Response, Err := BuildLyricsResponse(GuildID, Locale)

	if Err != nil {

		var ErrorTitle, ErrorDesc string

		if Err.Error() == "no song playing" {

			ErrorTitle = Localizations.Get("Commands.Lyrics.Error.NoSong.Title", Locale)
			ErrorDesc = Localizations.Get("Commands.Lyrics.Error.NoSong.Description", Locale)

		} else {

			ErrorTitle = Localizations.Get("Commands.Lyrics.Error.NotFound.Title", Locale)
			ErrorDesc = Localizations.Get("Commands.Lyrics.Error.NotFound.Description", Locale)

		}

		Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdateBuilder().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       ErrorTitle,
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: ErrorDesc,
				Color:       0xFFB3BA,

			})).
			SetFlags(discord.MessageFlagsNone).
			Build())

		return

	}

	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdateBuilder().
		AddEmbeds(Response.Embeds...).
		AddActionRow(Response.Buttons...).
		SetFlags(discord.MessageFlagsNone).
		Build())

}