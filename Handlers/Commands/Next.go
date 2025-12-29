package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func NextCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No active playback session found!",

		})

		return

	}

	Success := Guild.Queue.Next()

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Content: "There is no next song in the queue",

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: "Skipped to the next song",

	})

}