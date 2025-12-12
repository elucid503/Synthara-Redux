package Commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PingCommand(Event *events.ApplicationCommandInteractionCreate) {

	Event.CreateMessage(discord.MessageCreate{

		Content: "Pong!",

	})

}