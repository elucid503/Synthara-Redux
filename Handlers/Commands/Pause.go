package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PauseCommand(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Pause.Errors.NoSession", Locale),

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePaused)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: Localizations.Get("Commands.Pause.Success", Locale),
		
	})
	
}