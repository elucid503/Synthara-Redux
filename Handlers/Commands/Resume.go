package Commands

import (
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func ResumeCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID);

	Guild.Queue.SetState(Structs.StatePlaying)

	Event.CreateMessage(discord.MessageCreate{
		
		Content: "Resumed the currently paused song",
		
	})
	
}
