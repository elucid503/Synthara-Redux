package Components

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func AlbumEnqueue(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate that guild session exists

	if Guild == nil {
		
		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		
		return

	}
	
	Parts := strings.Split(Event.Data.CustomID(), ":")

	if len(Parts) < 2 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Components.Album.InvalidID.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Album.InvalidID.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	AlbumID := Parts[1]

	AlbumSongs, ErrorFetching := Innertube.GetAlbumSongs(AlbumID)

	if ErrorFetching != nil || len(AlbumSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Components.Album.FetchError.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Album.FetchError.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}
	
	for _, Song := range AlbumSongs {

		SongCopy := Song
		Guild.Queue.Add(&SongCopy, Event.User().ID.String())

	}
	
	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Components.Album.Enqueued.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Success", Locale),
			Description: Localizations.GetFormat("Components.Album.Enqueued.Description", Locale, len(AlbumSongs)),
			Color:       0xB3FFBA,

		})},
		
	})

}

func AlbumPlay(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate guild session exists

	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	Parts := strings.Split(Event.Data.CustomID(), ":")

	if len(Parts) < 2 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Components.Album.InvalidID.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Album.InvalidID.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	AlbumID := Parts[1]

	// Fetch album songs

	AlbumSongs, ErrorFetching := Innertube.GetAlbumSongs(AlbumID)

	if ErrorFetching != nil || len(AlbumSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Components.Album.FetchError.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Album.FetchError.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}

	Guild.Queue.Clear()

	for _, Song := range AlbumSongs {

		SongCopy := Song
		Guild.Queue.Add(&SongCopy, Event.User().ID.String())

	}

	if len(Guild.Queue.Upcoming) > 0 {

		Guild.Queue.Play()

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Components.Album.Playing.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Success", Locale),
			Description: Localizations.GetFormat("Components.Album.Playing.Description", Locale, len(AlbumSongs)),
			Color:       0xB3FFBA,

		})},
		
	})

}