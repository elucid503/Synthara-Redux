package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"fmt"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Queue(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || (Guild.Queue.Current == nil && len(Guild.Queue.Upcoming) == 0) {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Queue.Errors.NoQueue", Locale),
			
		})

		return

	}

	Page := fmt.Sprintf("%s/Queues/%s?View=Queue", os.Getenv("DOMAIN"), GuildID.String())

	var Body strings.Builder

	if len(Guild.Queue.Upcoming) > 0 {

		Max := 10

		if len(Guild.Queue.Upcoming) < Max {

			Max = len(Guild.Queue.Upcoming)

		}

		for i := 0; i < Max; i++ {

			SongItem := Guild.Queue.Upcoming[i]
			ArtistNames := strings.Join(SongItem.Artists, ", ")

			Body.WriteString(fmt.Sprintf("> **%s**\n> %s\n\n", SongItem.Title, ArtistNames))
		
		} 

		if len(Guild.Queue.Upcoming) > Max {

			More := len(Guild.Queue.Upcoming) - Max
			Body.WriteString(fmt.Sprintf("%s\n", Localizations.GetFormat("Embeds.Queue.More", Locale, More)))

		} 

	} else {

		// No upcoming songs

		Body.WriteString(fmt.Sprintf("%s\n", Localizations.Get("Embeds.Queue.NoUpcoming", Locale)))

	}

	// Append view link

	Body.WriteString(fmt.Sprintf("%s", Localizations.GetFormat("Embeds.Queue.View", Locale, Page)))

	// Build embed

	Embed := discord.NewEmbedBuilder()

	Embed.SetTitle(Localizations.Get("Embeds.Queue.Title", Locale))
	Embed.SetURL(Page)

	if Guild.Queue.Current != nil {

		SongColor, _ := Utils.GetDominantColorHex(Guild.Queue.Current.Cover)

		Embed.SetColor(SongColor)

	} else {

		Embed.SetColor(0xFFFFFF)

	}

	Embed.SetDescription(Body.String())

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Embed.Build()},

	})

}