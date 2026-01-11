package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Ping(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Ping.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
			Description: Localizations.Get("Commands.Ping.Description", Locale),

		})},

	})

}