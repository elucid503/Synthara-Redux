package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"

	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Conrols(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Details", os.Getenv("DOMAIN"), GuildID.String())

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Controls.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Controls", Locale),
			Description: Localizations.GetFormat("Commands.Controls.Description", Locale, Page),
			URL:         Page,

		})},

	})

}