package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func LastCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetOrCreateGuild(GuildID);

	Success := Guild.Queue.Last()

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Content: "There is no previously played song",

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: "Playing the previous song",

	})

}