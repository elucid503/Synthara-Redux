package YouTube

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Utils"
	"errors"
	"fmt"
	"strings"

	"github.com/kkdai/youtube/v2"
)

var Client youtube.Client

func Init() {

	Client = youtube.Client{}

}

// VideoIDToSong fetches a YouTube video by ID and converts it to a Tidal.Song using search
func VideoIDToSong(VideoID string) (Tidal.Song, *youtube.Video, error) {

	Video, Err := Client.GetVideo(VideoID)

	if Err != nil {

		return Tidal.Song{}, nil, fmt.Errorf("failed to fetch YouTube video: %w", Err)

	}

	// Build search query from video title and author

	SearchQuery := fmt.Sprintf("%s %s", Video.Title, Video.Author)

	Utils.Logger.Info(fmt.Sprintf("Searching Tidal for YouTube video: %s", SearchQuery))

	// Search Tidal for matching song

	Results, SearchErr := Tidal.SearchSongs(SearchQuery)

	if SearchErr != nil || len(Results) == 0 {

		return Tidal.Song{}, Video, errors.New("no matching song found on Tidal")

	}

	// Return the first (best) match

	return Results[0], Video, nil

}

// PlaylistIDToFirstSong fetches first video from a YouTube playlist and converts to Tidal.Song
func PlaylistIDToFirstSong(PlaylistID string) (Tidal.Song, *youtube.Playlist, error) {

	Playlist, Err := Client.GetPlaylist(PlaylistID)

	if Err != nil {

		return Tidal.Song{}, nil, fmt.Errorf("failed to fetch YouTube playlist: %w", Err)

	}

	if len(Playlist.Videos) == 0 {

		return Tidal.Song{}, Playlist, errors.New("playlist is empty")

	}

	FirstVideo := Playlist.Videos[0]

	SearchQuery := fmt.Sprintf("%s %s", FirstVideo.Title, FirstVideo.Author)

	Utils.Logger.Info(fmt.Sprintf("Searching Tidal for first playlist video: %s", SearchQuery))

	Results, SearchErr := Tidal.SearchSongs(SearchQuery)

	if SearchErr != nil || len(Results) == 0 {

		return Tidal.Song{}, Playlist, errors.New("no matching song found on Tidal for first video")

	}

	return Results[0], Playlist, nil
}

// PlaylistIDToAllSongs fetches all videos from a YouTube playlist and converts to Tidal.Song array
func PlaylistIDToAllSongs(Playlist *youtube.Playlist, IgnoreFirst bool) ([]Tidal.Song, int, *youtube.Playlist, error) {

	Songs := make([]Tidal.Song, 0)
	FailedCount := 0

	StartIndex := 0

	if IgnoreFirst {

		StartIndex = 1

	}

	for i := StartIndex; i < len(Playlist.Videos); i++ {

		Video := Playlist.Videos[i]

		// Build search query
		SearchQuery := fmt.Sprintf("%s %s", Video.Title, Video.Author)

		Utils.Logger.Info(fmt.Sprintf("Searching Tidal for playlist video %d/%d: %s", i+1, len(Playlist.Videos), SearchQuery))

		// Search Tidal

		Results, SearchErr := Tidal.SearchSongs(SearchQuery)

		if SearchErr != nil || len(Results) == 0 {

			Utils.Logger.Warn(fmt.Sprintf("No Tidal match found for video: %s", Video.Title))
			FailedCount++
			continue

		}

		Songs = append(Songs, Results[0])

	}

	return Songs, FailedCount, Playlist, nil

}

