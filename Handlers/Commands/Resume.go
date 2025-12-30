package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func ResumeCommand(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Resume.Errors.NoSession", Locale),

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePlaying)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: Localizations.Get("Commands.Resume.Success", Locale),
		
	})
	
}
