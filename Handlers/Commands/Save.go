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

func Save(Event *events.ApplicationCommandInteractionCreate) {

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

	Normalized, Error := Structs.NormalizeSavedQueueName(Name)

	if Error != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{buildSavedQueueNameError(Locale, Error)},
			Flags:  discord.MessageFlagEphemeral,

		})

		return

	}

	Snapshot, Error := Structs.SnapshotFromQueue(&Guild.Queue)

	if Error != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.EmptyQueue.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Save.Error.EmptyQueue.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Error = Structs.SaveGuildQueue(GuildID.String(), Normalized, Snapshot)

	if Error != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{buildSavedQueuePersistError(Locale, Error)},
			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	SongCount := Structs.SavedQueueSongCount(Snapshot)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title: Localizations.Get("Commands.Save.Success.Title", Locale),
			Author: Localizations.Get("Embeds.Categories.Playback", Locale),

			Description: Localizations.GetFormat(

				"Commands.Save.Success.Description",
				Locale,
				Normalized,
				SongCount,
				Localizations.Pluralize("Song", SongCount, Locale),

			),

			Color: Utils.PRIMARY,

		})},

	})

}

func buildSavedQueueNameError(Locale string, Error error) discord.Embed {

	switch {

		case errors.Is(Error, Structs.ErrSavedQueueNameEmpty):

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.InvalidName.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Save.Error.InvalidName.Description", Locale),
				Color: Utils.ERROR,

			})

		case errors.Is(Error, Structs.ErrSavedQueueNameTooLong):

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.NameTooLong.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Save.Error.NameTooLong.Description", Locale, Structs.MaxSavedQueueNameLen),
				Color: Utils.ERROR,

			})

		default:

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.Persist.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Save.Error.Persist.Description", Locale),
				Color: Utils.ERROR,

			})

	}

}

func buildSavedQueuePersistError(Locale string, Error error) discord.Embed {

	switch {

		case errors.Is(Error, Structs.ErrSavedQueueLimit):

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.LimitReached.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Save.Error.LimitReached.Description", Locale, Structs.MaxSavedQueuesPerGuild),
				Color: Utils.ERROR,

			})

		default:

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Save.Error.Persist.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Save.Error.Persist.Description", Locale),
				Color: Utils.ERROR,

			})

	}

}
