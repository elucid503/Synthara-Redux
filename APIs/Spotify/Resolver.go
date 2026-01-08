package Spotify

import (
	"Synthara-Redux/APIs/Innertube"
	"fmt"
	"slices"
	"sync"
)

func SpotifyIDToSong(SpotifyID string) (Innertube.Song, *Track, error) {

	SpotifyTrack, ErrorFetching := Client.GetTrack(SpotifyID)

	if ErrorFetching != nil {

		return Innertube.Song{}, nil, ErrorFetching

	}

	// We need a YouTube ID, so we must backfill via a search

	SearchQuery := fmt.Sprintf("%s %s", SpotifyTrack.Name, SpotifyTrack.Artists[0].Name)

	YouTubeResults := Innertube.SearchForSongs(SearchQuery)

	if len(YouTubeResults) == 0 {

		return Innertube.Song{}, nil, fmt.Errorf("no YouTube results found for Spotify track: %s", SpotifyID)

	}

	// Return the first YouTube result as the best match

	return YouTubeResults[0], SpotifyTrack, nil

}

func SpotifyPlaylistToFirstSong(SpotifyPlaylistID string) (Innertube.Song, *Playlist, error) {

	// Gets only the first song from the playlist

	SpotifyPlaylist, ErrorFetching := Client.GetPlaylist(SpotifyPlaylistID)

	if ErrorFetching != nil {

		return Innertube.Song{}, nil, ErrorFetching

	}

	if len(SpotifyPlaylist.Tracks.Items) == 0 {

		return Innertube.Song{}, nil, fmt.Errorf("Spotify playlist is empty: %s", SpotifyPlaylistID)

	}

	FirstSong, _, FirstSongError := SpotifyIDToSong(SpotifyPlaylist.Tracks.Items[0].Track.ID)

	if FirstSongError != nil {

		return Innertube.Song{}, nil, FirstSongError

	}

	return FirstSong, SpotifyPlaylist, nil

}

func SpotifyPlaylistToAllSongs(SpotifyPlaylist *Playlist, IgnoreFirst bool) ([]Innertube.Song, *Playlist, error) {

	AllPlaylistItems, ErrorFetchingTracks := SpotifyPlaylist.GetAllItems()

	if (len(AllPlaylistItems) < 1 || (IgnoreFirst && len(AllPlaylistItems) < 2)) {

		return []Innertube.Song{}, SpotifyPlaylist, fmt.Errorf("Spotify playlist has no tracks to process")

	}

	if IgnoreFirst {

		AllPlaylistItems = AllPlaylistItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Innertube.Song{}, SpotifyPlaylist, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Spotify tracks to Innertube songs

	InnertubeSongs := make([]Innertube.Song, 0, len(AllPlaylistItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllPlaylistItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllPlaylistItems[Index]

			ConvertedSong, _, ErrorConverting := SpotifyIDToSong(CurrentItem.Track.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Innertube.PlaylistMeta{

					Platform: "Spotify",

					Index:    Index + 1,
					Total:    len(AllPlaylistItems),

					Name: SpotifyPlaylist.Name,
					ID:   SpotifyPlaylist.ID,

				}

				WriteMutex.Lock()
				InnertubeSongs = append(InnertubeSongs, ConvertedSong)
				WriteMutex.Unlock()

			}

		}(i)
	}

	WaitGroup.Wait() // we will wait for all goroutines to finish

	// We must now sort the InnertubeSongs by their Playlist.Index to maintain order

	slices.SortFunc(InnertubeSongs, func(a, b Innertube.Song) int {

		return a.Internal.Playlist.Index - b.Internal.Playlist.Index

	})

	return InnertubeSongs, SpotifyPlaylist, nil

}