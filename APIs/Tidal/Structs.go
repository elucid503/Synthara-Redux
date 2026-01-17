package Tidal

import "encoding/xml"

type Response[T any] struct {
	Data T `json:"data"`
}

type Track struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	Duration       int      `json:"duration"`
	ReplayGain     float64  `json:"replayGain"`
	Peak           float64  `json:"peak"`
	AllowStreaming bool     `json:"allowStreaming"`
	StreamReady    bool     `json:"streamReady"`
	AudioQuality   string   `json:"audioQuality"`
	AudioModes     []string `json:"audioModes"`
	TrackNumber    int      `json:"trackNumber"`
	VolumeNumber   int      `json:"volumeNumber"`
	Popularity     int      `json:"popularity"`
	Explicit       bool     `json:"explicit"`
	Album          Album    `json:"album"`
	Artist         Artist   `json:"artist"`
	Artists        []Artist `json:"artists"`
	URL            string   `json:"url"`
	ISRC           string   `json:"isrc"`
	Mixes          Mixes    `json:"mixes"`
}

type Info struct {
	ID                     int64         `json:"id"`
	Title                  string        `json:"title"`
	Duration               int           `json:"duration"`
	ReplayGain             float64       `json:"replayGain"`
	Peak                   float64       `json:"peak"`
	AllowStreaming         bool          `json:"allowStreaming"`
	StreamReady            bool          `json:"streamReady"`
	PayToStream            bool          `json:"payToStream"`
	AdSupportedStreamReady bool          `json:"adSupportedStreamReady"`
	DjReady                bool          `json:"djReady"`
	StemReady              bool          `json:"stemReady"`
	StreamStartDate        string        `json:"streamStartDate"`
	PremiumStreamingOnly   bool          `json:"premiumStreamingOnly"`
	TrackNumber            int           `json:"trackNumber"`
	VolumeNumber           int           `json:"volumeNumber"`
	Version                *string       `json:"version"`
	Popularity             int           `json:"popularity"`
	Copyright              string        `json:"copyright"`
	BPM                    int           `json:"bpm"`
	Key                    string        `json:"key"`
	KeyScale               string        `json:"keyScale"`
	URL                    string        `json:"url"`
	ISRC                   string        `json:"isrc"`
	Editable               bool          `json:"editable"`
	Explicit               bool          `json:"explicit"`
	AudioQuality           string        `json:"audioQuality"`
	AudioModes             []string      `json:"audioModes"`
	MediaMetadata          MediaMetadata `json:"mediaMetadata"`
	Upload                 bool          `json:"upload"`
	AccessType             string        `json:"accessType"`
	Spotlighted            bool          `json:"spotlighted"`
	Artist                 InfoArtist    `json:"artist"`
	Artists                []InfoArtist  `json:"artists"`
	Album                  InfoAlbum     `json:"album"`
	Mixes                  Mixes         `json:"mixes"`

}

type MediaMetadata struct {
    Tags []string `json:"tags"`
}

type Mixes struct {
    TrackMix string `json:"TRACK_MIX"`
}

type InfoAlbum struct {
    ID            int64   `json:"id"`
    Title         string  `json:"title"`
    Cover         string  `json:"cover"`
    VibrantColor  string  `json:"vibrantColor"`
    VideoCover    *string `json:"videoCover"`
}

type InfoArtist struct {
    ID      int64   `json:"id"`
    Name    string  `json:"name"`
    Handle  *string `json:"handle"`
    Type    string  `json:"type"`
    Picture string  `json:"picture"`
}

type Album struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	Cover          string   `json:"cover"`
	ReleaseDate    string   `json:"releaseDate"`
	AudioQuality   string   `json:"audioQuality"`
	NumberOfTracks int      `json:"numberOfTracks"`
	Duration       int      `json:"duration"`
	Explicit       bool     `json:"explicit"`
	Artist         Artist   `json:"artist"`
	Artists        []Artist `json:"artists"`
	URL            string   `json:"url"`
}

type Artist struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

