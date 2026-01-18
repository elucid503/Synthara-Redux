package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Jump(Event *events.ApplicationCommandInteractionCreate) {

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
	Position := Data.Int("position")

	if Position < 0 || Position >= len(Guild.Queue.Upcoming) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Jump.Error.InvalidPosition.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Jump.Error.InvalidPosition.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Success := Guild.Queue.Jump(Position + 1)

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Jump.Error.InvalidPosition.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Jump.Error.InvalidPosition.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Jump.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: fmt.Sprintf(Localizations.Get("Commands.Jump.Description", Locale), Guild.Queue.Current.Title),
			Color:       Utils.PRIMARY,

		})},

	})

}
