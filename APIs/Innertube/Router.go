package Innertube

import (
	"errors"
)

const (

	// Youtube Music

	URITypeSong = "Song"
	URITypeVideo = "Video"
	URITypeAlbum = "Album"
	URITypeArtist = "Artist"
	URITypePlaylist = "Playlist"

	// Spotify 

	URITypeSPSong = "SPSong"
	URITypeSPAlbum = "SPAlbum"
	URITypeSPPlaylist = "SPPlaylist"

)

// URI Schema: Synthara-Redux:<Type>:<ID>

// IsURI checks if the given input string is a Synthara-Redux URI.
func IsURI(Input string) bool {

	return len(Input) > 14 && Input[:14] == "Synthara-Redux:"

}

// ParseURI takes a Synthara-Redux URI string and returns its Type and ID.
func ParseURI(Input string) (string, string, error) {

	if !IsURI(Input) {

		return "", "", errors.New("invalid Synthara-Redux URI")

	}

	Parts := make([]string, 0)

	CurrentPart := ""

	for i := 14; i < len(Input); i++ {

		if Input[i] == ':' {

			Parts = append(Parts, CurrentPart)
			CurrentPart = ""

		} else {

			CurrentPart += string(Input[i])

		}

	}

	Parts = append(Parts, CurrentPart)

	if len(Parts) != 2 {

		return "", "", errors.New("invalid Synthara-Redux URI format")

	}

	return Parts[0], Parts[1], nil

}

// RouteURI takes a Synthara-Redux URI string and returns a list of Songs found/cooresponding to that URI.
func RouteURI(URI string) ([]*Song, error) {

	Type, ID, ErrorParsing := ParseURI(URI)

	if ErrorParsing != nil {

		return nil, ErrorParsing

	}

	switch Type {

		case URITypeSong:

			// Song from YouTube ID. TODO: Possibly improve fetch function

			FetchedSong, ErrorFetchingSong := GetSongByYouTubeID(ID)

			if ErrorFetchingSong != nil {

				return nil, ErrorFetchingSong

			}

			return []*Song{FetchedSong}, nil

		case URITypeVideo:

			// Same process

			FetchedSong, ErrorFetchingSong := GetSongByYouTubeID(ID)

			if ErrorFetchingSong != nil {

				return nil, ErrorFetchingSong

			}

			return []*Song{FetchedSong}, nil

		case URITypeAlbum:

		case URITypeArtist:

		case URITypePlaylist:

		case URITypeSPSong:

		case URITypeSPAlbum:

		case URITypeSPPlaylist:

	}

	return []*Song{}, nil

}