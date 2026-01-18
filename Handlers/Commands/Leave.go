package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Leave(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Leave.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Leave.Error.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild.Cleanup(true)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Leave.Success.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
			Description: Localizations.Get("Commands.Leave.Success.Description", Locale),
			Color:       Utils.PRIMARY,

		})},

	})

}
