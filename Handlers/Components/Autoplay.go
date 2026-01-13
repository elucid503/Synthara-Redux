package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Autoplay(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate guild session
	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}})
		return

	}

	// Validate user is in voice
	if ErrorEmbed := Validation.VoiceStateError(GuildID, Event.User().ID, Locale); ErrorEmbed != nil {

		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{*ErrorEmbed}})
		return

	}

	// Toggle autoplay
	Guild.Features.Autoplay = !Guild.Features.Autoplay

	var StatusKey string

	if Guild.Features.Autoplay {

		StatusKey = "Commands.AutoPlay.Enabled"

	} else {

		StatusKey = "Commands.AutoPlay.Disabled"

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.AutoPlay.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.Get(StatusKey, Locale),

		})).
		Build())

}
