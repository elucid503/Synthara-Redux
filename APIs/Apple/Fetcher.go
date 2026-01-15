package Apple

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Types

type Artwork struct {

	Width  int    `json:"width"`
	URL    string `json:"url"`
	Height int    `json:"height"`

	TextColor3 string `json:"textColor3"`
	TextColor2 string `json:"textColor2"`
	TextColor4 string `json:"textColor4"`
	TextColor1 string `json:"textColor1"`

	BGColor string `json:"bgColor"`
	HasP3   bool   `json:"hasP3"`

}

type PlayParams struct {

	ID   string `json:"id"`
	Kind string `json:"kind"`

}

type EditorialNotes struct {

	Tagline  string `json:"tagline"`
	Standard string `json:"standard"`
	Short    string `json:"short"`

}

type Preview struct {

	URL string `json:"url"`

}

type SongAttributes struct {

	HasTimeSyncedLyrics         bool         `json:"hasTimeSyncedLyrics"`
	AlbumName                   string       `json:"albumName"`
	GenreNames                  []string     `json:"genreNames"`
	TrackNumber                 int          `json:"trackNumber"`
	ReleaseDate                 string       `json:"releaseDate"`
	DurationInMillis            int          `json:"durationInMillis"`
	IsVocalAttenuationAllowed   bool         `json:"isVocalAttenuationAllowed"`
	IsMasteredForItunes         bool         `json:"isMasteredForItunes"`
	ISRC                        string       `json:"isrc"`
	Artwork                     Artwork      `json:"artwork"`
	AudioLocale                 string       `json:"audioLocale"`
	ComposerName                string       `json:"composerName"`
	URL                         string       `json:"url"`
	PlayParams                  PlayParams   `json:"playParams"`
	DiscNumber                  int          `json:"discNumber"`
	HasLyrics                   bool         `json:"hasLyrics"`
	IsAppleDigitalMaster        bool         `json:"isAppleDigitalMaster"`
	AudioTraits                 []string     `json:"audioTraits"`
	Name                        string       `json:"name"`
	Previews                    []Preview    `json:"previews"`
	ArtistName                  string       `json:"artistName"`
	ContentRating               string       `json:"contentRating"`

}

type SongRelationships struct {

	Albums  Albums  `json:"albums"`
	Artists Artists `json:"artists"`

}

type Song struct {

	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Href          string            `json:"href"`
	Attributes    SongAttributes    `json:"attributes"`
	Relationships SongRelationships `json:"relationships"`

}

type Songs struct {

	Href string `json:"href"`
	Next string `json:"next"`
	Data []Song `json:"data"`

}

type AlbumAttributes struct {

	Copyright           string         `json:"copyright"`
	GenreNames          []string       `json:"genreNames"`
	ReleaseDate         string         `json:"releaseDate"`
	IsMasteredForItunes bool           `json:"isMasteredForItunes"`
	UPC                 string         `json:"upc"`
	Artwork             Artwork        `json:"artwork"`
	PlayParams          PlayParams     `json:"playParams"`
	URL                 string         `json:"url"`
	RecordLabel         string         `json:"recordLabel"`
	TrackCount          int            `json:"trackCount"`
	IsCompilation       bool           `json:"isCompilation"`
	IsPrerelease        bool           `json:"isPrerelease"`
	AudioTraits         []string       `json:"audioTraits"`
	IsSingle            bool           `json:"isSingle"`
	Name                string         `json:"name"`
	ArtistName          string         `json:"artistName"`
	ContentRating       string         `json:"contentRating"`
	EditorialNotes      EditorialNotes `json:"editorialNotes"`
	IsComplete          bool           `json:"isComplete"`

}

type AlbumRelationships struct {

	Tracks  Songs   `json:"tracks"`
	Artists Artists `json:"artists"`

}

type Album struct {

	ID            string             `json:"id"`
	Type          string             `json:"type"`
	Href          string             `json:"href"`
	Attributes    AlbumAttributes    `json:"attributes"`
	Relationships AlbumRelationships `json:"relationships"`

}

type Albums struct {

	Href string  `json:"href"`
	Next string  `json:"next"`
	Data []Album `json:"data"`

}

