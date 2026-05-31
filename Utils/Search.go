package Utils

import (
	"strings"
	"unicode"
)

// Rankable is satisfied by any search result that exposes a title and artist list.
type Rankable interface {

	GetTitle() string
	GetArtists() []string

}

// GetBestSearchResult returns the element from Results that best matches Query.
func GetBestSearchResult[T Rankable](Query string, Results []T) T {

	if len(Results) == 1 {

		return Results[0]

	}

	normalizedQuery := normalizeSearchText(Query)
	hintTitle, hintArtist := parseSearchQuery(normalizedQuery)

	bestIdx := 0
	bestScore := -1.0

	for i, Result := range Results {

		score := scoreSearchResult(hintTitle, hintArtist, Result.GetTitle(), Result.GetArtists())

		if score > bestScore {

			bestScore = score
			bestIdx = i

		}

	}

	return Results[bestIdx]
}

// parseSearchQuery splits "some title by some artist" into (title, artist).
func parseSearchQuery(query string) (hintTitle, hintArtist string) {

	idx := strings.LastIndex(query, " by ")

	if idx == -1 {

		return query, ""

	}

	title := strings.TrimSpace(query[:idx])
	artist := strings.TrimSpace(query[idx+4:])

	if title != "" && artist != "" {

		return title, artist

	}

	return query, ""

}

func scoreSearchResult(hintTitle, hintArtist, resultTitle string, resultArtists []string) float64 {

	normalizedTitle := normalizeSearchText(resultTitle)

	score := searchTextSimilarity(hintTitle, normalizedTitle) * 60

	if hintArtist != "" {

		artistScore := 0.0

		for _, artist := range resultArtists {

			s := searchTextSimilarity(hintArtist, normalizeSearchText(artist))

			if s > artistScore {

				artistScore = s

			}

		}

		score += artistScore * 40

	}

	return score

}

func searchTextSimilarity(a, b string) float64 {

	if a == "" || b == "" {

		return 0.0

	}

	if a == b {

		return 1.0

	}

	if strings.Contains(b, a) || strings.Contains(a, b) {

		return 0.9

	}

	return searchWordOverlap(a, b)

}

func searchWordOverlap(a, b string) float64 {

	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {

		return 0.0

	}

	setB := make(map[string]bool, len(wordsB))

	for _, w := range wordsB {

		setB[w] = true

	}

	matches := 0

	for _, w := range wordsA {

		if setB[w] {

			matches++

		}

	}

	union := len(wordsA) + len(wordsB) - matches

	if union == 0 {

		return 0.0

	}

	return float64(matches) / float64(union)

}

// normalizeSearchText lowercases, expands common abbreviations, strips punctuation, and normalizes whitespace.
func normalizeSearchText(s string) string {

	s = strings.ToLower(s)

	s = strings.ReplaceAll(s, "&", " and ")
	s = strings.ReplaceAll(s, "'n'", " and ")

	var buf strings.Builder

	for _, r := range s {

		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {

			buf.WriteRune(r)

		} else {

			buf.WriteRune(' ')

		}

	}

	words := strings.Fields(buf.String())

	for i, w := range words {

		switch w {

		case "thru":

			words[i] = "through"

		case "n":

			words[i] = "and"

		}

	}

	return strings.Join(words, " ")

}
