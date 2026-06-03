package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"
	"errors"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Load(Event *events.ApplicationCommandInteractionCreate) {

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
	Name := Data.String("name")

	Snapshot, Error := Structs.GetSavedQueue(GuildID.String(), Name)

	if Error != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{buildSavedQueueLoadError(Locale, Error)},
			Flags:  discord.MessageFlagEphemeral,

		})

		return

	}

	Guild.ApplySavedQueue(*Snapshot)

	Normalized, _ := Structs.NormalizeSavedQueueName(Name)
	SongCount := Structs.SavedQueueSongCount(*Snapshot)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title: Localizations.Get("Commands.Load.Success.Title", Locale),
			Author: Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.GetFormat(

				"Commands.Load.Success.Description",
				Locale,
				Normalized,
				SongCount,
				Localizations.Pluralize("Song", SongCount, Locale),

			),
			Color: Utils.PRIMARY,

		})},

	})

}

func buildSavedQueueLoadError(Locale string, Error error) discord.Embed {

	switch {

		case errors.Is(Error, Structs.ErrSavedQueueNotFound):

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Load.Error.NotFound.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Load.Error.NotFound.Description", Locale),
				Color:       Utils.ERROR,

			})

		case errors.Is(Error, Structs.ErrSavedQueueNameEmpty), errors.Is(Error, Structs.ErrSavedQueueNameTooLong):

			return buildSavedQueueNameError(Locale, Error)

		default:

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Load.Error.Persist.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Load.Error.Persist.Description", Locale),
				Color:       Utils.ERROR,

			})

	}

}