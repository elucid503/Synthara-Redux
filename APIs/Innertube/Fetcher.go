package Innertube

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Utils"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/elucid503/Overture-Play/POToken"
	OverturePlay "github.com/elucid503/Overture-Play/Public"
	OverturePlayStructs "github.com/elucid503/Overture-Play/Structs"
	innertubego "github.com/nezbut/innertube-go"
)

// Types

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
var POTokenGenerator *POToken.BgUtilGenerator;

// Functions

func InitClient() error {

	InitializedClient, ErrorInitializing := innertubego.NewInnerTube(nil, "WEB_REMIX", "1.20240715.01.00", "", "", "", nil, true);

	if ErrorInitializing != nil {

		Utils.Logger.Error("Error initializing InnerTube client: " + ErrorInitializing.Error())
		return ErrorInitializing;

	}

	InnerTubeClient = InitializedClient;

	Generator := POToken.NewGenerator(nil) // Uses default settings (localhost:4416)

	// Checks if bgutil server is available

	PingResp, PingErr := Generator.Ping()

	if PingErr != nil {

		Utils.Logger.Warn("PO-token server not available; expect streaming problems")
		POTokenGenerator = nil;

		
	} else {

		Utils.Logger.Info(fmt.Sprintf("PO-token server available; version %s", PingResp.Version))
		POTokenGenerator = Generator;

	}

	Utils.Logger.Info("InnerTube client initialized successfully.")
	return nil;

}

func GetSong(YouTubeID string) (Song, error) {

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer RequestCancel()

	VideoDetails, VideoDetailsError := InnerTubeClient.Next(RequestContext, &YouTubeID, nil, nil, nil, nil)

	if VideoDetailsError != nil {

		return Song{}, VideoDetailsError

	}

	TabsVal, TabsExists := Utils.GetNestedValue(VideoDetails, "contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs", )
	
	if !TabsExists {

		return Song{}, errors.New("could not find tabs in VideoDetails response")

	}

	Tabs, TabsOK := TabsVal.([]interface{})

	if !TabsOK || len(Tabs) == 0 {

		return Song{}, errors.New("tabs is not a valid slice or is empty")

	}

	TabRendererVal, TabRendererExists := Utils.GetNestedValue(Tabs[0], "tabRenderer", "content", "musicQueueRenderer", "content", "playlistPanelRenderer", "contents")
	
	if !TabRendererExists {

		return Song{}, errors.New("could not find playlistPanelRenderer.contents in tabRenderer")
	}

	Contents, ContentsOK := TabRendererVal.([]interface{})

	if !ContentsOK || len(Contents) == 0 {

		return Song{}, errors.New("playlistPanelRenderer.contents is not a valid slice or is empty")

	}

	SongPanelVal, SongPanelExists := Utils.GetNestedValue(Contents[0], "playlistPanelVideoRenderer")

	if !SongPanelExists {

		return Song{}, errors.New("could not find playlistPanelVideoRenderer in contents")

	}

	SongPanelMap, SongPanelMapOK := SongPanelVal.(map[string]interface{})

	if !SongPanelMapOK {

		return Song{}, errors.New("playlistPanelVideoRenderer is not a valid map")

	}

	ParsedSong, ParseError := ParseSongPanel(SongPanelMap)

	if ParseError != nil {

		return Song{}, ParseError

	}

	return ParsedSong, nil

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

	// Finds the musicShelfRenderer (skips itemSectionRenderer if present, e.g., "showing results for...")
	
	var MusicShelfRendererFound bool

	for _, Section := range SectionContentsVal {

		ShelfContentsVal, ShelfContentsExists = Utils.GetNestedValue(Section, "musicShelfRenderer", "contents")

		if ShelfContentsExists {

			MusicShelfRendererFound = true
			break

		}

	}

	if !MusicShelfRendererFound {

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

		Song, CreateError := ParseSongItem(Renderer)

		if CreateError == nil {

			Results = append(Results, Song)
			
		}

	}

	return Results 

}

