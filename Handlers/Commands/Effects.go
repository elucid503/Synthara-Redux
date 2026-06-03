package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// Both the Speed and Reverb commands are very similar, so they use the same helper function to reduce code duplication.

func Speed(event *events.ApplicationCommandInteractionCreate) {

	runPlaybackIntSetting(event, "Commands.Speed.Title",

		func(guild *Structs.Guild, locale string) string {

			return Localizations.GetFormat("Commands.Speed.Description", locale, Structs.FormatSpeedLabel(guild.Features.SpeedMilli))

		},

		(*Structs.Guild).SetSpeed,

	)

}

func Reverb(event *events.ApplicationCommandInteractionCreate) {

	runPlaybackIntSetting(event, "Commands.Reverb.Title",

		func(guild *Structs.Guild, locale string) string {

			return Localizations.GetFormat("Commands.Reverb.Description", locale, guild.Features.Reverb)

		},

		(*Structs.Guild).SetReverb,

	)

}

func runPlaybackIntSetting(event *events.ApplicationCommandInteractionCreate, titleKey string, describe func(*Structs.Guild, string) string, apply func(*Structs.Guild, int) int) {

	locale := event.Locale().Code()
	guildID := *event.GuildID()

	guild := Structs.GetGuild(guildID, false)

	if guild == nil {

		event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Validation.GuildSessionError(locale)},
			Flags:  discord.MessageFlagEphemeral,

		})

		return

	}

	if voiceErr := Validation.VoiceStateError(guildID, event.User().ID, locale); voiceErr != nil {

		event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{*voiceErr},
			Flags:  discord.MessageFlagEphemeral,

		})

		return

	}

	value := int(event.SlashCommandInteractionData().Int("value"))
	guild.ResetInactivityTimer()

	apply(guild, value)

	event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title: Localizations.Get(titleKey, locale),
			Author: Localizations.Get("Embeds.Categories.Playback", locale),
			Description: describe(guild, locale),
			Color: Utils.PRIMARY,

		})},

	})

}
