package Innertube

import (
	"Synthara-Redux/Utils"
	"errors"
	"fmt"
	"strings"
)

// Parser Functons

func ParseSong(Renderer map[string]interface{}) (Song, error) {

    VideoIDVal, VideoIDExists := Utils.GetNestedValue(Renderer, "playlistItemData", "videoId")

    if !VideoIDExists {

        return Song{}, errors.New("video ID not found in renderer")

    }

    VideoID, VideoIDValid := VideoIDVal.(string)

    if !VideoIDValid || VideoID == "" {

        return Song{}, errors.New("invalid video ID in renderer")

    }


    FlexColumns, FlexColumnsExists := Renderer["flexColumns"].([]interface{})

    if !FlexColumnsExists || len(FlexColumns) < 2 {

        return Song{}, errors.New("insufficient flex columns in renderer")

    }

    Title := ""
    TitleRuns, TitleRunsValid := Utils.GetNestedValue(FlexColumns[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")

    if TitleRunsValid {

        if Runs, runsOK := TitleRuns.([]interface{}); runsOK && len(Runs) > 0 {

            if FirstRun, FirstRunOK := Runs[0].(map[string]interface{}); FirstRunOK {

                if TitleText, titleTextOK := FirstRun["text"].(string); titleTextOK {

                    Title = TitleText

                }

            }

        }

    }

    Artists := []string{}

    Album := ""
    DurationFormatted := ""

    RunsVal, RunsValueOK := Utils.GetNestedValue(FlexColumns[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")

    if RunsValueOK {

        if Runs, RunsValid := RunsVal.([]interface{}); RunsValid {

            for Index, Run := range Runs {

                if RunMap, RunMapOK := Run.(map[string]interface{}); RunMapOK {
					
                    if RunText, RunTextOK := RunMap["text"].(string); RunTextOK {

                        if RunText == " • " { continue }

                        switch Index {

							case 0:

								SplitRunText := strings.SplitN(RunText, ", ", -1);

								for _, Artist := range SplitRunText {

									TrimmedArtist := strings.TrimSpace(strings.ReplaceAll(Artist, "\u0026", ""))

									if TrimmedArtist != "" {

										Artists = append(Artists, TrimmedArtist)

									}

								}

							case 2:

								Album = RunText // Run 2 -> Album

							case 4:

								DurationFormatted = RunText // Run 4 -> Formatted Duration

						}

					}

				}

			}

		}

	}

	Cover := ExtractSongThumbnail(Renderer)
	DurationSeconds := ParseFormattedDuration(DurationFormatted)

	return Song{

		YouTubeID: VideoID,

		Title:     Title,
		Artists:   Artists,
		Album:     Album,

		Duration: Duration{

			Seconds:   DurationSeconds,
			Formatted: DurationFormatted,

		},
		Cover: Cover,

	}, nil

}

func ParseSuggestion(Renderer map[string]interface{}) (SearchSuggestion, error) {

	Suggestion := SearchSuggestion{

		Text:     "",
		Metadata: SearchSuggestionMetadata{},

	}

	// Check if this is a simple search suggestion (searchSuggestionRenderer)

	if _, IsSimpleSuggestion := Renderer["searchSuggestionRenderer"]; IsSimpleSuggestion {

		return parseSimpleSuggestion(Renderer)

	}

	// Check if this is a music item suggestion (musicResponsiveListItemRenderer)

	if _, IsMusicItem := Renderer["musicResponsiveListItemRenderer"]; IsMusicItem {

		return parseMusicSuggestion(Renderer)

	}

	return Suggestion, errors.New("unknown suggestion renderer type")

}

func parseSimpleSuggestion(Renderer map[string]interface{}) (SearchSuggestion, error) {

	Suggestion := SearchSuggestion{

		Text:     "",
		Metadata: SearchSuggestionMetadata{

			Type: "Search",

		},

	}

	// Extract suggestion text from searchSuggestionRenderer

	SuggestionVal, SuggestionExists := Renderer["searchSuggestionRenderer"].(map[string]interface{})

	if !SuggestionExists {

		return Suggestion, errors.New("searchSuggestionRenderer not found")

	}

	// Get the text runs

	RunsVal, RunsExists := Utils.GetNestedValue(SuggestionVal, "suggestion", "runs")

	if !RunsExists {

		return Suggestion, errors.New("suggestion runs not found")

	}

	Runs, RunsValid := RunsVal.([]interface{})

	if !RunsValid || len(Runs) == 0 {

		return Suggestion, errors.New("invalid suggestion runs")

	}

	// Concatenate all text from runs (ignoring bold formatting)

	TextBuilder := strings.Builder{}

	for _, Run := range Runs {

		if RunMap, RunValid := Run.(map[string]interface{}); RunValid {

			if Text, TextExists := RunMap["text"].(string); TextExists {

				TextBuilder.WriteString(Text)

			}
		}
	}

	Suggestion.Text = TextBuilder.String()
	Suggestion.Metadata.Title = Suggestion.Text

	// Extract ID (query) from navigationEndpoint

	if QueryVal, QueryExists := Utils.GetNestedValue(SuggestionVal, "navigationEndpoint", "searchEndpoint", "query"); QueryExists {

		if Query, QueryOK := QueryVal.(string); QueryOK {

			Suggestion.Metadata.ID = Query

		}

	}

	return Suggestion, nil

}

func parseMusicSuggestion(Renderer map[string]interface{}) (SearchSuggestion, error) {

	Suggestion := SearchSuggestion{

		Text:     "",
		Metadata: SearchSuggestionMetadata{},

	}

	MusicItemVal, MusicItemExists := Renderer["musicResponsiveListItemRenderer"].(map[string]interface{})

	if !MusicItemExists {

		return Suggestion, errors.New("musicResponsiveListItemRenderer not found")

	}

	// Extract flex columns

	FlexColumns, FlexColumnsExists := MusicItemVal["flexColumns"].([]interface{})

	if !FlexColumnsExists || len(FlexColumns) < 2 {

		return Suggestion, errors.New("insufficient flex columns in music suggestion")

	}

	// Extract title from first flex column

	TitleRunsVal, TitleRunsExists := Utils.GetNestedValue(FlexColumns[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")

	if TitleRunsExists {

		if TitleRuns, TitleRunsValid := TitleRunsVal.([]interface{}); TitleRunsValid && len(TitleRuns) > 0 {

			if FirstRun, FirstRunValid := TitleRuns[0].(map[string]interface{}); FirstRunValid {

				if TitleText, TitleTextExists := FirstRun["text"].(string); TitleTextExists {

					Suggestion.Text = TitleText
					Suggestion.Metadata.Title = TitleText

				}

			}

		}

	}

	// Extract metadata from second flex column

	MetadataRunsVal, MetadataRunsExists := Utils.GetNestedValue(FlexColumns[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")

	if MetadataRunsExists {

		if MetadataRuns, MetadataRunsValid := MetadataRunsVal.([]interface{}); MetadataRunsValid && len(MetadataRuns) > 0 {

			// First run is always the type (Song, Artist, Video, etc.)

			if FirstRun, FirstRunOK := MetadataRuns[0].(map[string]interface{}); FirstRunOK {

				if TypeText, TypeTextOK := FirstRun["text"].(string); TypeTextOK {

					Suggestion.Metadata.Type = TypeText

				}

			}

			// Remaining runs (after the separator) form the subtitle

			if len(MetadataRuns) > 2 {

				SubtitleBuilder := strings.Builder{}

				for i := 2; i < len(MetadataRuns); i++ {

					if RunMap, RunMapOK := MetadataRuns[i].(map[string]interface{}); RunMapOK {

						if RunText, RunTextOK := RunMap["text"].(string); RunTextOK {

							SubtitleBuilder.WriteString(strings.ReplaceAll(RunText, "\u0026", "&"))

						}

					}

				}

				Suggestion.Metadata.Subtitle = strings.TrimSpace(SubtitleBuilder.String())

			}

		}

	}

	// Refine type from navigationEndpoint if it's ambiguous or missing

	if PageTypeVal, PageTypeExists := Utils.GetNestedValue(MusicItemVal, "navigationEndpoint", "browseEndpoint", "browseEndpointContextSupportedConfigs", "browseEndpointContextMusicConfig", "pageType"); PageTypeExists {

		if PageType, PageTypeOK := PageTypeVal.(string); PageTypeOK {

			switch PageType {

			case "MUSIC_PAGE_TYPE_ARTIST":

				if Suggestion.Metadata.Type != "Artist" {

					if Suggestion.Metadata.Subtitle == "" && Suggestion.Metadata.Type != "" {

						Suggestion.Metadata.Subtitle = Suggestion.Metadata.Type

					}

					Suggestion.Metadata.Type = "Artist"

				}

			case "MUSIC_PAGE_TYPE_ALBUM":

				Suggestion.Metadata.Type = "Album"

			case "MUSIC_PAGE_TYPE_PLAYLIST":

				Suggestion.Metadata.Type = "Playlist"

			}

		}

	}

	if _, WatchExists := Utils.GetNestedValue(MusicItemVal, "navigationEndpoint", "watchEndpoint"); WatchExists {

		if Suggestion.Metadata.Type == "" {

			Suggestion.Metadata.Type = "Song"

		}

	}

	// Extract ID from navigationEndpoint

	if VideoIDVal, VideoIDExists := Utils.GetNestedValue(MusicItemVal, "navigationEndpoint", "watchEndpoint", "videoId"); VideoIDExists {

		if VideoID, VideoIDOK := VideoIDVal.(string); VideoIDOK {

			Suggestion.Metadata.ID = VideoID

		}

	} else if BrowseIDVal, BrowseIDExists := Utils.GetNestedValue(MusicItemVal, "navigationEndpoint", "browseEndpoint", "browseId"); BrowseIDExists {

		if BrowseID, BrowseIDOK := BrowseIDVal.(string); BrowseIDOK {

			Suggestion.Metadata.ID = BrowseID

		}

	} else if PlaylistIDVal, PlaylistIDExists := Utils.GetNestedValue(MusicItemVal, "navigationEndpoint", "watchPlaylistEndpoint", "playlistId"); PlaylistIDExists {

		if PlaylistID, PlaylistIDOK := PlaylistIDVal.(string); PlaylistIDOK {

			Suggestion.Metadata.ID = PlaylistID

		}

	}

	return Suggestion, nil

}

func ExtractSongThumbnail(Renderer map[string]interface{}) string {

	ThumbnailsVal, ThumbnailsValExists := Utils.GetNestedValue(Renderer, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")

	if !ThumbnailsValExists || len(ThumbnailsVal.([]interface{})) == 0 {

		return "https://cdn.discordapp.com/embed/avatars/1.png" // Default 'thumbnail'

	}

	LastThumbnail, LastThumbnailExists := ThumbnailsVal.([]interface{})[len(ThumbnailsVal.([]interface{}))-1].(map[string]interface{})

	if !LastThumbnailExists {

		return "https://cdn.discordapp.com/embed/avatars/1.png"

	}

	URL, LastThumbnailURLExists := LastThumbnail["url"].(string)

	if !LastThumbnailURLExists {

		return "https://cdn.discordapp.com/embed/avatars/1.png"

	}

	return URL

}

func ParseFormattedDuration(FormattedDuration string) int {

	if FormattedDuration == "" {

		return 0
		
	}

	var Minutes, Seconds int

	if _, ParseError := fmt.Sscanf(FormattedDuration, "%d:%d", &Minutes, &Seconds); ParseError == nil {

		return Minutes*60 + Seconds

	}

	return 0

}