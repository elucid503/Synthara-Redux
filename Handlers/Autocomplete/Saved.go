package Autocomplete

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"sort"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func SavedQueueAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	if GuildID == nil {

		Event.AutocompleteResult([]discord.AutocompleteChoice{

			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.SavedQueue.None", Locale),
				Value: "none",

			},

		})

		return

	}

	Names, Error := Structs.ListSavedQueueNames(GuildID.String())

	if Error != nil || len(Names) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{

			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.SavedQueue.None", Locale),
				Value: "none",

			},

		})

		return

	}

	sort.Strings(Names)

	Focused := strings.ToLower(Event.Data.String("name"))
	Choices := []discord.AutocompleteChoice{}

	for _, Name := range Names {

		if Focused != "" && !strings.Contains(strings.ToLower(Name), Focused) {

			continue

		}

		DisplayName := Name

		if len(DisplayName) > 100 {

			DisplayName = DisplayName[:97] + "..."

		}

		Choices = append(Choices, discord.AutocompleteChoiceString{

			Name:  DisplayName,
			Value: Name,

		})

		if len(Choices) >= 25 {

			break

		}

	}

	if len(Choices) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{

			discord.AutocompleteChoiceString{

				Name:  Localizations.Get("Autocomplete.SavedQueue.None", Locale),
				Value: "none",

			},

		})

		return

	}

	Event.AutocompleteResult(Choices)

}