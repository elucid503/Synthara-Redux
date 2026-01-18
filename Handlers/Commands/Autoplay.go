package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Autoplay(Event *events.ApplicationCommandInteractionCreate) {

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

	Guild.Features.Autoplay = Enabled

	var Title, Description string

	if Enabled {

		Title = Localizations.Get("Commands.AutoPlay.Title", Locale)
		Description = Localizations.Get("Commands.AutoPlay.Enabled", Locale)

		// Generate initial suggestions when enabling autoplay

		if len(Guild.Queue.Suggestions) == 0 {
			Guild.Queue.RegenerateSuggestions()
		}

		// If queue is empty, start playback from suggestions

		if Guild.Queue.Current == nil && len(Guild.Queue.Upcoming) == 0 && len(Guild.Queue.Suggestions) > 0 {

			Guild.Queue.Next(true)

		}

	} else {

		Title = Localizations.Get("Commands.AutoPlay.Title", Locale)
		Description = Localizations.Get("Commands.AutoPlay.Disabled", Locale)

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