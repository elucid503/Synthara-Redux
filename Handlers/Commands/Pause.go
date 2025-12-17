package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PauseCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetOrCreateGuild(GuildID);

	Guild.Queue.ChangeState(Structs.StatePaused)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: "Paused the currently playing song",
		
	})
	
}