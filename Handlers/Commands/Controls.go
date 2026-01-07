package Commands

import (
	"Synthara-Redux/Globals/Localizations"

	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Conrols(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Details", os.Getenv("DOMAIN"), GuildID.String())

	Embed := discord.NewEmbedBuilder()

	Embed.SetTitle(Localizations.Get("Embeds.Controls.Title", Locale))
	Embed.SetDescription(Localizations.GetFormat("Embeds.Controls.Description", Locale, Page))
	Embed.SetAuthor(Localizations.Get("Embeds.Notifications.Author", Locale), "", "")
	
	Embed.SetColor(0xFFFFFF)
	Embed.SetURL(Page)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Embed.Build()},

	})

}