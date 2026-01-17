package Components

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func AlbumEnqueue(Event *events.ComponentInteractionCreate) {

	// Defer response since fetching album tracks may take time
	Event.DeferCreateMessage(false)

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

	AlbumIDStr := Parts[1]
	AlbumID, ParseErr := strconv.ParseInt(AlbumIDStr, 10, 64)

	if ParseErr != nil {

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

	AlbumSongs, ErrorFetching := Tidal.FetchAlbumTracks(AlbumID)

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
		
	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Components.Album.Enqueued.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Success", Locale),
			Description: Localizations.GetFormat("Components.Album.Enqueued.Description", Locale, len(AlbumSongs)),
			Color:       0xB3FFBA,

		})).Build())

}

func AlbumPlay(Event *events.ComponentInteractionCreate) {

	// Defer response since fetching album tracks may take time
	Event.DeferCreateMessage(false)

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

	AlbumIDStr := Parts[1]
	AlbumID, ParseErr := strconv.ParseInt(AlbumIDStr, 10, 64)

	if ParseErr != nil {

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

	// Fetch album songs from Tidal

	AlbumSongs, ErrorFetching := Tidal.FetchAlbumTracks(AlbumID)

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

	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Components.Album.Playing.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Success", Locale),
			Description: Localizations.GetFormat("Components.Album.Playing.Description", Locale, len(AlbumSongs)),
			Color:       0xB3FFBA,

		})).Build())

}