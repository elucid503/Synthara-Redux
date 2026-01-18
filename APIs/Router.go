package APIs

import (
	"errors"
	"regexp"
	"strings"
)

const (

	// Tidal (native platform)

	URITypeTidalSong = "Song"
	URITypeTidalAlbum = "Album"
	URITypeTidalPlaylist = "Playlist"

	// YouTube (regular platform)
	
	URITypeYouTubeVideo = "YTVideo"
	URITypeYouTubePlaylist = "YTPlaylist"

	// YouTube Music (native platform)

	URITypeYTMusicAlbum = "YTMAlbum"
	URITypeYTMusicArtist = "YTMArtist"

	// Spotify 

	URITypeSPSong = "SPSong"
	URITypeSPAlbum = "SPAlbum"
	URITypeSPPlaylist = "SPPlaylist"

	// Apple Music 

	URITypeAMSong = "AMSong"
	URITypeAMAlbum = "AMAlbum"
	URITypeAMPlaylist = "AMPlaylist"

	// Search Query

	URITypeNone = "None"

	// User features

	URITypeFavorites = "Favorites"
	URITypeSuggestions = "Suggestions"

)

const (

	ExternalPlatformSpotify = "Spotify"
	ExternalPlatformAppleMusic = "AppleMusic"

)

// URI Schema: Synthara-Redux:<Type>:<ID>

// IsURI checks if the given input string is a Synthara-Redux URI.
func IsURI(Input string) bool {

	return len(Input) > 15 && strings.HasPrefix(Input, "Synthara-Redux:")

}

// ParseURI takes a Synthara-Redux URI string and returns its Type and ID.
func ParseURI(Input string) (string, string, error) {

	if !IsURI(Input) {

		return "", "", errors.New("invalid Synthara-Redux URI")

	}

	Parts := make([]string, 0)

	CurrentPart := ""

	for i := 15; i < len(Input); i++ {

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

// IsURL checks if the given input string is a URL
func IsURL(Input string) bool {

	URLRegex := regexp.MustCompile(`(?i)^(https?://[^\s]+)$`)

	return URLRegex.MatchString(Input)

}

// Route converts any input (URLs or plain text) to Synthara-Redux URIs
func Route(Input string) (string, error) {

	Input = strings.TrimSpace(Input)

	// Checks if input is a URI; if so, we skip further processing

	if IsURI(Input) {

		return Input, nil

	}

	// Check if the input is a URL

	if !IsURL(Input) {

		// Not a URL, treat as search query and return the corresponding URI

		return "Synthara-Redux:" + URITypeNone +":" + Input, nil

	}

	// Now, since we have a URL, we must determine which platform it belongs to

	URL := Input

	// YouTube Video (regular YouTube)

	YTVideoRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})`)
	
	if Match := YTVideoRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeYouTubeVideo + ":" + Match[1], nil

	}

	// YouTube Music - Album (OLAK5uy_ prefix)

	YTMAlbumRegex := regexp.MustCompile(`(?i)(?:https?://)?music\.youtube\.com/playlist\?list=(OLAK5uy_[a-zA-Z0-9_-]+)`)
	
	if Match := YTMAlbumRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeYTMusicAlbum + ":" + Match[1], nil

	}

	// YouTube Music - Artist/Channel

	YTMArtistRegex := regexp.MustCompile(`(?i)(?:https?://)?music\.youtube\.com/channel/([a-zA-Z0-9_-]+)`)

	if Match := YTMArtistRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeYTMusicArtist + ":" + Match[1], nil

	}

	// YouTube - Playlist

	YTPlaylistRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?youtube\.com/playlist\?list=([a-zA-Z0-9_-]+)`)
	
	if Match := YTPlaylistRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeYouTubePlaylist + ":" + Match[1], nil

	}

	// Spotify - Track

	SpotifyTrackRegex := regexp.MustCompile(`(?i)(?:https?://)?open\.spotify\.com/track/([a-zA-Z0-9]+)`)

	if Match := SpotifyTrackRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeSPSong + ":" + Match[1], nil

	}

	// Spotify - Album

	SpotifyAlbumRegex := regexp.MustCompile(`(?i)(?:https?://)?open\.spotify\.com/album/([a-zA-Z0-9]+)`)

	if Match := SpotifyAlbumRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeSPAlbum + ":" + Match[1], nil

	}

	// Spotify - Playlist

	SpotifyPlaylistRegex := regexp.MustCompile(`(?i)(?:https?://)?open\.spotify\.com/playlist/([a-zA-Z0-9]+)`)

	if Match := SpotifyPlaylistRegex.FindStringSubmatch(URL); Match != nil {
		return "Synthara-Redux:" + URITypeSPPlaylist + ":" + Match[1], nil
	}

	// Apple Music - Song

	AppleSongRegex := regexp.MustCompile(`(?i)(?:https?://)?music\.apple\.com/[a-z]{2}/song/[^/]+/(\d+)`)
	
	if Match := AppleSongRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeAMSong + ":" + Match[1], nil

	}

	// Apple Music - Album

	AppleAlbumRegex := regexp.MustCompile(`(?i)(?:https?://)?music\.apple\.com/[a-z]{2}/album/[^/]+/(\d+)`)
	
	if Match := AppleAlbumRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeAMAlbum + ":" + Match[1], nil

	}

	// Apple Music - Playlist

	ApplePlaylistRegex := regexp.MustCompile(`(?i)(?:https?://)?music\.apple\.com/[a-z]{2}/playlist/[^/]+/(pl\.[a-zA-Z0-9]+)`)
	
	if Match := ApplePlaylistRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeAMPlaylist + ":" + Match[1], nil

	}

	// Tidal - Track

	TidalTrackRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:listen\.)?tidal\.com/(?:browse/)?track/(\d+)`)
	
	if Match := TidalTrackRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeTidalSong + ":" + Match[1], nil

	}

	// Tidal - Album

	TidalAlbumRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:listen\.)?tidal\.com/(?:browse/)?album/(\d+)`)
	
	if Match := TidalAlbumRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeTidalAlbum + ":" + Match[1], nil

	}

	// Tidal - Playlist

	TidalPlaylistRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:listen\.)?tidal\.com/(?:browse/)?playlist/([a-zA-Z0-9-]+)`)
	
	if Match := TidalPlaylistRegex.FindStringSubmatch(URL); Match != nil {

		return "Synthara-Redux:" + URITypeTidalPlaylist + ":" + Match[1], nil

	}

	return "", errors.New("unsupported URL format")

}