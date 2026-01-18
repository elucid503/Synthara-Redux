package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Forget(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	UserData, ErrorGettingUser := Structs.GetUser(Event.User().ID.String())

	if ErrorGettingUser != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Forget.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Forget.Error.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	ErrorClearing := UserData.ClearRecentSearches()

	if ErrorClearing != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Forget.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Forget.Error.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Forget.Success.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
			Description: Localizations.Get("Commands.Forget.Success.Description", Locale),

		})},

	})

}