func GetSongInfo(YouTubeID string) (*OverturePlayStructs.YoutubeVideo, error) {
	
	VideosCache := Globals.GetOrCreateCache("Videos")

	FoundVideo, FoundVideoExists := VideosCache.Get(YouTubeID)

	if FoundVideoExists {

		return FoundVideo.(*OverturePlayStructs.YoutubeVideo), nil

	}

	Utils.Logger.Info(fmt.Sprintf("Fetching HLS manifest URL for song ID: %s", YouTubeID))

	Video, ErrorFetchingVideo := OverturePlay.Info(YouTubeID, &OverturePlay.InfoOptions{

		GetHLSFormats: true,

	}, nil, nil)

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

	VideosCache.Set(YouTubeID, Video, 1 * time.Hour) // 1 Hour TTL

	return Video, nil

}

func GetSongAudioSegments(YouTubeID string) ([]OverturePlayStructs.HLSSegment, int, error) {

	Video, ErrorGettingVideo := GetSongInfo(YouTubeID)

	if ErrorGettingVideo != nil {

		return nil, 0, ErrorGettingVideo

	}

	if len(Video.HLSFormats) == 0 {

		return nil, 0, errors.New("no HLS formats available for this video")

	}

	HLSManifestURL := Video.HLSFormats[0].URL

	Options := &OverturePlay.HLSOptions{ 
		
		IsAuthenticated: false,
		Generator: POTokenGenerator,
		VisitorData: Video.VisitorData,

	}

	Manifest, ErrorFetchingManifest := OverturePlay.GetHLSManifest(HLSManifestURL, Options) // 0 being lowest res

	if ErrorFetchingManifest != nil {

		return nil, 0, ErrorFetchingManifest

	}

	if len(Manifest.Playlists) == 0 {

		return nil, 0, errors.New("no playlists found in HLS manifest")

	}

	Playlist, ErrorFetchingPlaylist := OverturePlay.GetHLSPlaylist(Manifest.Playlists[0].URI, Options)

	if ErrorFetchingPlaylist != nil {

		return nil, 0, ErrorFetchingPlaylist

	}

	return Playlist.Segments, Playlist.TargetDuration, nil
	
}

func GetAudioSegmentBytes(Segment OverturePlayStructs.HLSSegment) ([]byte, error) {

	return OverturePlay.GetHLSSegment(Segment.URI, &OverturePlay.HLSOptions{ 
		
		IsAuthenticated: false,
	
	})
	
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

func GetPlaylistSongs(PlaylistID string) ([]Song, error) {

	Results := []Song{}

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second)

	defer RequestCancel()

	PlaylistResponse, PlaylistResponseError := InnerTubeClient.Browse(RequestContext, &PlaylistID, nil, nil)

	if PlaylistResponseError != nil {

		Utils.Logger.Error("Error fetching playlist: " + PlaylistResponseError.Error())
		return Results, PlaylistResponseError

	}

	// Extract playlist name

	PlaylistName := ""

	if NameVal, NameExists := Utils.GetNestedValue(PlaylistResponse, "microformat", "microformatDataRenderer", "title"); NameExists {

		if Name, NameValid := NameVal.(string); NameValid {

			PlaylistName = Name

		}

	}

	// Extract songs from musicPlaylistShelfRenderer

	SongItemsVal, SongItemsExists := Utils.GetNestedValue(PlaylistResponse, "contents", "twoColumnBrowseResultsRenderer", "secondaryContents", "sectionListRenderer", "contents")

	if !SongItemsExists {

		return Results, errors.New("could not find playlist contents")

	}

	SectionContents, SectionContentsValid := SongItemsVal.([]interface{})

	if !SectionContentsValid || len(SectionContents) == 0 {

		return Results, errors.New("section contents is not a valid slice or is empty")

	}

	ShelfContentsVal, ShelfContentsExists := Utils.GetNestedValue(SectionContents[0], "musicPlaylistShelfRenderer", "contents")

	if !ShelfContentsExists {

		return Results, errors.New("could not find musicPlaylistShelfRenderer contents")

	}

	SongItems, SongItemsValid := ShelfContentsVal.([]interface{})

	if !SongItemsValid {

		return Results, errors.New("musicPlaylistShelfRenderer contents is not a valid slice")

	}

	// Parse each song item

	for Index, Item := range SongItems {

		ItemMap, ItemMapValid := Item.(map[string]interface{})

		if !ItemMapValid {

			continue

		}

		Renderer, RendererExists := ItemMap["musicResponsiveListItemRenderer"].(map[string]interface{})

		if !RendererExists {

			continue

		}

		ParsedSong, ParseError := ParsePlaylistSongItem(Renderer)

		if ParseError != nil {

			continue

		}

		// Set playlist metadata

		ParsedSong.Internal.Playlist = PlaylistMeta{

			Platform: "YouTube",

			Index: Index + 1,
			Total: len(SongItems),

			Name: PlaylistName,
			ID:   PlaylistID,

		}

		Results = append(Results, ParsedSong)

	}

	return Results, nil

}

