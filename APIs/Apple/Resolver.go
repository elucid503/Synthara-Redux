package Apple

import (
	"Synthara-Redux/APIs/Tidal"
	"fmt"
	"slices"
	"sync"
)

func AppleMusicIDToSong(AppleMusicID string) (Tidal.Song, *Song, error) {

	AppleMusicSong, ErrorFetching := Client.GetSong(AppleMusicID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	// Search on Tidal for the same song

	SearchQuery := fmt.Sprintf("%s %s", AppleMusicSong.Attributes.Name, AppleMusicSong.Attributes.ArtistName)

	TidalResults, SearchErr := Tidal.SearchSongs(SearchQuery)

	if SearchErr != nil || len(TidalResults) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("no Tidal results found for Apple Music track: %s", AppleMusicID)

	}

	// Return the first Tidal result as the best match

	return TidalResults[0], AppleMusicSong, nil

}

func AppleMusicAlbumToFirstSong(AppleMusicAlbumID string) (Tidal.Song, *Album, error) {

	AppleMusicAlbum, ErrorFetching := Client.GetAlbum(AppleMusicAlbumID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	if len(AppleMusicAlbum.Relationships.Tracks.Data) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("Apple Music album has no tracks: %s", AppleMusicAlbumID)

	}

	FirstSong, _, FirstSongError := AppleMusicIDToSong(AppleMusicAlbum.Relationships.Tracks.Data[0].ID)

	if FirstSongError != nil {

		return Tidal.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Tidal.PlaylistMeta{

		Platform: "Apple Music",

		Index: 0,
		Total: len(AppleMusicAlbum.Relationships.Tracks.Data),

		Name: AppleMusicAlbum.Attributes.Name,
		ID:   AppleMusicAlbum.ID,

	}

	return FirstSong, AppleMusicAlbum, nil

}

func AppleMusicPlaylistToFirstSong(AppleMusicPlaylistID string) (Tidal.Song, *Playlist, error) {

	// Gets only the first song from the playlist

	AppleMusicPlaylist, ErrorFetching := Client.GetPlaylist(AppleMusicPlaylistID)

	if ErrorFetching != nil {

		return Tidal.Song{}, nil, ErrorFetching

	}

	if len(AppleMusicPlaylist.Relationships.Tracks.Data) == 0 {

		return Tidal.Song{}, nil, fmt.Errorf("Apple Music playlist is empty: %s", AppleMusicPlaylistID)

	}

	FirstSong, _, FirstSongError := AppleMusicIDToSong(AppleMusicPlaylist.Relationships.Tracks.Data[0].ID)

	if FirstSongError != nil {

		return Tidal.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Tidal.PlaylistMeta{

		Platform: "Apple Music",
		
		Index: 0,
		Total: len(AppleMusicPlaylist.Relationships.Tracks.Data),

		Name: AppleMusicPlaylist.Attributes.Name,
		ID:   AppleMusicPlaylist.ID,

	}

	return FirstSong, AppleMusicPlaylist, nil

}

func AppleMusicAlbumToAllSongs(AppleMusicAlbum *Album, IgnoreFirst bool) ([]Tidal.Song, *Album, error) {

	AllAlbumItems, ErrorFetchingTracks := AppleMusicAlbum.GetAllItems()

	if (len(AllAlbumItems) < 1 || (IgnoreFirst && len(AllAlbumItems) < 2)) {

		return []Tidal.Song{}, AppleMusicAlbum, fmt.Errorf("Apple Music album has no tracks to process")

	}

	if IgnoreFirst {

		AllAlbumItems = AllAlbumItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Tidal.Song{}, AppleMusicAlbum, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Apple Music tracks to Tidal songs

	TidalSongs := make([]Tidal.Song, 0, len(AllAlbumItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllAlbumItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllAlbumItems[Index]
			
			ConvertedSong, _, ErrorConverting := AppleMusicIDToSong(CurrentItem.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Tidal.PlaylistMeta{

					Platform: "Apple Music",
					
					Index:    Index + 1,
					Total:    len(AllAlbumItems),

					Name: AppleMusicAlbum.Attributes.Name,

					ID:   AppleMusicAlbum.ID,

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

	return TidalSongs, AppleMusicAlbum, nil

}

func AppleMusicPlaylistToAllSongs(AppleMusicPlaylist *Playlist, IgnoreFirst bool) ([]Tidal.Song, *Playlist, error) {

	AllPlaylistItems, ErrorFetchingTracks := AppleMusicPlaylist.GetAllItems()

	if (len(AllPlaylistItems) < 1 || (IgnoreFirst && len(AllPlaylistItems) < 2)) {

		return []Tidal.Song{}, AppleMusicPlaylist, fmt.Errorf("Apple Music playlist has no tracks to process")

	}

	if IgnoreFirst {

		AllPlaylistItems = AllPlaylistItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Tidal.Song{}, AppleMusicPlaylist, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Apple Music tracks to Tidal songs

	TidalSongs := make([]Tidal.Song, 0, len(AllPlaylistItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllPlaylistItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllPlaylistItems[Index]

			ConvertedSong, _, ErrorConverting := AppleMusicIDToSong(CurrentItem.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Tidal.PlaylistMeta{

					Platform: "Apple Music",

					Index:    Index + 1,
					Total:    len(AllPlaylistItems),

					Name: AppleMusicPlaylist.Attributes.Name,
					ID:   AppleMusicPlaylist.ID,

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

	return TidalSongs, AppleMusicPlaylist, nil

}
