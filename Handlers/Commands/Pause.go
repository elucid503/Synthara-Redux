package Commands

import (
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Pause(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Pause.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Pause.Error.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePaused)

	// Create control buttons
	Buttons := []discord.InteractiveComponent{}

	LastButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Last", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipBack)),

	})

	Buttons = append(Buttons, LastButton)

	ResumeButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Buttons.Resume", Locale), "Play", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.Play)),

	})

	Buttons = append(Buttons, ResumeButton)

	NextButton := discord.NewButton(discord.ButtonStyleSecondary, "", "Next", "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.PlaySkipForward)),

	})

	Buttons = append(Buttons, NextButton)

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Pause.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.Get("Commands.Pause.Description", Locale),

		})).
		AddActionRow(Buttons...).
		Build())
	
}