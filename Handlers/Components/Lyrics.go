package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Lyrics(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Response, Err := Commands.BuildLyricsResponse(GuildID, Locale)

	if Err != nil {

		var ErrorTitle, ErrorDesc string

		if Err.Error() == "no song playing" {

			ErrorTitle = Localizations.Get("Commands.Lyrics.Error.NoSong.Title", Locale)
			ErrorDesc = Localizations.Get("Commands.Lyrics.Error.NoSong.Description", Locale)

		} else {

			ErrorTitle = Localizations.Get("Commands.Lyrics.Error.NotFound.Title", Locale)
			ErrorDesc = Localizations.Get("Commands.Lyrics.Error.NotFound.Description", Locale)

		}

		Event.CreateMessage(discord.MessageCreate{
		
			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       ErrorTitle,
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: ErrorDesc,
				
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Response.Embeds...).
		AddActionRow(Response.Buttons...).
		Build())

}