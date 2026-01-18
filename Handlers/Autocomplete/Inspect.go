package Autocomplete

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func InspectAutocomplete(Event *events.AutocompleteInteractionCreate) {

	Locale := Event.Locale().Code()

	Structs.GuildStoreMutex.Lock()
	
	Choices := []discord.AutocompleteChoice{}

	for _, Guild := range Structs.GuildStore {

		// Get guild name
		
		GuildName := Guild.ID.String()
		
		CachedGuild, ExistsInCache := Globals.DiscordClient.Caches.GuildCache().Get(Guild.ID)
		
		if ExistsInCache {
			
			GuildName = CachedGuild.Name
			
		}

		// Build display name with current state

		var StateLabel string
		
		switch Guild.Queue.State {
			
			case Structs.StatePlaying:
				
				StateLabel = "Playing"
				
			case Structs.StatePaused:
				
				StateLabel = "Paused"
				
			default:
				
				StateLabel = "Idle"
				
		}
		
		DisplayName := fmt.Sprintf("%s • %s", GuildName, StateLabel)

		// Truncate display name if necessary
		
		if len(DisplayName) > 100 {
			
			DisplayName = DisplayName[:97] + "..."
			
		}

		Choices = append(Choices, discord.AutocompleteChoiceString{
			
			Name:  DisplayName,
			Value: Guild.ID.String(),
			
		})

		// Discord has a limit of 25 autocomplete choices
		
		if len(Choices) >= 25 {
			
			break
			
		}

	}

	Structs.GuildStoreMutex.Unlock()

	if len(Choices) == 0 {

		Event.AutocompleteResult([]discord.AutocompleteChoice{
			
			discord.AutocompleteChoiceString{
				
				Name:  Localizations.Get("Autocomplete.Inspect.NoActiveGuilds", Locale),
				Value: "none",
				
			},
			
		})

		return

	}

	Event.AutocompleteResult(Choices)

}
