package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

const AboutCategoryVoice = "voice"

func About(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	Data := Event.SlashCommandInteractionData()

	Category := Data.String("category")

	var ContentKey string

	switch Category {

	case AboutCategoryVoice:

		ContentKey = "Commands.About.Voice.Content"

	default:

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.About.Error.UnknownCategory.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.About.Error.UnknownCategory.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.Get(ContentKey, Locale),
		Flags: discord.MessageFlagEphemeral,

	})

}
