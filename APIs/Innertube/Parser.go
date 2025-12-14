package Innertube

import (
	"Synthara-Redux/Utils"
	"errors"
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

	Cover := Utils.ExtractSongThumbnail(Renderer)
	DurationSeconds := Utils.ParseFormattedDuration(DurationFormatted)

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