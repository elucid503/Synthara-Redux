package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func NextCommand(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Next.Errors.NoSession", Locale),

		})

		return

	}

	Success := Guild.Queue.Next()

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Next.Errors.NoNextSong", Locale),

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.Get("Commands.Next.Success", Locale),

	})

}