package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Last(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	Success := Guild.Queue.Last()

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Last.Errors.NoPreviousSong", Locale),

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.Get("Commands.Last.Success", Locale),

	})

}