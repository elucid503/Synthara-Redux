package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PauseCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No active playback session found!",

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePaused)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: "Paused the currently playing song",
		
	})
	
}