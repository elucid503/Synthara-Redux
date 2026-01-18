package Components

import (
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func JumpToSong(Event *events.ComponentInteractionCreate, TidalID int64) {

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

			SongIndex = Index + 1 // Jump expects 1-indexed position
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

	Success := Guild.Queue.Jump(SongIndex)

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Failed to jump to song.",
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Buttons := []discord.InteractiveComponent{}

	LastButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Last", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipBack)),

	})

	Buttons = append(Buttons, LastButton)

	PauseButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Pause", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.Pause)),

	})

	Buttons = append(Buttons, PauseButton)

	NextButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Next", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipForward)),

	})

	Buttons = append(Buttons, NextButton)

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Embeds.Categories.JumpedToSong", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.GetFormat("Commands.Jump.Description", Locale, Guild.Queue.Current.Title),

		})).
		AddActionRow(Buttons...).
		Build())

}
