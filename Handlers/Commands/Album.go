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

func Album(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil || Guild.Queue.Current == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Album.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Album.Error.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	CurrentSong := Guild.Queue.Current

	if CurrentSong.AlbumID == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Album.NoAlbum.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Album.NoAlbum.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	// Fetch album songs from Tidal
	AlbumSongs, ErrorFetching := Tidal.FetchAlbumTracks(CurrentSong.AlbumID)

	if ErrorFetching != nil || len(AlbumSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Album.FetchError.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Album.FetchError.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	var Body strings.Builder

	// Stats

	TotalSongs := len(AlbumSongs)
	
	var TotalMs int64 = 0

	for _, s := range AlbumSongs {

		TotalMs += int64(s.Duration.Seconds * 1000)

	}

	Minutes := int((TotalMs + 59999) / 60000) // rounds up to minutes

	SongWord := Localizations.Pluralize("Song", TotalSongs, Locale)

	Stats := Localizations.GetFormat("Embeds.Album.Stats", Locale, TotalSongs, SongWord, Minutes)
	
	Body.WriteString(fmt.Sprintf("%s\n\n", Stats))

	// List songs (max 10)

	Max := 10

	if len(AlbumSongs) < Max {

		Max = len(AlbumSongs)

	}

	for i := 0; i < Max; i++ {

		SongItem := AlbumSongs[i]
		ArtistNames := strings.Join(SongItem.Artists, ", ")

		Body.WriteString(fmt.Sprintf("%d. **%s** • %s\n", i + 1, SongItem.Title, ArtistNames))
	
	} 

	if len(AlbumSongs) > Max {

		More := len(AlbumSongs) - Max
		Body.WriteString(fmt.Sprintf("%s\n", Localizations.GetFormat("Embeds.Album.More", Locale, More)))

	}

	// Build embed

	Embed := discord.NewEmbedBuilder()

	Embed.SetAuthor(Localizations.Get("Embeds.Album.Title", Locale), "", "")
	Embed.SetTitle(CurrentSong.Album)

	SongColor, _ := Utils.GetDominantColorHex(CurrentSong.Cover)
	Embed.SetColor(SongColor)

	Embed.SetDescription(Body.String())

	// Build buttons - use int64 AlbumID

	EnqueueButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Embeds.Album.EnqueueAll", Locale), fmt.Sprintf("AlbumEnqueue:%d", CurrentSong.AlbumID), "", 0).WithEmoji(discord.ComponentEmoji{
		ID: snowflake.MustParse(Icons.GetID(Icons.Albums)),
	})

	PlayButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Embeds.Album.PlayAll", Locale), fmt.Sprintf("AlbumPlay:%d", CurrentSong.AlbumID), "", 0).WithEmoji(discord.ComponentEmoji{
		ID: snowflake.MustParse(Icons.GetID(Icons.Play)),
	})

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Embed.Build()).
		AddActionRow(EnqueueButton, PlayButton).
		Build())

}
