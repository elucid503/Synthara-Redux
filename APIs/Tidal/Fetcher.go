package Tidal

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Utils"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Generic wrapper for API responses

var BaseAPIURL string
var BearerToken string
var HTTPClient = &http.Client{Timeout: 10 * time.Second}

// Token caching

var TokenMutex sync.RWMutex
var CachedToken string
var TokenExpiry time.Time

var StreamURLMutexes sync.Map

// Tidal OAuth credentials

const (
	TidalClientID     = "txNoH4kkV41MfH25"
	TidalClientSecret = "dQjy0MinCEvxi1O4UmxvxWnDjt4cgHBPw8ll6nYBk98="
	TidalAuthURL      = "https://auth.tidal.com/v1/oauth2/token"
)

// TokenResponse represents the OAuth token response from Tidal

type TokenResponse struct {

	ClientName  string `json:"clientName"`

	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`

	AccessToken string `json:"access_token"`

	ExpiresIn   int    `json:"expires_in"`

}

var DefaultHeaders = map[string]string{

	"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:144.0) Gecko/20100101 Firefox/144.0",
	"Accept":          "application/json, text/plain, */*",
	"Accept-Language": "en-US,en;q=0.9",
	"Connection":      "keep-alive",

}

func Init() {

	BaseAPIURL = strings.TrimSpace(os.Getenv("STREAMING_API_ENDPOINT"))

	if !strings.HasPrefix(BaseAPIURL, "http://") && !strings.HasPrefix(BaseAPIURL, "https://") {

		BaseAPIURL = "http://" + BaseAPIURL

	}

	BaseAPIURL = strings.TrimRight(BaseAPIURL, "/")

	// Fetch initial token

	if _, Err := GetBearerToken(); Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to fetch initial Tidal token: %s", Err.Error()))

	}

}

// GetBearerToken returns a valid bearer token, fetching a new one if expired
func GetBearerToken() (string, error) {

	TokenMutex.RLock()

	if CachedToken != "" && time.Now().Before(TokenExpiry) {

		Token := CachedToken
		TokenMutex.RUnlock()

		return Token, nil


	}

	TokenMutex.RUnlock()

	// Need to fetch new token

	TokenMutex.Lock()
	defer TokenMutex.Unlock()

	// Double-check after acquiring write lock

	if CachedToken != "" && time.Now().Before(TokenExpiry) {

		return CachedToken, nil

	}

	// Create URL-encoded form request

	FormData := url.Values{}
	
	FormData.Set("client_id", TidalClientID)
	FormData.Set("client_secret", TidalClientSecret)
	FormData.Set("grant_type", "client_credentials")

	Req, Err := http.NewRequest("POST", TidalAuthURL, strings.NewReader(FormData.Encode()))

	if Err != nil {

		return "", fmt.Errorf("failed to create token request: %w", Err)

	}

	Req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	Resp, Err := HTTPClient.Do(Req)
	
	if Err != nil {

		return "", fmt.Errorf("failed to fetch token: %w", Err)

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != http.StatusOK {

		BodyBytes, _ := io.ReadAll(Resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", Resp.StatusCode, string(BodyBytes))

	}

	var TokenResp TokenResponse

	if Err := json.NewDecoder(Resp.Body).Decode(&TokenResp); Err != nil {

		return "", fmt.Errorf("failed to parse token response: %w", Err)

	}

	// Cache the token with some buffer before expiry (5 minutes early)

	CachedToken = TokenResp.AccessToken
	TokenExpiry = time.Now().Add(time.Duration(TokenResp.ExpiresIn-300) * time.Second)
	BearerToken = CachedToken

	Utils.Logger.Info("Tidal API", "Tidal bearer token refreshed.")

	return CachedToken, nil

}

func DoGet(Endpoint string) (*http.Response, error) {

	Req, Err := http.NewRequest(http.MethodGet, Endpoint, nil)

	if Err != nil {

		return nil, Err

	}

	AddDefaultHeaders(Req)

	Resp, Err := HTTPClient.Do(Req)

	if Err == nil {

		return Resp, nil

	}

	var URLErr *url.Error

	if strings.HasPrefix(Endpoint, "https://") && errors.As(Err, &URLErr) && strings.Contains(Err.Error(), "first record does not look like a TLS handshake") {

		Fallback := "http://" + strings.TrimPrefix(Endpoint, "https://")

		FallbackReq, FallbackErr := http.NewRequest(http.MethodGet, Fallback, nil)

		if FallbackErr != nil {

			return nil, FallbackErr

		}

		AddDefaultHeaders(FallbackReq)

		return HTTPClient.Do(FallbackReq)

	}

	return nil, Err

}

func AddDefaultHeaders(Req *http.Request) {

	for Key, Value := range DefaultHeaders {

		Req.Header.Set(Key, Value)

	}

}

func Search(Query string, SearchType string) (*SearchResult, error) {

	Params := url.Values{}

	Params.Set(SearchType, Query)

	Endpoint := fmt.Sprintf("%s/search/?%s", BaseAPIURL, Params.Encode())

	Resp, Err := DoGet(Endpoint)

	if Err != nil {

		return nil, Err

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	var Wrapper Response[SearchResult]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return &Wrapper.Data, nil

}

func FetchStreaming(ID int64, Quality string) (*Streaming, error) {

	Utils.Logger.Info("Tidal API", fmt.Sprintf("Fetching streaming info for track %d", ID))

	Endpoint := fmt.Sprintf("%s/track/?id=%d&quality=%s", BaseAPIURL, ID, Quality)

	Resp, Err := DoGet(Endpoint)

	if Err != nil {

		return nil, Err

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	var Wrapper Response[Streaming]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return &Wrapper.Data, nil

}

func FetchInfo(ID int64) (*Info, error) {

	Endpoint := fmt.Sprintf("%s/info/?id=%d", BaseAPIURL, ID)

	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	var Wrapper Response[Info]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}
	
	return &Wrapper.Data, nil

}

func FetchDash(ID int64, Quality string) ([]byte, error) {

	Endpoint := fmt.Sprintf("%s/dash/?id=%d&quality=%s", BaseAPIURL, ID, Quality)
	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	return io.ReadAll(Resp.Body)
}

func FetchCover(ID int64, Query string) ([]Cover, error) {

	Params := url.Values{}

	if ID != 0 { Params.Set("id", fmt.Sprintf("%d", ID)) }
	if Query != "" { Params.Set("q", Query) }

	Endpoint := fmt.Sprintf("%s/cover/?%s", BaseAPIURL, Params.Encode())
	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	var Wrapper Response[[]Cover]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return Wrapper.Data, nil

}

func FetchAlbum(ID int64) (*Album, error) {

	Endpoint := fmt.Sprintf("%s/album/?id=%d", BaseAPIURL, ID)
	Resp, Err := DoGet(Endpoint)

	if Err != nil {

		return nil, Err

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 { 
		
		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode) 
	
	}

	var Wrapper Response[Album]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return &Wrapper.Data, nil

}

func FetchPlaylist(ID string) (map[string]interface{}, error) {

	Endpoint := fmt.Sprintf("%s/playlist/?id=%s", BaseAPIURL, ID)
	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {
		
		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)
	
	}

	var Wrapper Response[map[string]interface{}]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return Wrapper.Data, nil

}

func FetchArtist(ID int64, Full bool) ([]Artist, error) {

	Params := url.Values{}

	if (Full) {

		Params.Set("f", fmt.Sprintf("%d", ID)) 

	} else {

		Params.Set("id", fmt.Sprintf("%d", ID))

	}
	
	Endpoint := fmt.Sprintf("%s/artist/?%s", BaseAPIURL, Params.Encode())
	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {
		
		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode) 
		
	} 
	
	var Wrapper Response[[]Artist]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return Wrapper.Data, nil

}

func FetchLyrics(ID int64) ([]Lyrics, error) { // not really good

	Endpoint := fmt.Sprintf("%s/lyrics/?id=%d", BaseAPIURL, ID)
	Resp, Err := DoGet(Endpoint)

	if Err != nil { return nil, Err }

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 { 
		
		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode) 
	
	}

	var Wrapper Response[[]Lyrics]

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		return nil, Err

	}

	return Wrapper.Data, nil

}

// FetchAlbumTracks fetches all tracks in an album
func FetchAlbumTracks(AlbumID int64) ([]Song, error) {

	Cache := Globals.GetOrCreateCache("TidalAlbumTracks")
	Key := fmt.Sprintf("%d", AlbumID)

	if Cached, Exists := Cache.Get(Key); Exists {

		if Songs, Ok := Cached.([]Song); Ok {

			Copy := make([]Song, len(Songs))
			copy(Copy, Songs)

			return Copy, nil

		}

	}


	// Get bearer token

	Token, Err := GetBearerToken()

	if Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to get bearer token: %s", Err.Error()))
		return nil, Err

	}
	
	Songs := make([]Song, 0)
	Limit := 100
	Offset := 0

	for {

		ItemsEndpoint := fmt.Sprintf("https://api.tidal.com/v1/albums/%d/items?countryCode=US&limit=%d&offset=%d", AlbumID, Limit, Offset)

		Request, Err := http.NewRequest("GET", ItemsEndpoint, nil)

		if Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to create album items request: %s", Err.Error()))
			return nil, Err

		}

		// Set headers

		Request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
		Request.Header.Set("Accept", "application/json")
		Request.Header.Set("Accept-Language", "en-US,en;q=0.9")
		Request.Header.Set("Authorization", "Bearer "+Token)

		ItemsResp, ItemsErr := HTTPClient.Do(Request)

		if ItemsErr != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to fetch album items for ID %d: %s", AlbumID, ItemsErr.Error()))
			return nil, ItemsErr

		}

		defer ItemsResp.Body.Close()

		if ItemsResp.StatusCode != 200 {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Album items request returned HTTP %d for ID %d", ItemsResp.StatusCode, AlbumID))
			return nil, fmt.Errorf("HTTP %d fetching album items", ItemsResp.StatusCode)

		}

		var ItemsData AlbumItems

		if Err := json.NewDecoder(ItemsResp.Body).Decode(&ItemsData); Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to decode album items for ID %d: %s", AlbumID, Err.Error()))
			return nil, Err

		}

		if len(ItemsData.Items) == 0 {

			Utils.Logger.Warn("Tidal API", fmt.Sprintf("Album %d has no items", AlbumID))
			return nil, fmt.Errorf("album has no tracks")

		}

		for _, Item := range ItemsData.Items {

			Song := TrackToSong(Item.Item)
			Songs = append(Songs, Song)

		}

		Offset += len(ItemsData.Items)

		if len(ItemsData.Items) < Limit || Offset >= ItemsData.TotalNumberOfItems {

			break

		}

	}

	Cache.Set(Key, Songs, 1 * time.Hour) // 1 hour TTL

	Utils.Logger.Info("Tidal API", fmt.Sprintf("Fetched %d tracks from album %d", len(Songs), AlbumID))

	return Songs, nil

}

func FetchPlaylistTracks(PlaylistID int64) ([]Song, error) {

	// Get bearer token
	Token, Err := GetBearerToken()

	if Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to get bearer token: %s", Err.Error()))
		return nil, Err

	}

	Songs := make([]Song, 0)
	Offset := 0
	Limit := 100

	for {

		Endpoint := fmt.Sprintf("https://api.tidal.com/v1/playlists/%d/items?countryCode=US&limit=%d&offset=%d", PlaylistID, Limit, Offset)

		Utils.Logger.Info("Tidal API", fmt.Sprintf("Fetching playlist items from: %s", Endpoint))

		Request, Err := http.NewRequest("GET", Endpoint, nil)

		if Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to create playlist items request: %s", Err.Error()))
			return nil, Err

		}

		// Set headers

		Request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
		Request.Header.Set("Accept", "application/json")
		Request.Header.Set("Accept-Language", "en-US,en;q=0.9")
		Request.Header.Set("Authorization", "Bearer " + Token)

		Resp, Err := HTTPClient.Do(Request)

		if Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to fetch playlist items for ID %d: %s", PlaylistID, Err.Error()))
			return nil, Err

		}

		defer Resp.Body.Close()

		if Resp.StatusCode != 200 {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Playlist items request returned HTTP %d for ID %d", Resp.StatusCode, PlaylistID))
			return nil, fmt.Errorf("HTTP %d fetching playlist items", Resp.StatusCode)

		}

		var Wrapper MixItems

		if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

			Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to decode playlist items for ID %d: %s", PlaylistID, Err.Error()))
			return nil, Err

		}

		if len(Wrapper.Items) == 0 { break }

		for _, Item := range Wrapper.Items {

			Song := TrackToSong(Item.Item)
			Songs = append(Songs, Song)

		}

		Offset += len(Wrapper.Items)

		if Offset >= Wrapper.TotalNumberOfItems {

			break

		}

	}

	if len(Songs) == 0 {

		Utils.Logger.Warn("Tidal API", fmt.Sprintf("Playlist %d has no items", PlaylistID))
		return nil, fmt.Errorf("playlist has no tracks")

	}

	Utils.Logger.Info("Tidal API", fmt.Sprintf("Fetched %d tracks from playlist %d", len(Songs), PlaylistID))

	return Songs, nil

}

// FetchTrackMix fetches the mix ID for a track (used for AutoPlay)
func FetchTrackMix(TrackID int64) (string, error) {

	// First, try to get mix ID from track info
	Info, Err := FetchInfo(TrackID)
	
	if Err != nil { return "", Err }

	if Info.Mixes.TrackMix != "" {

		return Info.Mixes.TrackMix, nil

	}

	return "", fmt.Errorf("no mix available for track %d", TrackID)
}

// FetchMixItems fetches all tracks in a mix (for AutoPlay)
func FetchMixItems(MixID string) ([]Song, error) {

	// We need bearer token

	Token, Err := GetBearerToken()

	if Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to get bearer token: %s", Err.Error()))
		return nil, Err

	}

	Endpoint := fmt.Sprintf("https://api.tidal.com/v1/mixes/%s/items?countryCode=US", MixID)

	Request, Err := http.NewRequest("GET", Endpoint, nil)

	if Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to create mix items request: %s", Err.Error()))
		return nil, Err

	}

	Request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
	Request.Header.Set("Accept", "application/json")
	Request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	Request.Header.Set("Authorization", "Bearer " + Token)

	Resp, Err := HTTPClient.Do(Request)

	if Err != nil { 

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to fetch mix items: %s", Err.Error()))
		return nil, Err 

	}

	defer Resp.Body.Close()

	if Resp.StatusCode != 200 {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Mix items request failed with HTTP %d", Resp.StatusCode))
		return nil, fmt.Errorf("HTTP %d", Resp.StatusCode)

	}

	var Wrapper MixItems

	if Err := json.NewDecoder(Resp.Body).Decode(&Wrapper); Err != nil {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Failed to decode mix items response: %s", Err.Error()))
		return nil, Err

	}

	Songs := make([]Song, 0, len(Wrapper.Items))
	
	for _, Item := range Wrapper.Items {

		Song := TrackToSong(Item.Item)
		Songs = append(Songs, Song)

	}

	return Songs, nil

}

// SearchSongs searches for songs and returns Song structs
func SearchSongs(Query string) ([]Song, error) {

	Result, Err := Search(Query, SearchTypeSong)

	if Err != nil {

		return nil, Err

	}

	Songs := make([]Song, 0, len(Result.Items))

	for _, Track := range Result.Items {

		Song := TrackToSong(Track)
		Songs = append(Songs, Song)

	}

	return Songs, nil

}

// GetSong fetches a single song by ID
func GetSong(TrackID int64) (Song, error) {

	Info, Err := FetchInfo(TrackID)

	if Err != nil {

		return Song{}, Err

	}

	return InfoToSong(*Info), nil

}

// GetStreamURL fetches the direct streaming URL for a track
func GetStreamURL(TrackID int64) (string, error) {

	Cache := Globals.GetOrCreateCache("TidalStreamURLs")
	Key := fmt.Sprintf("%d", TrackID)

	if Cached, Exists := Cache.Get(Key); Exists {

		if URL, Ok := Cached.(string); Ok {

			return URL, nil

		}

	}

	MutexInterface, _ := StreamURLMutexes.LoadOrStore(Key, &sync.Mutex{})
	Mutex := MutexInterface.(*sync.Mutex)

	Mutex.Lock()
	defer Mutex.Unlock()

	if Cached, Exists := Cache.Get(Key); Exists {

		if URL, Ok := Cached.(string); Ok {

			return URL, nil

		}

	}

	Streaming, Err := FetchStreaming(TrackID, QualityLow)

	if Err != nil {

		return "", Err

	}

	DirectURL, _, _, Err := ParseManifest(Streaming.Manifest)

	if Err != nil {

		return "", Err

	}

	if DirectURL == "" {

		return "", fmt.Errorf("no direct URL available for track %d", TrackID)

	}

	Cache.Set(Key, DirectURL, 1 * time.Hour) // 1 hour TTL

	return DirectURL, nil

}

// ParseManifest decodes a base64 manifest and extracts playable URLs
// Returns: directURL (for BTS), initURL + segmentURLs (for DASH), error
func ParseManifest(ManifestBase64 string) (string, string, []string, error) {

	// Decode base64

	ManifestBytes, Err := base64.StdEncoding.DecodeString(ManifestBase64)
	
	if Err != nil {

		return "", "", nil, fmt.Errorf("failed to decode manifest: %w", Err)

	}

	ManifestStr := string(ManifestBytes)

	// Checks if BTS format (JSON)

	if strings.HasPrefix(strings.TrimSpace(ManifestStr), "{") {

		var BTS BTSManifest
		
		if Err := json.Unmarshal(ManifestBytes, &BTS); Err != nil {

			return "", "", nil, fmt.Errorf("failed to parse BTS manifest: %w", Err)

		}

		if len(BTS.URLs) == 0 {

			return "", "", nil, fmt.Errorf("no URLs in BTS manifest")

		}

		// Return direct URL

		return BTS.URLs[0], "", nil, nil
		
	}

	// DASH format (XML)

	var MPD DASHManifest
	
	if Err := xml.Unmarshal(ManifestBytes, &MPD); Err != nil {

		return "", "", nil, fmt.Errorf("failed to parse DASH manifest: %w", Err)

	}

	SegTemplate := MPD.Period.AdaptationSet.Representation.SegmentTemplate
	InitURL := SegTemplate.Initialization
	MediaTemplate := SegTemplate.Media

	// Fallback: regex extraction

	if InitURL == "" || MediaTemplate == "" {
		
		InitRe := regexp.MustCompile(`initialization="([^"]+)"`)
		MediaRe := regexp.MustCompile(`media="([^"]+)"`)

		if Match := InitRe.FindStringSubmatch(ManifestStr); len(Match) > 1 {

			InitURL = Match[1]

		}
		
		if Match := MediaRe.FindStringSubmatch(ManifestStr); len(Match) > 1 {

			MediaTemplate = Match[1]

		}

	}

	if InitURL == "" {

		return "", "", nil, fmt.Errorf("no initialization URL in DASH manifest")

	}

	// Unescape HTML entities

	InitURL = strings.ReplaceAll(InitURL, "&amp;", "&")
	MediaTemplate = strings.ReplaceAll(MediaTemplate, "&amp;", "&")

	// Calculate segment count

	SegmentCount := 0
	
	for _, Seg := range SegTemplate.Timeline.Segments {

		SegmentCount += Seg.Repeat + 1

	}

	// Fallback: regex for segment count

	if SegmentCount == 0 {
		
		SegRe := regexp.MustCompile(`<S d="\d+"(?: r="(\d+)")?`)
		Matches := SegRe.FindAllStringSubmatch(ManifestStr, -1)
		
		for _, Match := range Matches {

			Repeat := 0

			if len(Match) > 1 && Match[1] != "" {

				fmt.Sscanf(Match[1], "%d", &Repeat)

			}

			SegmentCount += Repeat + 1

		}
	}

	// Generate segment URLs
	
	var SegmentURLs []string
	
	for I := 1; I <= SegmentCount; I++ {

		SegmentURL := strings.ReplaceAll(MediaTemplate, "$Number$", fmt.Sprintf("%d", I))
		SegmentURLs = append(SegmentURLs, SegmentURL)

	}

	return "", InitURL, SegmentURLs, nil
	
}

