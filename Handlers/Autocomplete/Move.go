package Autocomplete

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func MoveAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	if GuildID == nil {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceInt{

				Name:  Localizations.Get("Autocomplete.Move.NoSongs", Locale),
				Value: 0,

			},

		})

		return

	}

	Guild := Structs.GetGuild(*GuildID, false)

	if Guild == nil || len(Guild.Queue.Upcoming) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceInt{

				Name:  Localizations.Get("Autocomplete.Move.NoSongs", Locale),
				Value: 0,

			},

		})

		return

	}

	Data := Event.Data
	FocusedOption := Data.Focused().Name

	Choices := []discord.AutocompleteChoice{}

	MaxChoices := 25
	
	if len(Guild.Queue.Upcoming) < MaxChoices {
		MaxChoices = len(Guild.Queue.Upcoming)
	}

	// For "song" option, shows all songs in queue

	switch FocusedOption {

		case "song":

			for Index := 0; Index < MaxChoices; Index++ {

				Song := Guild.Queue.Upcoming[Index]
				PositionLabel := Localizations.Get("Autocomplete.Move.Position", Locale)

				Choices = append(Choices, discord.AutocompleteChoiceInt{

					Name:  fmt.Sprintf("%s %d • %s", PositionLabel, Index+1, Song.Title),
					Value: Index,

				})

			}

		case "position":

			// For "position" option, show queue positions (insert after)
			// Include position 0 which means "move to beginning"
			
			PositionLabel := Localizations.Get("Autocomplete.Move.Position", Locale)
			BeforeLabel := Localizations.Get("Autocomplete.Move.BeforeAll", Locale)

			Choices = append(Choices, discord.AutocompleteChoiceInt{

				Name:  fmt.Sprintf("%s %d • %s", PositionLabel, 0, BeforeLabel),
				Value: -1, // Special value for "before everything"

			})

			for Index := 0; Index < MaxChoices; Index++ {

				Song := Guild.Queue.Upcoming[Index]

				Choices = append(Choices, discord.AutocompleteChoiceInt{

					Name:  fmt.Sprintf("%s %d • %s", PositionLabel, Index+1, Song.Title),
					Value: Index,

				})

			}

	}

	Event.AutocompleteResult(Choices)

}