func GetArtistSongs(ArtistID string) ([]Song, error) {

	Results := []Song{}

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second)

	defer RequestCancel()

	ArtistResponse, ArtistResponseError := InnerTubeClient.Browse(RequestContext, &ArtistID, nil, nil)

	if ArtistResponseError != nil {

		Utils.Logger.Error("Error fetching artist: " + ArtistResponseError.Error())
		return Results, ArtistResponseError

	}

	// Extract playlist ID from musicShelfRenderer bottomEndpoint

	PlaylistID := ""

	if PlaylistIDVal, PlaylistIDExists := Utils.GetNestedValue(ArtistResponse, "contents", "singleColumnBrowseResultsRenderer", "tabs"); PlaylistIDExists {

		if Tabs, TabsValid := PlaylistIDVal.([]interface{}); TabsValid && len(Tabs) > 0 {

			if SectionContentsVal, SectionContentsExists := Utils.GetNestedValue(Tabs[0], "tabRenderer", "content", "sectionListRenderer", "contents"); SectionContentsExists {

				if SectionContents, SectionContentsValid := SectionContentsVal.([]interface{}); SectionContentsValid && len(SectionContents) > 0 {

					if BrowseIDVal, BrowseIDExists := Utils.GetNestedValue(SectionContents[0], "musicShelfRenderer", "bottomEndpoint", "browseEndpoint", "browseId"); BrowseIDExists {

						if BrowseID, BrowseIDValid := BrowseIDVal.(string); BrowseIDValid {

							PlaylistID = BrowseID

						}

					}

				}

			}

		}

	}

	if PlaylistID == "" {

		return Results, errors.New("could not extract playlist ID from artist response")

	}

	// Fetch playlist songs

	return GetPlaylistSongs(PlaylistID)

}

