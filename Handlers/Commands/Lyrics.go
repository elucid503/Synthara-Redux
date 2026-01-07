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

func LyricsFetcher(Title string, Artist string, Album string) (*LyricsAPIResponse, error) {

	Params := url.Values{}
	Params.Set("title", Title)

	if Artist != "" {

		Params.Set("artist", Artist)

	}

	Params.Set("album", Album)
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

func Lyrics(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || Guild.Queue.Current == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Lyrics.Errors.NoSong", Locale),


		})

		return

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

	APIRespPtr, Err := LyricsFetcher(Cleaned, Artist, Song.Album)

	if Err != nil && Cleaned != Song.Title {

		// Retries with original title

		APIRespPtr, Err = LyricsFetcher(Song.Title, Artist, Song.Album)

	}

	if Err != nil || APIRespPtr == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Lyrics.Errors.NotFound", Locale),

		})

		return

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

	Embeds := []discord.Embed{}
	SongColor, _ := Utils.GetDominantColorHex(Song.Cover)

	// Determine which view link to use (word vs line)

	ViewKey := "Embeds.Lyrics.ViewLine"

	if strings.ToLower(APIResp.Type) == "word" {

		ViewKey = "Embeds.Lyrics.ViewWord"

	}

	ViewText := Localizations.GetFormat(ViewKey, Locale, Page)

	// Builds a single embed with truncation if necessary

	MaxDesc := 4000
	Desc := ""

	Writers := ""

	if len(APIResp.Metadata.SongWriters) > 0 {

		Writers = Localizations.GetFormat("Embeds.Lyrics.WrittenBy", Locale, strings.Join(APIResp.Metadata.SongWriters, ", ")) + "\n\n"

	} 

	if  len(Writers) + len(LyricsText) + 2 + len(ViewText) <= MaxDesc { // +2 for newlines

		Desc = Writers + LyricsText + "\n\n" + ViewText

	} else {

		Truncated := Localizations.Get("Embeds.Lyrics.Truncated", Locale)
		Reserve := len(Writers) + (len("\n\n") * 2) + len(Truncated) + len("\n\n") + 3 // 3 for ellipsis

		Limit := MaxDesc - Reserve

		if Limit < 0 {

			Limit = 0

		}

		Prefix := Writers + LyricsText

		if len(Prefix) > Limit {

			Prefix = Prefix[:Limit]

		}

		Desc = strings.TrimSpace(Prefix) + "..." + "\n\n" + Truncated + "\n\n"

	}

	Embed := discord.NewEmbedBuilder()

	Embed.SetURL(Page)
	Embed.SetTitle(Song.Title)
	Embed.SetThumbnail(Song.Cover)
	Embed.SetDescription(Desc)
	Embed.SetColor(SongColor)
	Embed.SetAuthor(Localizations.Get("Embeds.Lyrics.Title", Locale), "", "")

	Embeds = append(Embeds, Embed.Build())

	Event.CreateMessage(discord.MessageCreate{

		Embeds: Embeds,

	})

}