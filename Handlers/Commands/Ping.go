package Commands

import (
	"Synthara-Redux/Globals/Localizations"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PingCommand(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.Get("Commands.Ping.Success", Locale),

	})

}