type PlaylistAttributes struct {

	LastModifiedDate string `json:"lastModifiedDate"`
	SupportsSing     bool   `json:"supportsSing"`
	Description      struct {
		Standard string `json:"standard"`
		Short    string `json:"short"`
	} `json:"description"`
	Artwork             Artwork        `json:"artwork"`
	URL                 string         `json:"url"`
	PlayParams          PlayParams     `json:"playParams"`
	HasCollaboration    bool           `json:"hasCollaboration"`
	CuratorName         string         `json:"curatorName"`
	AudioTraits         []string       `json:"audioTraits"`
	IsChart             bool           `json:"isChart"`
	Name                string         `json:"name"`
	EditorialPlaylistKind string       `json:"editorialPlaylistKind"`
	PlaylistType        string         `json:"playlistType"`
	EditorialNotes      struct {
		Name     string `json:"name"`
		Standard string `json:"standard"`
		Short    string `json:"short"`
	} `json:"editorialNotes"`

}

type PlaylistRelationships struct {

	Tracks  Songs   `json:"tracks"`
	Curator Artists `json:"curator"`

}

type Playlist struct {

	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Href          string                `json:"href"`
	Attributes    PlaylistAttributes    `json:"attributes"`
	Relationships PlaylistRelationships `json:"relationships"`

}

type ArtistAttributes struct {

	Name       string   `json:"name"`
	GenreNames []string `json:"genreNames"`
	Artwork    Artwork  `json:"artwork"`
	URL        string   `json:"url"`

}

type ArtistRelationships struct {

	Albums Albums `json:"albums"`
	Songs  Songs  `json:"songs"`

}

type Artist struct {

	ID            string              `json:"id"`
	Type          string              `json:"type"`
	Href          string              `json:"href"`
	Attributes    ArtistAttributes    `json:"attributes"`
	Relationships ArtistRelationships `json:"relationships"`

}

type Artists struct {

	Href string   `json:"href"`
	Data []Artist `json:"data"`

}

type AppleMusicResponse struct {

	Data []json.RawMessage `json:"data"`

}

// Variables

var Client *AppleClient
var JWTToken string

type AppleClient struct {

	JWT         string
	CountryCode string

}

// Functions

func Initialize(JWT string) *AppleClient {

	JWTToken = JWT

	Client = &AppleClient{

		JWT:         JWT,
		CountryCode: "us",

	}

	return Client

}

func (Ac *AppleClient) MakeRequest(Endpoint string, Params map[string]string) ([]byte, error) {

	ConstructedURL, ParseErr := url.Parse(fmt.Sprintf("https://amp-api.music.apple.com%s", Endpoint))

	if ParseErr != nil {

		return nil, ParseErr

	}

	Query := ConstructedURL.Query()

	for Key, Value := range Params {

		Query.Set(Key, Value)

	}

	ConstructedURL.RawQuery = Query.Encode()

	Req, ReqErr := http.NewRequest("GET", ConstructedURL.String(), nil)

	if ReqErr != nil {

		return nil, ReqErr

	}

	Req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", Ac.JWT))
	Req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:144.0) Gecko/20100101 Firefox/144.0")
	Req.Header.Set("Origin", "https://music.apple.com")

	HTTPClient := &http.Client{}

	Resp, RespErr := HTTPClient.Do(Req)

	if RespErr != nil {

		return nil, RespErr

	}

	defer Resp.Body.Close()

	Body, BodyErr := io.ReadAll(Resp.Body)

	if BodyErr != nil {

		return nil, BodyErr

	}

	if Resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("unexpected status: %d", Resp.StatusCode)

	}

	return Body, nil

}

func (Ac *AppleClient) GetSong(ID string) (*Song, error) {

	Params := map[string]string{
		"l":       "en-us",
		"include": "artists,albums",
	}

	Body, Err := Ac.MakeRequest(fmt.Sprintf("/v1/catalog/%s/songs/%s", Ac.CountryCode, ID), Params)

	if Err != nil {

		return nil, Err

	}

	var Response AppleMusicResponse

	if Err := json.Unmarshal(Body, &Response); Err != nil {

		return nil, Err

	}

	if len(Response.Data) == 0 {

		return nil, fmt.Errorf("no song found with ID: %s", ID)

	}

	var SongData Song

	if Err := json.Unmarshal(Response.Data[0], &SongData); Err != nil {

		return nil, Err

	}

	return &SongData, nil

}

func (Ac *AppleClient) GetAlbum(ID string) (*Album, error) {

	Params := map[string]string{
		"l":       "en-us",
		"include": "artists",
	}

	Body, Err := Ac.MakeRequest(fmt.Sprintf("/v1/catalog/%s/albums/%s", Ac.CountryCode, ID), Params)

	if Err != nil {

		return nil, Err

	}

	var Response AppleMusicResponse

	if Err := json.Unmarshal(Body, &Response); Err != nil {

		return nil, Err

	}

	if len(Response.Data) == 0 {

		return nil, fmt.Errorf("no album found with ID: %s", ID)

	}

	var AlbumData Album

	if Err := json.Unmarshal(Response.Data[0], &AlbumData); Err != nil {

		return nil, Err

	}

	return &AlbumData, nil

}

