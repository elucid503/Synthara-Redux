package Autocomplete

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()
	Input := Event.Data.String("query");

	if len(Input) < 3 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.Play.InputTooShort", Locale),
				Value: Localizations.Get("Autocomplete.Play.Placeholder", Locale),
			},

		})

		return;

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

	Event.AutocompleteResult(append(AutocompleteMetadataResults, AutocompleteTextResults...)); // Metadata results are prefered over text results

}