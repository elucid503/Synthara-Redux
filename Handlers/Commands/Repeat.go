package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Repeat(Event *events.ApplicationCommandInteractionCreate) {

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
	Mode := Data.Int("mode")

	Guild.Features.Repeat = Mode

	var Title, Description string

	switch Mode {

		case Structs.RepeatOff:

			Title = Localizations.Get("Commands.Repeat.Off.Title", Locale)
			Description = Localizations.Get("Commands.Repeat.Off.Description", Locale)

		case Structs.RepeatOne:

			Title = Localizations.Get("Commands.Repeat.One.Title", Locale)
			Description = Localizations.Get("Commands.Repeat.One.Description", Locale)

		case Structs.RepeatAll:

			Title = Localizations.Get("Commands.Repeat.All.Title", Locale)
			Description = Localizations.Get("Commands.Repeat.All.Description", Locale)

		default:

			Title = Localizations.Get("Commands.Repeat.Off.Title", Locale)
			Description = Localizations.Get("Commands.Repeat.Off.Description", Locale)
			Guild.Features.Repeat = Structs.RepeatOff

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Title,
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Description,
			Color:       0xB3D9FF,

		})},

	})

}
