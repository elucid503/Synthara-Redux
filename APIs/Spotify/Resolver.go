package Spotify

import (
	"Synthara-Redux/APIs/Tidal"
	"fmt"
	"slices"
	"sync"
)

func SpotifyIDToSong(SpotifyID string) (Tidal.Song, *Track, error) {

	SpotifyTrack, ErrorFetching := Client.GetTrack(SpotifyID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	// Search on Tidal for the same song

	SearchQuery := fmt.Sprintf("%s %s", SpotifyTrack.Name, SpotifyTrack.Artists[0].Name)

	TidalResults, SearchErr := Tidal.SearchSongs(SearchQuery)

	if SearchErr != nil || len(TidalResults) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("no Tidal results found for Spotify track: %s", SpotifyID)

	}

	// Return the first Tidal result as the best match

	return TidalResults[0], SpotifyTrack, nil

}

func SpotifyAlbumToFirstSong(SpotifyAlbumID string) (Tidal.Song, *Album, error) {

	SpotifyAlbum, ErrorFetching := Client.GetAlbum(SpotifyAlbumID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	if len(SpotifyAlbum.Tracks.Items) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("Spotify album has no tracks: %s", SpotifyAlbumID)

	}

	FirstSong, _, FirstSongError := SpotifyIDToSong(SpotifyAlbum.Tracks.Items[0].ID)

	if FirstSongError != nil {

		return Tidal.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Tidal.PlaylistMeta{

		Platform: "Spotify",

		Index: 0,
		Total: len(SpotifyAlbum.Tracks.Items),

		Name: SpotifyAlbum.Name,
		ID:   SpotifyAlbum.ID,

	}

	return FirstSong, SpotifyAlbum, nil

}

func SpotifyPlaylistToFirstSong(SpotifyPlaylistID string) (Tidal.Song, *Playlist, error) {

	// Gets only the first song from the playlist

	SpotifyPlaylist, ErrorFetching := Client.GetPlaylist(SpotifyPlaylistID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	if len(SpotifyPlaylist.Tracks.Items) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("Spotify playlist is empty: %s", SpotifyPlaylistID)

	}

	FirstSong, _, FirstSongError := SpotifyIDToSong(SpotifyPlaylist.Tracks.Items[0].Track.ID)

	if FirstSongError != nil {

		return Tidal.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Tidal.PlaylistMeta{

		Platform: "Spotify",
		
		Index: 0,
		Total: len(SpotifyPlaylist.Tracks.Items),

		Name: SpotifyPlaylist.Name,
		ID:   SpotifyPlaylist.ID,

	}

	return FirstSong, SpotifyPlaylist, nil

}

func SpotifyAlbumToAllSongs(SpotifyAlbum *Album, IgnoreFirst bool) ([]Tidal.Song, *Album, error) {

	AllAlbumItems, ErrorFetchingTracks := SpotifyAlbum.GetAllItems()

	if (len(AllAlbumItems) < 1 || (IgnoreFirst && len(AllAlbumItems) < 2)) {

		return []Tidal.Song{}, SpotifyAlbum, fmt.Errorf("Spotify album has no tracks to process")

	}

	if IgnoreFirst {

		AllAlbumItems = AllAlbumItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Tidal.Song{}, SpotifyAlbum, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Spotify tracks to Tidal songs

	TidalSongs := make([]Tidal.Song, 0, len(AllAlbumItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllAlbumItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllAlbumItems[Index]
			
			ConvertedSong, _, ErrorConverting := SpotifyIDToSong(CurrentItem.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Tidal.PlaylistMeta{

					Platform: "Spotify",
					
					Index:    Index + 1,
					Total:    len(AllAlbumItems),

					Name: SpotifyAlbum.Name,

					ID:   SpotifyAlbum.ID,

				}

				WriteMutex.Lock()
				TidalSongs = append(TidalSongs, ConvertedSong)
				WriteMutex.Unlock()

			}

		}(i)

	}

	WaitGroup.Wait() // we will wait for all goroutines to finish

	// We must now sort the TidalSongs by their Playlist.Index to maintain order

	slices.SortFunc(TidalSongs, func(a, b Tidal.Song) int {

		return a.Internal.Playlist.Index - b.Internal.Playlist.Index

	})

	return TidalSongs, SpotifyAlbum, nil

}

func SpotifyPlaylistToAllSongs(SpotifyPlaylist *Playlist, IgnoreFirst bool) ([]Tidal.Song, *Playlist, error) {

	AllPlaylistItems, ErrorFetchingTracks := SpotifyPlaylist.GetAllItems()

	if (len(AllPlaylistItems) < 1 || (IgnoreFirst && len(AllPlaylistItems) < 2)) {

		return []Tidal.Song{}, SpotifyPlaylist, fmt.Errorf("Spotify playlist has no tracks to process")

	}

	if IgnoreFirst {

		AllPlaylistItems = AllPlaylistItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Tidal.Song{}, SpotifyPlaylist, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Spotify tracks to Tidal songs

	TidalSongs := make([]Tidal.Song, 0, len(AllPlaylistItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllPlaylistItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllPlaylistItems[Index]

			ConvertedSong, _, ErrorConverting := SpotifyIDToSong(CurrentItem.Track.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Tidal.PlaylistMeta{

					Platform: "Spotify",

					Index:    Index + 1,
					Total:    len(AllPlaylistItems),

					Name: SpotifyPlaylist.Name,
					ID:   SpotifyPlaylist.ID,

				}

				WriteMutex.Lock()
				TidalSongs = append(TidalSongs, ConvertedSong)
				WriteMutex.Unlock()

			}

		}(i)
	}

	WaitGroup.Wait() // we will wait for all goroutines to finish

	// We must now sort the TidalSongs by their Playlist.Index to maintain order

	slices.SortFunc(TidalSongs, func(a, b Tidal.Song) int {

		return a.Internal.Playlist.Index - b.Internal.Playlist.Index

	})

	return TidalSongs, SpotifyPlaylist, nil

}