// MusicAlbumIDToFirstSong fetches first track from YouTube Music album and converts to Tidal.Song
func MusicAlbumIDToFirstSong(AlbumID string) (Tidal.Song, *youtube.Playlist, error) {

	// YouTube Music albums are playlists with "OLAK5uy_" prefix or "VL" prefix
	Playlist, Err := Client.GetPlaylist(AlbumID)

	if Err != nil {

		return Tidal.Song{}, nil, fmt.Errorf("failed to fetch YouTube Music album: %w", Err)

	}

	if len(Playlist.Videos) == 0 {

		return Tidal.Song{}, Playlist, errors.New("album is empty")

	}

	FirstVideo := Playlist.Videos[0]

	AlbumInfo := Playlist.Title
	ArtistName := Playlist.Author

	SearchQuery := fmt.Sprintf("%s %s %s", FirstVideo.Title, ArtistName, AlbumInfo)

	Utils.Logger.Info(fmt.Sprintf("Searching Tidal for first album track: %s", SearchQuery))
	
	Results, SearchErr := Tidal.SearchSongs(SearchQuery)

	if SearchErr != nil || len(Results) == 0 {

		return Tidal.Song{}, Playlist, errors.New("no matching song found on Tidal for first track")

	}

	return Results[0], Playlist, nil

}

// MusicAlbumIDToAllSongs fetches all tracks from YouTube Music album and converts to Tidal.Song array
func MusicAlbumIDToAllSongs(Playlist *youtube.Playlist, IgnoreFirst bool) ([]Tidal.Song, int, *youtube.Playlist, error) {

	Songs := make([]Tidal.Song, 0)
	FailedCount := 0

	StartIndex := 0

	if IgnoreFirst {

		StartIndex = 1

	}

	AlbumInfo := Playlist.Title
	ArtistName := Playlist.Author

	for i := StartIndex; i < len(Playlist.Videos); i++ {

		Video := Playlist.Videos[i]

		SearchQuery := fmt.Sprintf("%s %s %s", Video.Title, ArtistName, AlbumInfo)

		Utils.Logger.Info(fmt.Sprintf("Searching Tidal for album track %d/%d: %s", i+1, len(Playlist.Videos), SearchQuery))

		Results, SearchErr := Tidal.SearchSongs(SearchQuery)

		if SearchErr != nil || len(Results) == 0 {

			Utils.Logger.Warn(fmt.Sprintf("No Tidal match found for track: %s", Video.Title))
			FailedCount++
			continue

		}

		Songs = append(Songs, Results[0])

	}

	return Songs, FailedCount, Playlist, nil
	
}

// MusicArtistIDToSongs fetches songs from YouTube Music artist channel
func MusicArtistIDToSongs(ArtistID string) ([]Tidal.Song, error) {

	PlaylistID := ArtistID
	
	// If it's a channel ID (starts with UC), convert to uploads playlist

	if strings.HasPrefix(ArtistID, "UC") {

		PlaylistID = "UU" + ArtistID[2:]

	}

	Playlist, PlaylistErr := Client.GetPlaylist(PlaylistID)

	if PlaylistErr != nil {

		return nil, fmt.Errorf("failed to fetch YouTube Music artist playlist: %w", PlaylistErr)

	}

	Songs := make([]Tidal.Song, 0)

	Limit := 50

	if len(Playlist.Videos) < Limit {

		Limit = len(Playlist.Videos)

	}

	ArtistName := Playlist.Author

	for i := 0; i < Limit; i++ {

		Video := Playlist.Videos[i]

		// Build search query

		SearchQuery := fmt.Sprintf("%s %s", Video.Title, ArtistName)

		Utils.Logger.Info(fmt.Sprintf("Searching Tidal for artist track %d/%d: %s", i+1, Limit, SearchQuery))

		// Search Tidal

		Results, SearchErr := Tidal.SearchSongs(SearchQuery)

		if SearchErr != nil || len(Results) == 0 {

			Utils.Logger.Warn(fmt.Sprintf("No Tidal match found for track: %s", Video.Title))
			continue
			
		}

		Songs = append(Songs, Results[0])

	}

	if len(Songs) == 0 {

		return nil, errors.New("no matching songs found on Tidal for artist")

	}

	return Songs, nil

}