func (Ac *AppleClient) GetPlaylist(ID string) (*Playlist, error) {

	Params := map[string]string{
		"l": "en-us",
	}

	Body, Err := Ac.MakeRequest(fmt.Sprintf("/v1/catalog/%s/playlists/%s", Ac.CountryCode, ID), Params)

	if Err != nil {

		return nil, Err

	}

	var Response AppleMusicResponse

	if Err := json.Unmarshal(Body, &Response); Err != nil {

		return nil, Err

	}

	if len(Response.Data) == 0 {

		return nil, fmt.Errorf("no playlist found with ID: %s", ID)

	}

	var PlaylistData Playlist

	if Err := json.Unmarshal(Response.Data[0], &PlaylistData); Err != nil {

		return nil, Err

	}

	return &PlaylistData, nil

}

func (Al *Album) GetItems() []Song {

	return Al.Relationships.Tracks.Data

}

func (Al *Album) GetAllItems() ([]Song, error) {

	Tracks := make([]Song, 0, len(Al.Relationships.Tracks.Data))
	Tracks = append(Tracks, Al.Relationships.Tracks.Data...)

	// If we already have all tracks, we should return now

	Next := Al.Relationships.Tracks.Next

	if Next == "" {

		return Tracks, nil

	}

	for Next != "" {

		Body, Err := Client.MakeRequest(Next, map[string]string{})

		if Err != nil {

			return nil, Err

		}

		var Page Songs

		if Err := json.Unmarshal(Body, &Page); Err != nil {

			return nil, Err

		}

		Tracks = append(Tracks, Page.Data...)

		Next = Page.Next

	}

	return Tracks, nil

}

func (Pl *Playlist) GetItems() []Song {

	return Pl.Relationships.Tracks.Data

}

func (Pl *Playlist) GetAllItems() ([]Song, error) {

	Tracks := make([]Song, 0, len(Pl.Relationships.Tracks.Data))
	Tracks = append(Tracks, Pl.Relationships.Tracks.Data...)

	// If we already have all tracks, we should return now

	Next := Pl.Relationships.Tracks.Next

	if Next == "" {

		return Tracks, nil

	}

	FixedUpperLimit := 5000
	I := 0

	for Next != "" && I < FixedUpperLimit {

		Body, Err := Client.MakeRequest(Next, map[string]string{})

		if Err != nil {

			return nil, Err

		}

		var Page Songs

		if Err := json.Unmarshal(Body, &Page); Err != nil {

			return nil, Err

		}

		Tracks = append(Tracks, Page.Data...)

		Next = Page.Next
		I++

	}

	return Tracks, nil

}

func GetIDFromAppleMusicURL(URL string) (string, string, error) {

	// https://music.apple.com/us/album/album-name/1234567890
	// https://music.apple.com/us/song/song-name/1234567890
	// https://music.apple.com/us/playlist/playlist-name/pl.1234567890

	SongRegex := regexp.MustCompile(`music\.apple\.com/[a-z]{2}/song/[^/]+/(\d+)`)
	AlbumRegex := regexp.MustCompile(`music\.apple\.com/[a-z]{2}/album/[^/]+/(\d+)`)
	PlaylistRegex := regexp.MustCompile(`music\.apple\.com/[a-z]{2}/playlist/[^/]+/(pl\.[a-zA-Z0-9-]+)`)

	if Matches := SongRegex.FindStringSubmatch(URL); len(Matches) > 1 {

		return Matches[1], "song", nil

	}

	if Matches := AlbumRegex.FindStringSubmatch(URL); len(Matches) > 1 {

		return Matches[1], "album", nil

	}

	if Matches := PlaylistRegex.FindStringSubmatch(URL); len(Matches) > 1 {

		return Matches[1], "playlist", nil

	}

	return "", "", fmt.Errorf("could not extract ID from Apple Music URL: %s", URL)

}

func ManipulateArtworkURL(ArtworkURL string, Width int, Height int, Format string) string {

	if Format == "" {

		Format = "png"

	}

	Result := strings.ReplaceAll(ArtworkURL, "{w}", fmt.Sprintf("%d", Width))
	Result = strings.ReplaceAll(Result, "{h}", fmt.Sprintf("%d", Height))
	Result = strings.ReplaceAll(Result, "{f}", Format)

	return Result

}