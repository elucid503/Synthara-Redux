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

func Move(Event *events.ApplicationCommandInteractionCreate) {

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

	SongIndex := Data.Int("song")
	NewPosition := Data.Int("position")

	// Validates index

	if SongIndex < 0 || SongIndex >= len(Guild.Queue.Upcoming) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Move.Error.InvalidSong.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Move.Error.InvalidSong.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Song := Guild.Queue.Upcoming[SongIndex]

	// Handles special case: position -1 means move to beginning

	var ToIndex int

	if NewPosition == -1 {

		ToIndex = 0

	} else if NewPosition < 0 || NewPosition >= len(Guild.Queue.Upcoming) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Move.Error.InvalidPosition.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Move.Error.InvalidPosition.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	} else {

		ToIndex = NewPosition

	}

	// Checks if moving to same position (or adjacent position which has no effect)

	if SongIndex == ToIndex || (SongIndex == ToIndex - 1 && NewPosition != -1) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Move.Error.SamePosition.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Move.Error.SamePosition.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Success := Guild.Queue.Move(SongIndex, ToIndex)

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Move.Error.Failed.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Move.Error.Failed.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Move.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Queue", Locale),
			Description: fmt.Sprintf(Localizations.Get("Commands.Move.Description", Locale), Song.Title, ToIndex + 1),
			Color:       Utils.PRIMARY,

		})},

	})

}
