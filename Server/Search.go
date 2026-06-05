package Server

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Structs"
	"net/http"
	"strconv"

	"github.com/disgoorg/snowflake/v2"
	"github.com/gin-gonic/gin"
)

type SuggestionItem struct {

	Type     string `json:"type"`               // "Track" or "Text"

	Text     string `json:"text,omitempty"`     // Text suggestions

	TidalID  int64  `json:"tidal_id,omitempty"` // Track suggestions

	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`

}

// resolveQuery validates ID + q params and returns the guild and query string.
func resolveQuery(Context *gin.Context) (*Structs.Guild, string, bool) {

	GuildIDStr := Context.Query("ID")
	Query := Context.Query("q")

	if GuildIDStr == "" || Query == "" {

		Context.JSON(http.StatusBadRequest, gin.H{"Error": "ID and q are required"})
		return nil, "", false

	}

	GuildID, Err := snowflake.Parse(GuildIDStr)

	if Err != nil {

		Context.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid ID"})
		return nil, "", false

	}

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		Context.JSON(http.StatusNotFound, gin.H{"Error": "Guild not found"})
		return nil, "", false

	}

	return Guild, Query, true

}

func HandleSuggestions(Context *gin.Context) {

	Guild, Query, Ok := resolveQuery(Context)

	if !Ok {

		return

	}

	if WebControlsLocked(Guild.Features.Locked, Context.Request) {

		Context.JSON(webControlsLockStatus(Guild.Features.Locked, Context.Request), gin.H{"Error": WebControlsLockMessage(Guild.Features.Locked, Context.Request)})
		return

	}

	Items := []SuggestionItem{}

	for _, S := range Tidal.GetSearchSuggestions(Query) {

		if S.Metadata.Type == "Song" {

			ID, Err := strconv.ParseInt(S.Metadata.ID, 10, 64)

			if Err != nil {

				continue

			}

			Items = append(Items, SuggestionItem{Type: "Track", TidalID: ID, Title: S.Metadata.Title, Subtitle: S.Metadata.Subtitle})

		} else if S.Text != "" {

			Items = append(Items, SuggestionItem{Type: "Text", Text: S.Text})

		}

	}

	Context.JSON(http.StatusOK, Items)

}
