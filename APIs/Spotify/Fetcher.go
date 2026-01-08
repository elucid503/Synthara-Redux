package Spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var Client *Spotify

type Token struct {

	Token   string `json:"access_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
	Time    int

}

type Spotify struct {

	client_id     string
	client_secret string
	AccessToken   Token

}

type Followers struct {

	Href  string `json:"href"`
	Total int    `json:"total"`

}

type Image struct {

	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
	
}

type LinkedFrom struct {

	ExternalUrls map[string]string `json:"external_urls"`
	Href         string            `json:"href"`
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	URL          string            `json:"url"`

}

type Restriction struct {

	Reason string `json:"reason"`

}

type Copyright struct {

	Text string `json:"text"`
	Type string `json:"type"`

}

type ExternalIDs struct {

	ISRC string `json:"isrc"`
	EAN  string `json:"ean"`
	UPC  string `json:"upc"`

}

type AlbumShort struct {

	AlbumType        string            `json:"album_type"`
	TotalTracks      int               `json:"total_tracks"`
	AvailableMarkets []string          `json:"available_markets"`
	External         map[string]string `json:"external_urls"`
	Href             string            `json:"href"`
	ID               string            `json:"id"`
	Images           []Image           `json:"images"`
	Name             string            `json:"name"`
	ReleaseDate      string            `json:"release_date"`
	ReleaseDatePrecision string        `json:"release_date_precision"`
	Restrictions     Restriction       `json:"restrictions"`
	Type             string            `json:"type"`
	URL              string            `json:"url"`
	Artists          []Artists         `json:"artists"`

}

type PagingAlbumTracks struct {

	Href     string      `json:"href"`
	Limit    int         `json:"limit"`
	Next     string      `json:"next"`
	Offset   int         `json:"offset"`
	Previous string      `json:"previous"`
	Total    int         `json:"total"`
	Items    []AlbumItem `json:"items"`

}

type PagingPlaylistTracks struct {

	Href     string         `json:"href"`
	Limit    int            `json:"limit"`
	Next     string         `json:"next"`
	Offset   int            `json:"offset"`
	Previous string         `json:"previous"`
	Total    int            `json:"total"`
	Items    []PlaylistItem `json:"items"`

}

type TrackSearchItem struct {

	Artists          []Artists         `json:"artists"`
	AvailableMarkets []string          `json:"available_markets"`
	DiscNumber       int               `json:"disc_number"`
	DurationMS       int               `json:"duration_ms"`
	Explicit         bool              `json:"explicit"`
	ExternalUrls     map[string]string `json:"external_urls"`
	Href             string            `json:"href"`
	ID               string            `json:"id"`
	IsPlayable       bool              `json:"is_playable"`
	LinkedFrom       LinkedFrom        `json:"linked_from"`
	Restrictions     Restriction       `json:"restrictions"`
	Name             string            `json:"name"`
	PreviewURL       string            `json:"preview_url"`
	TrackNumber      int               `json:"track_number"`
	Type             string            `json:"type"`
	URL              string            `json:"url"`
	IsLocal          bool              `json:"is_local"`

}

type AlbumSimple struct {

	AlbumType        string            `json:"album_type"`
	TotalTracks      int               `json:"total_tracks"`
	AvailableMarkets []string          `json:"available_markets"`
	ExternalUrls     map[string]string `json:"external_urls"`
	Href             string            `json:"href"`
	ID               string            `json:"id"`
	Images           []Image           `json:"images"`
	Name             string            `json:"name"`
	ReleaseDate      string            `json:"release_date"`
	ReleaseDatePrecision string        `json:"release_date_precision"`
	Restrictions     Restriction       `json:"restrictions"`
	Type             string            `json:"type"`
	URL              string            `json:"url"`
	Artists          []Artists         `json:"artists"`

}

type Artists struct {

	External  map[string]string `json:"external_urls"`

	Followers Followers         `json:"followers"`

	Genres    []string          `json:"genres"`
	Href      string            `json:"href"`
	ID        string            `json:"id"`

	Images    []Image           `json:"images"`

	Name       string `json:"name"`
	Popularity int    `json:"popularity"`
	Type       string `json:"type"`
	URL        string `json:"url"`

}

type User struct {

	DisplayName  string            `json:"display_name"`
	ExternalUrls map[string]string `json:"external_urls"`

	Followers    Followers         `json:"followers"`

	Href         string            `json:"href"`
	ID           string            `json:"id"`

	Images       []Image           `json:"images"`

	Type string `json:"type"`
	URL  string `json:"url"`

}

type AlbumItem struct {

	Artists          []Artists         `json:"artists"`
	AvailableMarkets []string          `json:"available_markets"`
	DiscNumber       int               `json:"disc_number"`
	DurationMS       int               `json:"duration_ms"`
	Explicit         bool              `json:"explicit"`
	ExternalUrls     map[string]string `json:"external_urls"`
	Href             string            `json:"href"`
	ID               string            `json:"id"`
	IsPlayable       bool              `json:"is_playable"`
	LinkedFrom       LinkedFrom        `json:"linked_from"`
	Restrictions     Restriction       `json:"restrictions"`
	Name             string            `json:"name"`
	PreviewURL       string            `json:"preview_url"`
	TrackNumber      int               `json:"track_number"`
	Type             string            `json:"type"`
	URL              string            `json:"url"`
	IsLocal          bool              `json:"is_local"`

}

type Album struct {

	AlbumType        string            `json:"album_type"`
	TotalTracks      int               `json:"total_tracks"`
	AvailableMarkets []string          `json:"available_markets"`
	ExternalUrls     map[string]string `json:"external_urls"`
	Href             string            `json:"href"`
	ID               string            `json:"id"`
	Images           []Image           `json:"images"`
	Name             string            `json:"name"`
	ReleaseDate      string            `json:"release_date"`
	ReleaseDatePrecision string        `json:"release_date_precision"`
	Restrictions     Restriction       `json:"restrictions"`
	Type             string            `json:"type"`
	URL              string            `json:"url"`
	Artists          []Artists         `json:"artists"`
	Tracks           PagingAlbumTracks `json:"tracks"`
	Copyrights       []Copyright       `json:"copyrights"`
	ExternalIDs       ExternalIDs      `json:"external_ids"`
	Genres           []string          `json:"genres"`
	Label            string            `json:"label"`
	Popularity       int               `json:"popularity"`

}

type Track struct {

	Album     AlbumShort `json:"album"`
	Name      string     `json:"name"`
	Popularity int       `json:"popularity"`
	TrackNumber int      `json:"track_number"`
	DiscNumber int       `json:"disc_number"`
	ID         string    `json:"id"`
	URI        string    `json:"uri"`
	Explicit   bool      `json:"explicit"`
	DurationMS int       `json:"duration_ms"`
	Artists    []Artists `json:"artists"`
	PreviewURL string `json:"preview_url"`
	IsPlayable bool   `json:"is_playable"`

}

type PlaylistItem struct {

	AddedAt string `json:"added_at"`
	AddedBy User   `json:"added_by"`
	IsLocal bool   `json:"is_local"`
	Track   Track  `json:"track"`

}

type Playlist struct {

	Collaborative bool              `json:"collaborative"`
	Description   string            `json:"description"`
	External      map[string]string `json:"external_urls"`
	Followers     Followers         `json:"followers"`
	Href          string            `json:"href"`
	ID            string            `json:"id"`
	Images        []Image           `json:"images"`
	Name          string            `json:"name"`
	Owner         User              `json:"owner"`
	Public        bool              `json:"public"`
	SnapshotID    string            `json:"snapshot_id"`
	Tracks        PagingPlaylistTracks `json:"tracks"`
	Type          string            `json:"type"`
	URL           string            `json:"url"`

}

type SearchTrack struct {
	
	Href     string           `json:"href"`
	Limit    int              `json:"limit"`
	Next     string           `json:"next"`
	Offset   int              `json:"offset"`
	Previous string           `json:"previous"`
	Total    int              `json:"total"`
	Items    []TrackSearchItem `json:"items"`

}

type SearchArtist struct {

	Href     string    `json:"href"`
	Limit    int       `json:"limit"`
	Next     string    `json:"next"`
	Offset   int       `json:"offset"`
	Previous string    `json:"previous"`
	Total    int       `json:"total"`
	Items    []Artists `json:"items"`

}

type SearchAlbum struct {
	
	Href     string        `json:"href"`
	Limit    int           `json:"limit"`
	Next     string        `json:"next"`
	Offset   int           `json:"offset"`
	Previous string        `json:"previous"`
	Total    int           `json:"total"`
	Items    []AlbumSimple `json:"items"`

}

type SearchPlaylist struct {

	Href     string     `json:"href"`
	Limit    int        `json:"limit"`
	Next     string     `json:"next"`
	Offset   int        `json:"offset"`
	Previous string     `json:"previous"`
	Total    int        `json:"total"`
	Items    []Playlist `json:"items"`

}

type Search struct {

	Tracks   SearchTrack    `json:"tracks"`
	Artists  SearchArtist   `json:"artists"`
	Album    SearchAlbum    `json:"albums"`
	Playlist SearchPlaylist `json:"playlists"`
	
}

func (Al *Album) GetItems() []AlbumItem {

	return Al.Tracks.Items

}

func (Pl *Playlist) GetItems() []PlaylistItem {

	return Pl.Tracks.Items

}

func Initialize(ID, Secret string) *Spotify {

	Client = &Spotify{
		client_id:     ID,
		client_secret: Secret,
		AccessToken:   Token{},
	}

	return Client

}

func (St *Spotify) Refresh() error {

	if St.AccessToken.Token == "" {

		if Err := St.NewToken(); Err != nil {

			return Err

		}

	}

	if int(time.Now().Unix())-St.AccessToken.Time >= St.AccessToken.Expires {

		if Err := St.NewToken(); Err != nil {

			return Err

		}

	}

	return nil

}

func (St *Spotify) MakeRequest(URL string) ([]byte, error) {

	if Err := St.Refresh(); Err != nil {

		return nil, Err

	}

	Req, Err := http.NewRequest("GET", URL, nil)

	if Err != nil {

		return nil, Err

	}

	Req.Header.Set("Authorization", fmt.Sprintf("%s %s", St.AccessToken.Type, St.AccessToken.Token))

	Client := &http.Client{}

	Resp, Err := Client.Do(Req)

	if Err != nil {

		return nil, Err

	}
	
	defer Resp.Body.Close()

	Body, Err := io.ReadAll(Resp.Body)
	if Err != nil {

		return nil, Err

	}

	if Resp.StatusCode != http.StatusOK {

		ErrUnexpectedStatus := fmt.Errorf("unexpected status: %d", Resp.StatusCode)

		return nil, ErrUnexpectedStatus

	}

	return Body, nil

}

func (St *Spotify) NewToken() error {

	Data := url.Values{}

	Data.Set("grant_type", "client_credentials")
	Data.Set("client_id", St.client_id)
	Data.Set("client_secret", St.client_secret)

	Req, Err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(Data.Encode()))
	if Err != nil {

		return Err

	}

	Req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	Client := &http.Client{}

	Resp, Err := Client.Do(Req)
	if Err != nil {

		return Err

	}
	defer Resp.Body.Close()

	Body, Err := io.ReadAll(Resp.Body)
	if Err != nil {

		return Err

	}

	if Resp.StatusCode != http.StatusOK {

		ErrUnexpectedStatus := fmt.Errorf("unexpected status: %d", Resp.StatusCode)

		return ErrUnexpectedStatus

	}

	var TokenVar Token

	Err = json.Unmarshal(Body, &TokenVar)
	if Err != nil {

		return Err

	}

	TokenVar.Time = int(time.Now().Unix())

	St.AccessToken = TokenVar

	return nil

}

func (St *Spotify) GetArtist(ID string) (*Artists, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/artists/%s", ID))
	if Err != nil {

		return nil, Err

	}

	var Info Artists

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info, nil

}

func (St *Spotify) GetUser(ID string) (*User, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/users/%s", ID))
	if Err != nil {

		return nil, Err

	}

	var Info User

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info, nil

}

func (St *Spotify) GetAlbum(ID string) (*Album, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/albums/%s", ID))
	if Err != nil {

		return nil, Err

	}

	var Info Album

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info, nil

}

func (St *Spotify) GetTrack(ID string) (*Track, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/tracks/%s", ID))
	if Err != nil {

		return nil, Err

	}

	var Info Track

	Err = json.Unmarshal(Body, &Info)

	if Err != nil {

		return nil, Err

	}

	return &Info, nil

}

func (St *Spotify) GetPlaylist(ID string) (*Playlist, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", ID))
	if Err != nil {

		return nil, Err

	}

	var Info Playlist

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info, nil

}

func (St *Spotify) SearchTrack(Q string, Limit int) (*SearchTrack, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=%d", strings.ReplaceAll(Q, " ", ""), Limit))
	if Err != nil {

		return nil, Err

	}

	var Info Search

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info.Tracks, nil

}

func (St *Spotify) SearchArtist(Q string, Limit int) (*SearchArtist, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=artist&limit=%d", strings.ReplaceAll(Q, " ", ""), Limit))
	if Err != nil {

		return nil, Err

	}

	var Info Search

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info.Artists, nil

}

func (St *Spotify) SearchAlbum(Q string, Limit int) (*SearchAlbum, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=album&limit=%d", strings.ReplaceAll(Q, " ", ""), Limit))
	if Err != nil {

		return nil, Err

	}

	var Info Search

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info.Album, nil

}

func (St *Spotify) SearchPlaylist(Q string, Limit int) (*SearchPlaylist, error) {

	Body, Err := St.MakeRequest(fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=playlist&limit=%d", strings.ReplaceAll(Q, " ", ""), Limit))
	if Err != nil {

		return nil, Err

	}

	var Info Search

	Err = json.Unmarshal(Body, &Info)
	if Err != nil {

		return nil, Err

	}

	return &Info.Playlist, nil

}