package Commands

import (
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Next(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Next.Error.NoSession.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Next.Error.NoSession.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Success := Guild.Queue.Next()

	if !Success {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Next.Error.NoNextSong.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Next.Error.NoNextSong.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Create control buttons
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

			Title:       Localizations.Get("Commands.Next.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: fmt.Sprintf(Localizations.Get("Commands.Next.Description", Locale), Guild.Queue.Current.Title),

		})).
		AddActionRow(Buttons...).
		Build())

}