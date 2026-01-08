package Spotify

import (
	"Synthara-Redux/APIs/Innertube"
	"fmt"
)

func SpotifyIDToSong(SpotifyID string) (Innertube.Song, error) {

	SpotifyTrack, ErrorFetching := Client.GetTrack(SpotifyID)

	if ErrorFetching != nil {

		return Innertube.Song{}, ErrorFetching

	}

	// We need a YouTube ID, so we must backfill via a search

	SearchQuery := fmt.Sprintf("%s %s", SpotifyTrack.Name, SpotifyTrack.Artists[0].Name)

	YouTubeResults := Innertube.SearchForSongs(SearchQuery)

	if len(YouTubeResults) == 0 {

		return Innertube.Song{}, fmt.Errorf("no YouTube results found for Spotify track: %s", SpotifyID)

	}

	// Return the first YouTube result as the best match

	return YouTubeResults[0], nil

}