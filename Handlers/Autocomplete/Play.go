package Autocomplete

import (
	"Synthara-Redux/APIs/Innertube"
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

		} else if len(User.RecentSearches) > 0 {

			RecentlyPlayedLabel := Localizations.Get("Autocomplete.Play.RecentlyPlayed", Locale)

			for _, Search := range User.RecentSearches {

				RecentlyPlayedChoices = append(RecentlyPlayedChoices, discord.AutocompleteChoiceString{

					Name:  fmt.Sprintf("%s • %s", RecentlyPlayedLabel, Search.Title),
					Value: Search.URI,

				})

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

	Suggestions := Innertube.GetSearchSuggestions(Input);

	if len(Suggestions) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.Play.NoSuggestions", Locale),
				Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),

			},

		})

		return;

	}

	AutocompleteTextResults := []discord.AutocompleteChoice{};
	AutocompleteMetadataResults := []discord.AutocompleteChoice{};

	SeenText := make(map[string]bool)

	for _, Suggestion := range Suggestions {

		if (Suggestion.Metadata.Title != "" && Suggestion.Metadata.Subtitle != "") {

			AutocompleteMetadataResults = append(AutocompleteMetadataResults, discord.AutocompleteChoiceString{

				Name: fmt.Sprintf("%s • %s • %s", Suggestion.Metadata.Type, Suggestion.Metadata.Title, Suggestion.Metadata.Subtitle),
				Value: Utils.GetURI(Suggestion.Metadata.Type, Suggestion.Metadata.ID),

			})

			SeenText[strings.ToLower(Suggestion.Metadata.Title)] = true

		}

	}

	for _, Suggestion := range Suggestions {

		if (Suggestion.Metadata.Title != "" && Suggestion.Metadata.Subtitle != "") {

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

	// Combine results: Recently Played + Metadata + Text
	AllResults := append(RecentlyPlayedChoices, AutocompleteMetadataResults...)
	AllResults = append(AllResults, AutocompleteTextResults...)

	// Limit to 25 results (Discord limit)
	if len(AllResults) > 25 {
		AllResults = AllResults[:25]
	}

	Event.AutocompleteResult(AllResults)

}