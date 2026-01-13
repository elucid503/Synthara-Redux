package Components

import (
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Validation"
	"unsafe"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Queue(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate guild session exists
	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}})
		return

	}

	// Delegates to command using unsafe pointer conversion
	Commands.Queue(*(**events.ApplicationCommandInteractionCreate)(unsafe.Pointer(&Event)))

}
