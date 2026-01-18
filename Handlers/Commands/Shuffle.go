package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Shuffle(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	if ErrorEmbed := Validation.VoiceStateError(GuildID, Event.User().ID, Locale); ErrorEmbed != nil {

		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{*ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	Data := Event.SlashCommandInteractionData()
	Enabled := Data.Bool("enabled")

	Guild.Features.Shuffle = Enabled

	var Title, Description string

	if Enabled {

		Title = Localizations.Get("Commands.Shuffle.Enabled.Title", Locale)
		Description = Localizations.Get("Commands.Shuffle.Enabled.Description", Locale)

	} else {

		Title = Localizations.Get("Commands.Shuffle.Disabled.Title", Locale)
		Description = Localizations.Get("Commands.Shuffle.Disabled.Description", Locale)

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Title,
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Description,
			Color:       Utils.WHITE,

		})},

	})

}
