package Utils

import (
	"Synthara-Redux/Structs"
	"context"
	"errors"
	"os"
	"slices"
	"time"

	OverturePlay "github.com/elucid503/Overture-Play/Public"
	OverturePlayStructs "github.com/elucid503/Overture-Play/Structs"
	innertubego "github.com/nezbut/innertube-go"
)

var InnerTubeClient *innertubego.InnerTube;

func InitInnerTubeClient() error {

	InitializedClient, ErrorInitializing := innertubego.NewInnerTube(nil, "WEB_REMIX", "1.20240715.01.00", "", "", "", nil, true);

	if ErrorInitializing != nil {

		Logger.Error("Error initializing InnerTube client: " + ErrorInitializing.Error())
		return ErrorInitializing;

	}

	InnerTubeClient = InitializedClient;

	Logger.Info("InnerTube client initialized successfully")
	return nil;

}

func SearchInnerTubeSongs(Query string) []Structs.Song {

	Results := []Structs.Song{}
	Params := "EgWKAQIIAWoQEAMQCRAFEBAQBBAVEAoQEQ%3D%3D"; // Songs

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second) // 5s timeout
	defer RequestCancel()

	SearchRequestResults, SearchRequestError := InnerTubeClient.Search(RequestContext, &Query, &Params, nil)

	if SearchRequestError != nil {

		Logger.Error("Error performing search request: " + SearchRequestError.Error())
		return Results

	}

	ShelfContentsVal, ShelfContentsExists := GetNestedValue(SearchRequestResults, "contents", "tabbedSearchResultsRenderer", "tabs", ) 

	if !ShelfContentsExists {

		return Results

	}

	Tabs, TabsExists := ShelfContentsVal.([]interface{})

	if !TabsExists || len(Tabs) == 0 {

		return Results

	}

	ContentsVal, ContentsExists := GetNestedValue(Tabs[0], "tabRenderer", "content", "sectionListRenderer", "contents")
	
	if !ContentsExists {

		return Results

	}

	SectionContentsVal, ShelfContentsExists := ContentsVal.([]interface{})

	if !ShelfContentsExists || len(SectionContentsVal) == 0 {

		return Results

	}

	ShelfContentsVal, ShelfContentsExists = GetNestedValue(SectionContentsVal[0], "musicShelfRenderer", "contents")

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