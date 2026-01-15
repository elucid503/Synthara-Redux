package Apple

import (
	"Synthara-Redux/APIs/Innertube"
	"fmt"
	"slices"
	"sync"
)

func AppleMusicIDToSong(AppleMusicID string) (Innertube.Song, *Song, error) {

	AppleMusicSong, ErrorFetching := Client.GetSong(AppleMusicID)

	if ErrorFetching != nil {

		return Innertube.Song{}, nil, ErrorFetching

	}

	// We need a YouTube ID, so we must backfill via a search

	SearchQuery := fmt.Sprintf("%s %s", AppleMusicSong.Attributes.Name, AppleMusicSong.Attributes.ArtistName)

	YouTubeResults := Innertube.SearchForSongs(SearchQuery)

	if len(YouTubeResults) == 0 {

		return Innertube.Song{}, nil, fmt.Errorf("no YouTube results found for Apple Music track: %s", AppleMusicID)

	}

	// Return the first YouTube result as the best match

	return YouTubeResults[0], AppleMusicSong, nil

}

func AppleMusicAlbumToFirstSong(AppleMusicAlbumID string) (Innertube.Song, *Album, error) {

	AppleMusicAlbum, ErrorFetching := Client.GetAlbum(AppleMusicAlbumID)

	if ErrorFetching != nil {

		return Innertube.Song{}, nil, ErrorFetching

	}

	if len(AppleMusicAlbum.Relationships.Tracks.Data) == 0 {

		return Innertube.Song{}, nil, fmt.Errorf("Apple Music album has no tracks: %s", AppleMusicAlbumID)

	}

	FirstSong, _, FirstSongError := AppleMusicIDToSong(AppleMusicAlbum.Relationships.Tracks.Data[0].ID)

	if FirstSongError != nil {

		return Innertube.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Innertube.PlaylistMeta{

		Platform: "Apple Music",

		Index: 0,
		Total: len(AppleMusicAlbum.Relationships.Tracks.Data),

		Name: AppleMusicAlbum.Attributes.Name,
		ID:   AppleMusicAlbum.ID,

	}

	return FirstSong, AppleMusicAlbum, nil

}

func AppleMusicPlaylistToFirstSong(AppleMusicPlaylistID string) (Innertube.Song, *Playlist, error) {

	// Gets only the first song from the playlist

	AppleMusicPlaylist, ErrorFetching := Client.GetPlaylist(AppleMusicPlaylistID)

	if ErrorFetching != nil {

		return Innertube.Song{}, nil, ErrorFetching

	}

	if len(AppleMusicPlaylist.Relationships.Tracks.Data) == 0 {

		return Innertube.Song{}, nil, fmt.Errorf("Apple Music playlist is empty: %s", AppleMusicPlaylistID)

	}

	FirstSong, _, FirstSongError := AppleMusicIDToSong(AppleMusicPlaylist.Relationships.Tracks.Data[0].ID)

	if FirstSongError != nil {

		return Innertube.Song{}, nil, FirstSongError

	}

	FirstSong.Internal.Playlist = Innertube.PlaylistMeta{

		Platform: "Apple Music",
		
		Index: 0,
		Total: len(AppleMusicPlaylist.Relationships.Tracks.Data),

		Name: AppleMusicPlaylist.Attributes.Name,
		ID:   AppleMusicPlaylist.ID,

	}

	return FirstSong, AppleMusicPlaylist, nil

}

func AppleMusicAlbumToAllSongs(AppleMusicAlbum *Album, IgnoreFirst bool) ([]Innertube.Song, *Album, error) {

	AllAlbumItems, ErrorFetchingTracks := AppleMusicAlbum.GetAllItems()

	if (len(AllAlbumItems) < 1 || (IgnoreFirst && len(AllAlbumItems) < 2)) {

		return []Innertube.Song{}, AppleMusicAlbum, fmt.Errorf("Apple Music album has no tracks to process")

	}

	if IgnoreFirst {

		AllAlbumItems = AllAlbumItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Innertube.Song{}, AppleMusicAlbum, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Apple Music tracks to Innertube songs

	InnertubeSongs := make([]Innertube.Song, 0, len(AllAlbumItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllAlbumItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllAlbumItems[Index]
			
			ConvertedSong, _, ErrorConverting := AppleMusicIDToSong(CurrentItem.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Innertube.PlaylistMeta{

					Platform: "Apple Music",
					
					Index:    Index + 1,
					Total:    len(AllAlbumItems),

					Name: AppleMusicAlbum.Attributes.Name,

					ID:   AppleMusicAlbum.ID,

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

	return InnertubeSongs, AppleMusicAlbum, nil

}

func AppleMusicPlaylistToAllSongs(AppleMusicPlaylist *Playlist, IgnoreFirst bool) ([]Innertube.Song, *Playlist, error) {

	AllPlaylistItems, ErrorFetchingTracks := AppleMusicPlaylist.GetAllItems()

	if (len(AllPlaylistItems) < 1 || (IgnoreFirst && len(AllPlaylistItems) < 2)) {

		return []Innertube.Song{}, AppleMusicPlaylist, fmt.Errorf("Apple Music playlist has no tracks to process")

	}

	if IgnoreFirst {

		AllPlaylistItems = AllPlaylistItems[1:] // removes the first item; is useful if processing the first seperately to save time

	}

	if ErrorFetchingTracks != nil {

		return []Innertube.Song{}, AppleMusicPlaylist, ErrorFetchingTracks

	}

	// We now will, in parallel, convert all Apple Music tracks to Innertube songs

	InnertubeSongs := make([]Innertube.Song, 0, len(AllPlaylistItems))

	var WriteMutex sync.Mutex
	var WaitGroup sync.WaitGroup

	for i := range AllPlaylistItems {

		WaitGroup.Add(1)

		go func(Index int) {

			defer WaitGroup.Done()

			CurrentItem := AllPlaylistItems[Index]

			ConvertedSong, _, ErrorConverting := AppleMusicIDToSong(CurrentItem.ID)

			if ErrorConverting == nil {

				ConvertedSong.Internal.Playlist = Innertube.PlaylistMeta{

					Platform: "Apple Music",

					Index:    Index + 1,
					Total:    len(AllPlaylistItems),

					Name: AppleMusicPlaylist.Attributes.Name,
					ID:   AppleMusicPlaylist.ID,

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

	return InnertubeSongs, AppleMusicPlaylist, nil

}
