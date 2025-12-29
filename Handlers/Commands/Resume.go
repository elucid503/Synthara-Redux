package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func ResumeCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No active playback session found!",

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePlaying)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: "Resumed the currently paused song",
		
	})
	
}
