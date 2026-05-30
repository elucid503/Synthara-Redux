package Commands

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Artist(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || Guild.Queue.Current == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Artist.Error.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Artist.Error.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	CurrentSong := Guild.Queue.Current

	if CurrentSong.ArtistID == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Artist.NoArtist.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Artist.NoArtist.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Fetch artist top tracks from Tidal
	ArtistSongs, ErrorFetching := Tidal.FetchArtistTopTracks(CurrentSong.ArtistID)

	if ErrorFetching != nil || len(ArtistSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Commands.Artist.FetchError.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Artist.FetchError.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	var Body strings.Builder

	// Stats

	TotalSongs := len(ArtistSongs)

	var TotalMs int64 = 0

	for _, s := range ArtistSongs {

		TotalMs += int64(s.Duration.Seconds * 1000)

	}

	Minutes := int((TotalMs + 59999) / 60000) // rounds up to minutes

	SongWord := Localizations.Pluralize("Song", TotalSongs, Locale)

	Stats := Localizations.GetFormat("Embeds.Artist.Stats", Locale, TotalSongs, SongWord, Minutes)

	Body.WriteString(fmt.Sprintf("%s\n\n", Stats))

	// List songs (max 10)

	Max := 10

	if len(ArtistSongs) < Max {

		Max = len(ArtistSongs)

	}

	for i := 0; i < Max; i++ {

		SongItem := ArtistSongs[i]

		Body.WriteString(fmt.Sprintf("%d. **%s** • %s\n", i+1, SongItem.Title, SongItem.Album))

	}

	if len(ArtistSongs) > Max {

		More := len(ArtistSongs) - Max
		Body.WriteString(fmt.Sprintf("%s\n", Localizations.GetFormat("Embeds.Artist.More", Locale, More)))

	}

	// Build embed

	ArtistName := CurrentSong.Artists[0]

	Embed := discord.NewEmbedBuilder()

	Embed.SetAuthor(Localizations.Get("Embeds.Artist.Title", Locale), "", "")
	Embed.SetTitle(ArtistName)

	SongColor, _ := Utils.GetDominantColorHex(CurrentSong.Cover)
	Embed.SetColor(SongColor)

	Embed.SetDescription(Body.String())

	// Build buttons

	EnqueueButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Embeds.Artist.EnqueueAll", Locale), fmt.Sprintf("ArtistEnqueue:%d", CurrentSong.ArtistID), "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.Sparkles)),

	})

	PlayButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Embeds.Artist.PlayAll", Locale), fmt.Sprintf("ArtistPlay:%d", CurrentSong.ArtistID), "", 0).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.Play)),

	})

	Event.CreateMessage(discord.NewMessageCreate().
		AddEmbeds(Embed).
		AddActionRow(EnqueueButton, PlayButton))

}
