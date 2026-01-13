package Commands

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"fmt"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Queue(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || (Guild.Queue.Current == nil && len(Guild.Queue.Upcoming) == 0) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Queue.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Queue.Error.Description", Locale),
				Color:       0xFFB3BA,

			})},
			
		})

		return

	}

	// Reset inactivity timer on activity
	Guild.ResetInactivityTimer()

	Page := fmt.Sprintf("%s/Queues/%s?View=Queue", os.Getenv("DOMAIN"), GuildID.String())

	var Body strings.Builder

	// Stats

	TotalSongs := len(Guild.Queue.Upcoming)

	// Calculates total milliseconds until finished (including current remaining if available)

	var TotalMs int64 = 0

	if Guild.Queue.Current != nil {

		CurrMs := int64(Guild.Queue.Current.Duration.Seconds * 1000)

		if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

			Prog := Guild.Queue.PlaybackSession.Streamer.Progress

			CurrMs -= Prog

			if CurrMs < 0 {

				CurrMs = 0 // for sanity

			}

		}

		TotalMs += CurrMs
	}

	for _, s := range Guild.Queue.Upcoming {

		TotalMs += int64(s.Duration.Seconds * 1000)

	}

	Minutes := int((TotalMs + 59999) / 60000) // rounds up to minutes

	// Counts unique contributors to the queue

	ContribSet := map[string]bool{}

	if Guild.Queue.Current != nil && Guild.Queue.Current.Internal.Requestor != "" {

		ContribSet[Guild.Queue.Current.Internal.Requestor] = true

	}

	for _, s := range Guild.Queue.Upcoming {

		if s.Internal.Requestor != "" {

			ContribSet[s.Internal.Requestor] = true

		}

	}

	Contributors := len(ContribSet)

	SongWord := Localizations.Pluralize("Song", TotalSongs, Locale)
	ContributorWord := Localizations.Pluralize("Contributor", Contributors, Locale)

	Stats := Localizations.GetFormat("Embeds.Queue.Stats", Locale, TotalSongs, SongWord, Minutes, Contributors, ContributorWord)
	
	Body.WriteString(fmt.Sprintf("%s\n\n", Stats))

	if len(Guild.Queue.Upcoming) > 0 {

		Max := 10

		if len(Guild.Queue.Upcoming) < Max {

			Max = len(Guild.Queue.Upcoming)

		}

		for i := 0; i < Max; i++ {

			SongItem := Guild.Queue.Upcoming[i]
			ArtistNames := strings.Join(SongItem.Artists, ", ")

			Body.WriteString(fmt.Sprintf("%d. **%s** • %s\n", i + 1, SongItem.Title, ArtistNames))
		
		} 

		if len(Guild.Queue.Upcoming) > Max {

			More := len(Guild.Queue.Upcoming) - Max
			Body.WriteString(fmt.Sprintf("%s\n", Localizations.GetFormat("Embeds.Queue.More", Locale, More)))

		} 

	} else {

		// No upcoming songs

		Body.WriteString(fmt.Sprintf("%s\n", Localizations.Get("Embeds.Queue.NoUpcoming", Locale)))

	}

	// Get full guild 

	FullGuild, Exists := Globals.DiscordClient.Caches.GuildCache().Get(GuildID)

	if (!Exists) {

		GuildFetchResp, GuildFetchErr := Globals.DiscordClient.Rest.GetGuild(GuildID, false)

		if GuildFetchErr == nil {

			FullGuild = GuildFetchResp.Guild

		}

	}

	// Build embed

	Embed := discord.NewEmbedBuilder()

	Embed.SetAuthor(Localizations.Get("Embeds.Queue.Title", Locale), "", "")
	Embed.SetTitle(FullGuild.Name)
	Embed.SetURL(Page)

	if Guild.Queue.Current != nil {

		SongColor, _ := Utils.GetDominantColorHex(Guild.Queue.Current.Cover)

		Embed.SetColor(SongColor)

	} else {

		Embed.SetColor(0xFFFFFF)

	}

	Embed.SetDescription(Body.String())

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Embed.Build()).
		AddActionRow(discord.NewButton(discord.ButtonStyleLink, Localizations.Get("Embeds.Queue.View", Locale), "", Page, snowflake.ID(0))).
		Build())

}