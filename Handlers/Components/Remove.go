package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func RemoveSong(Event *events.ComponentInteractionCreate, TidalID int64) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate guild session
	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	// Validate user is in voice
	if ErrorEmbed := Validation.VoiceStateError(GuildID, Event.User().ID, Locale); ErrorEmbed != nil {

		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{*ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	// Find the song in the upcoming queue by Tidal ID
	SongIndex := -1
	for Index, Song := range Guild.Queue.Upcoming {

		if Song.TidalID == TidalID {

			SongIndex = Index
			break

		}

	}

	if SongIndex == -1 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Song not found in queue.",
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Success := Guild.Queue.Remove(SongIndex)

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Failed to remove song from queue.",
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Embeds.Categories.Playback", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: "Song removed from queue.",

		})).
		Build())

}
