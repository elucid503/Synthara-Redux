package Components

import (
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Validation"
	"unsafe"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Lyrics(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate playback exists
	if Guild == nil || Guild.Queue.Current == nil {

		ErrorEmbed := Validation.PlaybackError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	// Delegates to command using unsafe pointer conversion
	Commands.Lyrics(*(**events.ApplicationCommandInteractionCreate)(unsafe.Pointer(&Event)))

}