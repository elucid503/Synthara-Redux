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

func Manage(Event *events.ApplicationCommandInteractionCreate) {

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
	Action := Data.String("action")

	Normalized, Error := Structs.NormalizeSavedQueueName(Name)

	if Error != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{buildSavedQueueNameError(Locale, Error)},
			Flags:  discord.MessageFlagEphemeral,

		})

		return

	}

	switch Action {

		case "delete":

			Error = Structs.DeleteSavedQueue(GuildID.String(), Normalized)

			if Error != nil {

				Event.CreateMessage(discord.MessageCreate{

					Embeds: []discord.Embed{buildSavedQueueManageError(Locale, Error)},
					Flags:  discord.MessageFlagEphemeral,

				})

				return

			}

			Event.CreateMessage(discord.MessageCreate{

				Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Commands.Manage.Delete.Success.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
					Description: Localizations.GetFormat("Commands.Manage.Delete.Success.Description", Locale, Normalized),
					Color:       Utils.PRIMARY,

				})},

			})

		case "overwrite":

			Snapshot, Error := Structs.SnapshotFromQueue(&Guild.Queue)

			if Error != nil {

				Event.CreateMessage(discord.MessageCreate{

					Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Commands.Manage.Overwrite.Error.EmptyQueue.Title", Locale),
						Author:      Localizations.Get("Embeds.Categories.Error", Locale),
						Description: Localizations.Get("Commands.Manage.Overwrite.Error.EmptyQueue.Description", Locale),
						Color:       Utils.ERROR,

					})},

					Flags: discord.MessageFlagEphemeral,

				})

				return

			}

			_, Error = Structs.GetSavedQueue(GuildID.String(), Normalized)

			if Error != nil {

				Event.CreateMessage(discord.MessageCreate{

					Embeds: []discord.Embed{buildSavedQueueManageError(Locale, Error)},
					Flags:  discord.MessageFlagEphemeral,

				})

				return

			}

			Error = Structs.SaveGuildQueue(GuildID.String(), Normalized, Snapshot)

			if Error != nil {

				Event.CreateMessage(discord.MessageCreate{

					Embeds: []discord.Embed{buildSavedQueuePersistError(Locale, Error)},
					Flags:  discord.MessageFlagEphemeral,

				})

				return

			}

			SongCount := Structs.SavedQueueSongCount(Snapshot)

			Event.CreateMessage(discord.MessageCreate{

				Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

					Title: Localizations.Get("Commands.Manage.Overwrite.Success.Title", Locale),
					Author: Localizations.Get("Embeds.Categories.Playback", Locale),
					Description: Localizations.GetFormat(

						"Commands.Manage.Overwrite.Success.Description",
						Locale,
						Normalized,
						SongCount,
						Localizations.Pluralize("Song", SongCount, Locale),

					),
					Color: Utils.PRIMARY,

				})},

			})

		default:

			Event.CreateMessage(discord.MessageCreate{

				Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Commands.Manage.Error.UnknownAction.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Error", Locale),
					Description: Localizations.Get("Commands.Manage.Error.UnknownAction.Description", Locale),
					Color:       Utils.ERROR,

				})},

				Flags: discord.MessageFlagEphemeral,

			})

	}

}

func buildSavedQueueManageError(Locale string, Error error) discord.Embed {

	switch {

		case errors.Is(Error, Structs.ErrSavedQueueNotFound):

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Manage.Error.NotFound.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Manage.Error.NotFound.Description", Locale),
				Color:       Utils.ERROR,

			})

		default:

			return Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Manage.Error.Persist.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Manage.Error.Persist.Description", Locale),
				Color:       Utils.ERROR,

			})

	}

}