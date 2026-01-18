package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Queue(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Response, Err := Commands.BuildQueueResponse(GuildID, Locale)

	if Err != nil {

		Event.CreateMessage(discord.MessageCreate{
		
			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Queue.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Queue.Error.Description", Locale),
				Color:       Utils.RED,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Response.Embeds...).
		AddActionRow(Response.Buttons...).
		Build())

}