// FetchSearchSuggestions fetches autocomplete suggestions from Tidal
func GetSearchSuggestions(Query string) []SearchSuggestion {

	Results := []SearchSuggestion{}

	if Query == "" {

		return Results

	}

	// Get a fresh token if needed

	Token, TokenErr := GetBearerToken()
	if TokenErr != nil {

		Utils.Logger.Error("Tidal API", "Failed to get bearer token: " + TokenErr.Error())
		return Results

	}

	// Build the suggestions URL

	Endpoint := fmt.Sprintf("https://tidal.com/v2/client-suggestions/?countryCode=US&explicit=true&hybrid=true&query=%s", url.QueryEscape(Query))

	Request, Err := http.NewRequest("GET", Endpoint, nil)

	if Err != nil {

		Utils.Logger.Error("Tidal API", "Failed to create suggestions request: " + Err.Error())
		return Results

	}

	// Set headers

	Request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
	Request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	Request.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	Request.Header.Set("Authorization", "Bearer " + Token)

	Response, Err := HTTPClient.Do(Request)

	if Err != nil {

		Utils.Logger.Error("Tidal API", "Failed to fetch suggestions: " + Err.Error())
		return Results

	}

	defer Response.Body.Close()

	if Response.StatusCode != 200 {

		Utils.Logger.Error("Tidal API", fmt.Sprintf("Suggestions request failed with status %d", Response.StatusCode))
		return Results

	}

	// Handle gzip/compressed response

	var Reader io.Reader = Response.Body

	if Response.Header.Get("Content-Encoding") == "gzip" {

		GzipReader, Err := gzip.NewReader(Response.Body)
		if Err != nil {

			Utils.Logger.Error("Tidal API", "Failed to create gzip reader: " + Err.Error())
			return Results

		}
		defer GzipReader.Close()
		Reader = GzipReader

	}

	Body, Err := io.ReadAll(Reader)

	if Err != nil {

		Utils.Logger.Error("Tidal API", "Failed to read suggestions response: " + Err.Error())
		return Results

	}

	var SuggestionsResp SuggestionsResponse

	if Err := json.Unmarshal(Body, &SuggestionsResp); Err != nil {

		Utils.Logger.Error("Tidal API", "Failed to parse suggestions response: " + Err.Error())
		return Results

	}

	// Process direct hits first (tracks, albums, artists, playlists)

	for _, Hit := range SuggestionsResp.DirectHits {

		ValueBytes, Err := json.Marshal(Hit.Value)
		if Err != nil {

			continue

		}

		switch Hit.Type {

		case "TRACKS":

			var Track DirectHitTrack
			if json.Unmarshal(ValueBytes, &Track) == nil {

				ArtistName := ""
				if len(Track.Artists) > 0 {

					ArtistName = Track.Artists[0].Name

				}

				Results = append(Results, SearchSuggestion{

					Metadata: SearchSuggestionMetadata{

						Type:     "Song",
						ID:       fmt.Sprintf("%d", Track.ID),
						Title:    Track.Title,
						Subtitle: ArtistName,

					},

				})

			}

		case "ALBUMS":

			var Album DirectHitAlbum
			if json.Unmarshal(ValueBytes, &Album) == nil {

				ArtistName := ""
				if len(Album.Artists) > 0 {

					ArtistName = Album.Artists[0].Name

				}

				Results = append(Results, SearchSuggestion{

					Metadata: SearchSuggestionMetadata{

						Type:     "Album",
						ID:       fmt.Sprintf("%d", Album.ID),
						Title:    Album.Title,
						Subtitle: ArtistName,

					},

				})

			}

		case "ARTISTS":

			var Artist DirectHitArtist
			if json.Unmarshal(ValueBytes, &Artist) == nil {

				Results = append(Results, SearchSuggestion{

					Metadata: SearchSuggestionMetadata{

						Type:     "Artist",
						ID:       fmt.Sprintf("%d", Artist.ID),
						Title:    Artist.Name,
						Subtitle: "Artist",

					},

				})

			}

		case "PLAYLISTS":

			var Playlist DirectHitPlaylist
			if json.Unmarshal(ValueBytes, &Playlist) == nil {

				Results = append(Results, SearchSuggestion{

					Metadata: SearchSuggestionMetadata{

						Type:     "Playlist",
						ID:       Playlist.UUID,
						Title:    Playlist.Title,
						Subtitle: "Playlist",

					},

				})

			}

		}

	}

	// Process text suggestions

	for _, Suggestion := range SuggestionsResp.Suggestions {

		Results = append(Results, SearchSuggestion{

			Text: Suggestion.Query,

		})

	}

	return Results

}