package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Resume(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Resume.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Resume.Error.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	Guild.Queue.SetState(Structs.StatePlaying)

	Event.CreateMessage(discord.MessageCreate{
		
		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Resume.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.Get("Commands.Resume.Description", Locale),

		})},
		
	})
	
}
