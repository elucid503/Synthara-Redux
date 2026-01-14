package Autocomplete

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func ReplayAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	if GuildID == nil {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceInt{

				Name:  Localizations.Get("Autocomplete.Replay.NoSongs", Locale),
				Value: 0,

			},

		})

		return

	}

	Guild := Structs.GetGuild(*GuildID, false)

	if Guild == nil || len(Guild.Queue.Previous) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceInt{

				Name:  Localizations.Get("Autocomplete.Replay.NoSongs", Locale),
				Value: 0,

			},

		})

		return

	}

	Choices := []discord.AutocompleteChoice{}

	MaxChoices := 25
	if len(Guild.Queue.Previous) < MaxChoices {
		MaxChoices = len(Guild.Queue.Previous)
	}

	for Index := 0; Index < MaxChoices; Index++ {

		Song := Guild.Queue.Previous[len(Guild.Queue.Previous)-1-Index]
		PositionLabel := Localizations.Get("Autocomplete.Jump.Position", Locale)

		Choices = append(Choices, discord.AutocompleteChoiceInt{

			Name:  fmt.Sprintf("%s %d • %s", PositionLabel, Index+1, Song.Title),
			Value: Index,

		})

	}

	Event.AutocompleteResult(Choices)

}
