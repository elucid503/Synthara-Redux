package Tidal

import (
	"Synthara-Redux/Audio"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"path"
	"strings"
)

// StreamSource resolves the URL used for voice playback.
type StreamSource interface {

	ResolveStreamURL() (string, error)

}

// Song implements StreamSource for Tidal tracks and direct media links.

func (song *Song) ResolveStreamURL() (string, error) {

	if song == nil {

		return "", errors.New("nil song")

	}

	if song.Internal.DirectURL != "" {

		return song.Internal.DirectURL, nil

	}

	if song.TidalID == 0 {

		return "", errors.New("no stream source for song")

	}

	return GetStreamURL(song.TidalID)

}

func (song *Song) IsDirectMedia() bool {

	return song != nil && song.Internal.DirectURL != ""

}

// SongFromDirectURL adapts a direct .mp3/.mp4/.wav (or .m4a) link into a queue Song.
func SongFromDirectURL(mediaURL string) (*Song, error) {

	mediaURL = strings.TrimSpace(mediaURL)

	if mediaURL == "" {

		return nil, errors.New("empty media URL")

	}

	if _, err := url.ParseRequestURI(mediaURL); err != nil {

		return nil, fmt.Errorf("invalid media URL: %w", err)

	}

	duration := SongDuration{Formatted: "--:--"}

	if seconds := Audio.ProbeDurationSec(mediaURL); seconds > 0 {

		duration.Seconds = seconds
		duration.Formatted = FormatDuration(seconds)

	}

	return &Song{

		TidalID: directMediaID(mediaURL),

		Title: directMediaTitle(mediaURL),

		Artists: []string{directMediaDomain(mediaURL)},

		Duration: duration,

		Internal: SongInternal{

			DirectURL: mediaURL,

		},

	}, nil

}

func directMediaTitle(rawURL string) string {

	parsed, err := url.Parse(rawURL)

	if err != nil {

		return "unknown"

	}

	name := path.Base(parsed.Path)
	name = strings.TrimSuffix(name, path.Ext(name))

	if name == "" || name == "." || name == "/" {

		return directMediaDomain(rawURL)

	}

	return name

}

func directMediaDomain(rawURL string) string {

	parsed, err := url.Parse(rawURL)

	if err != nil {

		return "unknown"

	}

	host := parsed.Hostname()

	if host == "" {

		return "unknown"

	}

	return strings.TrimPrefix(strings.ToLower(host), "www.")

}

func directMediaID(rawURL string) int64 {

	hash := fnv.New64a()
	hash.Write([]byte(rawURL))

	return -int64(hash.Sum64() & 0x7fffffffffffffff) // Negative to avoid collisions with real track IDs.


}
