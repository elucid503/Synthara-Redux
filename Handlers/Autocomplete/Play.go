package Autocomplete

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Event.AutocompleteResult([]discord.AutocompleteChoice{ 

		discord.AutocompleteChoiceString{

			Name: "Test",
			Value: "Test",

		},

	})
	
}