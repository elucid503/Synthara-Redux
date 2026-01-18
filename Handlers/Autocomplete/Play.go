package Autocomplete

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()
	Input := Event.Data.String("query");

	// Get user's recent searches from MongoDB

	RecentlyPlayedChoices := []discord.AutocompleteChoice{}

	User, UserError := Structs.GetUser(Event.User().ID.String())

	// User Found, Processing Recent Searches

	if UserError == nil {

		if User.FirstUse {

			if len(Input) < 3 {

				Event.AutocompleteResult([]discord.AutocompleteChoice{

					discord.AutocompleteChoiceString{

						Name:  Localizations.Get("Autocomplete.Play.Welcome", Locale),
						Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),
						
					},
				})

				return

			}

		} else {

			if len(User.RecentSearches) > 0 {

				RecentlyPlayedLabel := Localizations.Get("Autocomplete.Play.RecentlyPlayed", Locale)

				for _, Search := range User.RecentSearches {

					RecentlyPlayedChoices = append(RecentlyPlayedChoices, discord.AutocompleteChoiceString{

						Name:  fmt.Sprintf("%s • %s", RecentlyPlayedLabel, Search.Title),
						Value: Search.URI,

					})

				}

			}

			// Add Play Favorites

			if len(User.Favorites) > 0 {

				RecentlyPlayedChoices = append([]discord.AutocompleteChoice{discord.AutocompleteChoiceString{

					Name:  Localizations.Get("Autocomplete.Play.Favorites", Locale),
					Value: "Synthara-Redux:Favorites:" + User.DiscordID,
					
				}}, RecentlyPlayedChoices...)

			}

			// Add Play Suggestions

			if User.MostRecentMix != "" {

				RecentlyPlayedChoices = append([]discord.AutocompleteChoice{discord.AutocompleteChoiceString{

					Name: Localizations.Get("Autocomplete.Play.Suggestions", Locale),
					Value: "Synthara-Redux:Suggestions:" + User.DiscordID,

				}}, RecentlyPlayedChoices...)

			}

		}

	}

	// Input Length Check

	if len(Input) < 3 {

		if len(RecentlyPlayedChoices) > 0 {

			Event.AutocompleteResult(RecentlyPlayedChoices)

		} else {

			Event.AutocompleteResult([]discord.AutocompleteChoice{
				
				discord.AutocompleteChoiceString{

					Name:  Localizations.Get("Autocomplete.Play.NoRecentSearches", Locale),
					Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),
				},

			})

		}

		return;

	}

	// URL

	if strings.HasPrefix(Input, "http://") || strings.HasPrefix(Input, "https://") {

		DirectURLLabel := Localizations.Get("Autocomplete.Play.DirectURL", Locale)

		Event.AutocompleteResult([]discord.AutocompleteChoice{

			discord.AutocompleteChoiceString{

				Name:  fmt.Sprintf("%s • %s", DirectURLLabel, Input),
				Value: Input,

			},

		})

		return

	}

	Suggestions := Tidal.GetSearchSuggestions(Input);

	if len(Suggestions) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.Play.NoSuggestions", Locale),
				Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),

			},

		})

		return;

	}

	// Group metadata suggestions by type so we can order and cap them

	AutocompleteTextResults := []discord.AutocompleteChoice{}

	// per-type buckets

	Songs := []discord.AutocompleteChoice{}
	Albums := []discord.AutocompleteChoice{}
	Artists := []discord.AutocompleteChoice{}
	Others := []discord.AutocompleteChoice{}

	SeenText := make(map[string]bool)

	for _, Suggestion := range Suggestions {

		if Suggestion.Metadata.Title != "" && Suggestion.Metadata.Subtitle != "" {

			Entry := discord.AutocompleteChoiceString{

				Name:  fmt.Sprintf("%s • %s • %s", Suggestion.Metadata.Type, Suggestion.Metadata.Title, Suggestion.Metadata.Subtitle),
				Value: Utils.GetURI(Suggestion.Metadata.Type, Suggestion.Metadata.ID),
			}

			TypeLower := strings.ToLower(Suggestion.Metadata.Type)

			switch {

				case strings.Contains(TypeLower, "song") || strings.Contains(TypeLower, "track"):

					Songs = append(Songs, Entry)

				case strings.Contains(TypeLower, "album"):

					Albums = append(Albums, Entry)

				case strings.Contains(TypeLower, "artist"):

					Artists = append(Artists, Entry)

				default:

					Others = append(Others, Entry)
					
			}

			// avoid showing duplicate plain-text entries later

			SeenText[strings.ToLower(Suggestion.Metadata.Title)] = true

		}

	}

	// collects plain-text suggestions (deduplicated)

	for _, Suggestion := range Suggestions {

		if Suggestion.Metadata.Title != "" && Suggestion.Metadata.Subtitle != "" {

			continue

		}

		LowerText := strings.ToLower(Suggestion.Text)

		if SeenText[LowerText] {

			continue
			
		}

		AutocompleteTextResults = append(AutocompleteTextResults, discord.AutocompleteChoiceString{

			Name:  Suggestion.Text,
			Value: Suggestion.Text,

		})

		SeenText[LowerText] = true

	}

	// Limits (tuned: more songs than other types)

	const (

		SongLimit   = 5
		AlbumLimit  = 3
		ArtistLimit = 3
		OtherLimit  = 2
		TextLimit   = 5
		MaxTotal    = 25

	)

	// Songs -> Albums -> Artists -> Other -> Plain text

	AllResults := []discord.AutocompleteChoice{}

	AppendUpTo := func(Choices []discord.AutocompleteChoice, Source []discord.AutocompleteChoice, n int) []discord.AutocompleteChoice {
		
		if n <= 0 || len(Source) == 0 {

			return Choices

		}

		if len(Source) > n {

			Choices = append(Choices, Source[:n]...)

		} else {

			Choices = append(Choices, Source...)

		}

		return Choices

	}

	AllResults = AppendUpTo(AllResults, Songs, SongLimit)
	AllResults = AppendUpTo(AllResults, Albums, AlbumLimit)
	AllResults = AppendUpTo(AllResults, Artists, ArtistLimit)
	AllResults = AppendUpTo(AllResults, Others, OtherLimit)
	AllResults = AppendUpTo(AllResults, AutocompleteTextResults, TextLimit)

	// Fallback

	if len(AllResults) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{

			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.Play.NoSuggestions", Locale),
				Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),
			},

		})

		return

	}

	if len(AllResults) > MaxTotal {

		AllResults = AllResults[:MaxTotal]
		
	}

	Event.AutocompleteResult(AllResults)

}