func GetAlbumSongs(AlbumID string) ([]Song, error) {

	Results := []Song{}

	RequestContext, RequestCancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer RequestCancel()

	AlbumResponse, AlbumResponseError := InnerTubeClient.Browse(RequestContext, &AlbumID, nil, nil)

	if AlbumResponseError != nil {

		Utils.Logger.Error("Error fetching album: " + AlbumResponseError.Error())
		return Results, AlbumResponseError

	}

	// Marshalled, _ := json.MarshalIndent(AlbumResponse, "", "  ")

	// os.WriteFile("album_response.json", Marshalled, 0644);

	// Extract album name and cover from header

	AlbumName := ""
	AlbumCover := "https://cdn.discordapp.com/embed/avatars/1.png" // Default fallback
	AlbumArtists := []string{}

	HeaderSubtitleVal, HeaderSubtitleExists := Utils.GetNestedValue(AlbumResponse, "contents", "twoColumnBrowseResultsRenderer", "tabs")

	if HeaderSubtitleExists {

		if Tabs, TabsOK := HeaderSubtitleVal.([]interface{}); TabsOK && len(Tabs) > 0 {

			HeaderContentsVal, HeaderContentsExists := Utils.GetNestedValue(Tabs[0], "tabRenderer", "content", "sectionListRenderer", "contents")

			if HeaderContentsExists {

				if HeaderContents, HeaderContentsOK := HeaderContentsVal.([]interface{}); HeaderContentsOK && len(HeaderContents) > 0 {

				// Extract album name

				TitleRuns, TitleRunsValid := Utils.GetNestedValue(HeaderContents[0], "musicResponsiveHeaderRenderer", "title", "runs")

				if TitleRunsValid {

					if Runs, RunsOK := TitleRuns.([]interface{}); RunsOK && len(Runs) > 0 {

						if FirstRun, FirstRunOK := Runs[0].(map[string]interface{}); FirstRunOK {

							if TitleText, TitleTextOK := FirstRun["text"].(string); TitleTextOK {

								AlbumName = TitleText

							}

						}

					}

				}

				// Extract album cover

				ThumbnailsVal, ThumbnailsExists := Utils.GetNestedValue(HeaderContents[0], "musicResponsiveHeaderRenderer", "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")

				if ThumbnailsExists {

					if Thumbnails, ThumbnailsOK := ThumbnailsVal.([]interface{}); ThumbnailsOK && len(Thumbnails) > 0 {

						if LastThumbnail, LastThumbnailOK := Thumbnails[len(Thumbnails)-1].(map[string]interface{}); LastThumbnailOK {

							if URL, URLOK := LastThumbnail["url"].(string); URLOK {

								AlbumCover = URL

							}

						}

					}

				}

				// Extract album artists

				ArtistRuns, ArtistRunsValid := Utils.GetNestedValue(HeaderContents[0], "musicResponsiveHeaderRenderer", "straplineTextOne", "runs")

				if ArtistRunsValid {

					if Runs, RunsOK := ArtistRuns.([]interface{}); RunsOK {

						for _, Run := range Runs {

							if RunMap, RunMapOK := Run.(map[string]interface{}); RunMapOK {

								if ArtistText, ArtistTextOK := RunMap["text"].(string); ArtistTextOK {

									TrimmedArtist := strings.TrimSpace(ArtistText)

									if TrimmedArtist != "" && TrimmedArtist != " & " && TrimmedArtist != ", " {

										AlbumArtists = append(AlbumArtists, TrimmedArtist)

									}

									}

								}

							}

						}

					}

				}

			}

		}

	}

	// Extract songs from musicShelfRenderer

	SongItemsVal, SongItemsExists := Utils.GetNestedValue(AlbumResponse, "contents", "twoColumnBrowseResultsRenderer", "secondaryContents", "sectionListRenderer", "contents")

	if !SongItemsExists {

		return Results, errors.New("could not find album contents")

	}

	SectionContents, SectionContentsValid := SongItemsVal.([]interface{})

	if !SectionContentsValid || len(SectionContents) == 0 {

		return Results, errors.New("section contents is not a valid slice or is empty")

	}

	ShelfContentsVal, ShelfContentsExists := Utils.GetNestedValue(SectionContents[0], "musicShelfRenderer", "contents")

	if !ShelfContentsExists {

		return Results, errors.New("could not find musicShelfRenderer contents")

	}

	SongItems, SongItemsValid := ShelfContentsVal.([]interface{})

	if !SongItemsValid {

		return Results, errors.New("musicShelfRenderer contents is not a valid slice")

	}

	// Parse each song item

	for Index, Item := range SongItems {

		ItemMap, ItemMapValid := Item.(map[string]interface{})

		if !ItemMapValid {

			continue

		}

		Renderer, RendererExists := ItemMap["musicResponsiveListItemRenderer"].(map[string]interface{})

		if !RendererExists {

			continue

		}

		ParsedSong, ParseError := ParseAlbumSongItem(Renderer, AlbumName, AlbumArtists, AlbumCover, AlbumID)

		if ParseError != nil {

			continue

		}

		// Set album metadata

		ParsedSong.Internal.Playlist = PlaylistMeta{

			Platform: "YouTube",

			Index: Index + 1,
			Total: len(SongItems),

			Name: AlbumName,
			ID:   AlbumID,

		}

		Results = append(Results, ParsedSong)

	}

	return Results, nil

}