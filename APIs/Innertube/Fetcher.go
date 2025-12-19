package Innertube

import (
	"Synthara-Redux/Utils"
	"context"
	"errors"
	"os"
	"slices"
	"time"

	OverturePlay "github.com/elucid503/Overture-Play/Public"
	OverturePlayStructs "github.com/elucid503/Overture-Play/Structs"
	innertubego "github.com/nezbut/innertube-go"
)

// Types

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

type SearchSuggestionMetadata struct {

	ID       string `json:"id"`
	Type     string `json:"type"`

	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

}

type SearchSuggestion struct {

	Text string `json:"text"`

	Metadata SearchSuggestionMetadata `json:"metadata"`

}

// Variables 

var InnerTubeClient *innertubego.InnerTube;

// Functions

func InitClient() error {

	InitializedClient, ErrorInitializing := innertubego.NewInnerTube(nil, "WEB_REMIX", "1.20240715.01.00", "", "", "", nil, true);

	if ErrorInitializing != nil {

		Utils.Logger.Error("Error initializing InnerTube client: " + ErrorInitializing.Error())
		return ErrorInitializing;

	}

	InnerTubeClient = InitializedClient;

	Utils.Logger.Info("InnerTube client initialized successfully.")
	return nil;

}

func SearchForSongs(Query string) []Song {

	Results := []Song{}
	Params := "EgWKAQIIAWoQEAMQCRAFEBAQBBAVEAoQEQ%3D%3D"; // Songs

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second) // 5s timeout
	defer RequestCancel()

	SearchRequestResults, SearchRequestError := InnerTubeClient.Search(RequestContext, &Query, &Params, nil)

	if SearchRequestError != nil {

		Utils.Logger.Error("Error performing search request: " + SearchRequestError.Error())
		return Results

	}

	ShelfContentsVal, ShelfContentsExists := Utils.GetNestedValue(SearchRequestResults, "contents", "tabbedSearchResultsRenderer", "tabs", ) 

	if !ShelfContentsExists {

		return Results

	}

	Tabs, TabsExists := ShelfContentsVal.([]interface{})

	if !TabsExists || len(Tabs) == 0 {

		return Results

	}

	ContentsVal, ContentsExists := Utils.GetNestedValue(Tabs[0], "tabRenderer", "content", "sectionListRenderer", "contents")
	
	if !ContentsExists {

		return Results

	}

	SectionContentsVal, ShelfContentsExists := ContentsVal.([]interface{})

	if !ShelfContentsExists || len(SectionContentsVal) == 0 {

		return Results

	}

	ShelfContentsVal, ShelfContentsExists = Utils.GetNestedValue(SectionContentsVal[0], "musicShelfRenderer", "contents")

	if !ShelfContentsExists {

		return Results

	}

	ShelfContents, ShelfContentsExists := ShelfContentsVal.([]interface{})

	if !ShelfContentsExists {

		return Results

	}

	// Parse each song result
	
	for _, Item := range ShelfContents {

		ItemMap, ItemMapOK := Item.(map[string]interface{})

		if !ItemMapOK {

			continue

		}

		Renderer, RendererExists := ItemMap["musicResponsiveListItemRenderer"].(map[string]interface{})

		if !RendererExists {

			continue

		}

		Song, CreateError := ParseSong(Renderer)

		if CreateError == nil {

			Results = append(Results, Song)
			
		}

	}

	return Results 

}

func GetSongAudioSegments(YouTubeID string) ([]OverturePlayStructs.HLSSegment, error) {

	Cookie := os.Getenv("YOUTUBE_COOKIE")

	Video, ErrorFetchingVideo := OverturePlay.Info(YouTubeID, &OverturePlay.InfoOptions{

		GetHLSFormats: true,

	}, nil, &Cookie)

	if ErrorFetchingVideo != nil {

		return nil, ErrorFetchingVideo

	}

	if (len(Video.HLSFormats) == 0) {	

		return nil, errors.New("no HLS formats available for this video")

	}

	// Ideal HLS format is lowest res, so we sort ascending by width

	slices.SortFunc(Video.HLSFormats, func(a, b OverturePlayStructs.Format) int {

		return *a.Width - *b.Width

	})

	Manifest, ErrorFetchingManifest := OverturePlay.GetHLSManifest(Video.HLSFormats[0].URL, nil) // 0 being lowest res

	if ErrorFetchingManifest != nil {

		return nil, ErrorFetchingManifest

	}

	if len(Manifest.Playlists) == 0 {

		return nil, errors.New("no playlists found in HLS manifest")

	}

	Playlist, ErrorFetchingPlaylist := OverturePlay.GetHLSPlaylist(Manifest.Playlists[0].URI, &OverturePlay.HLSOptions{})

	if ErrorFetchingPlaylist != nil {

		return nil, ErrorFetchingPlaylist

	}

	return Playlist.Segments, nil
	
}

func GetAudioSegmentBytes(Segment OverturePlayStructs.HLSSegment) ([]byte, error) {

	return OverturePlay.GetHLSSegment(Segment.URI, &OverturePlay.HLSOptions{ })
	
}

func GetSearchSuggestions(Query string) []SearchSuggestion {

	Results := []SearchSuggestion{}

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second) // 5s timeout
	defer RequestCancel()

	SearchSuggestions, SearchError := InnerTubeClient.MusicGetSearchSuggestions(RequestContext, &Query)
	
	if SearchError != nil {

		Utils.Logger.Error("Error fetching search suggestions: " + SearchError.Error())
		return Results

	}

	// Get root contents array

	ContentsVal, ContentsExists := Utils.GetNestedValue(SearchSuggestions, "contents")

	if !ContentsExists {

		return Results

	}

	Contents, ContentsValid := ContentsVal.([]interface{})

	if !ContentsValid || len(Contents) == 0 {

		return Results

	}

	// Iterate through each suggestion section

	for _, SectionItem := range Contents {

		SectionMap, SectionMapValid := SectionItem.(map[string]interface{})

		if !SectionMapValid {

			continue

		}

		// Get the suggestion section renderer
		SectionRendererVal, SectionRendererExists := Utils.GetNestedValue(SectionMap, "searchSuggestionsSectionRenderer", "contents")

		if !SectionRendererExists {

			continue

		}

		SuggestionContents, SuggestionContentsValid := SectionRendererVal.([]interface{})

		if !SuggestionContentsValid {

			continue

		}

		// Parse each suggestion renderer

		for _, SuggestionItem := range SuggestionContents {

			SuggestionMap, SuggestionMapValid := SuggestionItem.(map[string]interface{})

			if !SuggestionMapValid {

				continue

			}

			// Use the parser to convert renderer to SearchSuggestion
			
			Suggestion, ParseError := ParseSuggestion(SuggestionMap)

			if ParseError == nil {

				Results = append(Results, Suggestion)

			}

		}

	}

	return Results

}