type Playlist struct {
	UUID           string  `json:"uuid"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	NumberOfTracks int     `json:"numberOfTracks"`
	Duration       int     `json:"duration"`
	Image          string  `json:"image"`
	Tracks         []Track `json:"tracks"`
	URL            string  `json:"url"`
}

type Cover struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Size1280 string `json:"1280"`
	Size640  string `json:"640"`
	Size80   string `json:"80"`
}

type Lyrics struct {
	TrackID       int64  `json:"trackId"`
	Lyrics        string `json:"lyrics"`
	IsRightToLeft bool   `json:"isRightToLeft"`
	Provider      string `json:"lyricsProvider"`
	Subtitles     string `json:"subtitles"`
}

type SearchResult struct {
	Limit              int     `json:"limit"`
	Offset             int     `json:"offset"`
	TotalNumberOfItems int     `json:"totalNumberOfItems"`
	Items              []Track `json:"items"`
}

type Streaming struct {
	AlbumPeakAmplitude float64 `json:"albumPeakAmplitude"`
	AlbumReplayGain    float64 `json:"albumReplayGain"`
	AssetPresentation  string  `json:"assetPresentation"`
	AudioMode          string  `json:"audioMode"`
	AudioQuality       string  `json:"audioQuality"`
	Manifest           string  `json:"manifest"`
	ManifestMimeType   string  `json:"manifestMimeType"`
	TrackId            int64   `json:"trackId"`
	TrackPeakAmplitude float64 `json:"trackPeakAmplitude"`
	TrackReplayGain    float64 `json:"trackReplayGain"`
}

// BTS manifest structure (JSON)

type BTSManifest struct {
	MimeType       string   `json:"mimeType"`
	Codecs         string   `json:"codecs"`
	EncryptionType string   `json:"encryptionType"`
	URLs           []string `json:"urls"`
}

// DASH manifest structure (XML)

type DASHManifest struct {
	XMLName xml.Name `xml:"MPD"`
	Period  struct {
		AdaptationSet struct {
			Representation struct {
				SegmentTemplate struct {
					Initialization string `xml:"initialization,attr"`
					Media          string `xml:"media,attr"`
					Timeline       struct {
						Segments []struct {
							Duration int `xml:"d,attr"`
							Repeat   int `xml:"r,attr"`
						} `xml:"S"`
					} `xml:"SegmentTimeline"`
				} `xml:"SegmentTemplate"`
			} `xml:"Representation"`
		} `xml:"AdaptationSet"`
	} `xml:"Period"`
}

const (
	SearchTypeSong     = "s"
	SearchTypeAlbum    = "al"
	SearchTypeArtist   = "a"
	SearchTypeVideo    = "v"
	SearchTypePlaylist = "p"
)

const (
	QualityLossless = "LOSSLESS"
	QualityHigh     = "HIGH"
	QualityLow      = "LOW"
)

// Album items response structure
type AlbumItems struct {
	Limit              int         `json:"limit"`
	Offset             int         `json:"offset"`
	TotalNumberOfItems int         `json:"totalNumberOfItems"`
	Items              []AlbumItem `json:"items"`
}

type AlbumItem struct {
	Item Track  `json:"item"`
	Type string `json:"type"`
}

// Mix items response structure
type MixItems struct {
	Limit              int       `json:"limit"`
	Offset             int       `json:"offset"`
	TotalNumberOfItems int       `json:"totalNumberOfItems"`
	Items              []MixItem `json:"items"`
}

type MixItem struct {
	Item Track  `json:"item"`
	Type string `json:"type"`
}

// Search suggestions response structure (from /v2/client-suggestions)
type SuggestionsResponse struct {
	History        []interface{}       `json:"history"`
	Suggestions    []SuggestionQuery   `json:"suggestions"`
	DirectHits     []DirectHit         `json:"directHits"`
	SuggestionUUID string              `json:"suggestionUuid"`
}

type SuggestionQuery struct {
	Query      string      `json:"query"`
	Highlights []Highlight `json:"highlights"`
}

type Highlight struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

type DirectHit struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type"` // ARTISTS, TRACKS, ALBUMS, PLAYLISTS
}

type DirectHitTrack struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Album    struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Cover string `json:"cover"`
	} `json:"album"`
	Artists []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"artists"`
	Mixes map[string]string `json:"mixes"`
}

type DirectHitArtist struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type DirectHitAlbum struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
	Artists  []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"artists"`
}

type DirectHitPlaylist struct {
	UUID        string `json:"uuid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// SearchSuggestion is the unified suggestion type returned by GetSearchSuggestions
type SearchSuggestion struct {
	Text     string                  // The suggestion text (for query suggestions)
	Metadata SearchSuggestionMetadata // For direct hits
}

type SearchSuggestionMetadata struct {
	Type     string // "Track", "Artist", "Album", "Playlist"
	ID       string // The ID (track id, album id, playlist uuid, etc.)
	Title    string
	Subtitle string // Artist name for tracks, etc.
}