package Components

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
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
				Color:       Utils.ERROR,

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
				Color:       Utils.ERROR,

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
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,
			
		})

		return

	}
	
	// Determine if the current song is part of this album and find its position
	StartIndex := 0

	if Guild.Queue.Current != nil && Guild.Queue.Current.AlbumID == AlbumID {

		CurrentID := Guild.Queue.Current.TidalID

		for i, s := range AlbumSongs {

			if s.TidalID == CurrentID {

				StartIndex = i + 1 // enqueue songs AFTER the current track
				break

			}

		}

	}

	EnqueueSongs := AlbumSongs

	if StartIndex > 0 && StartIndex < len(AlbumSongs) {

		EnqueueSongs = AlbumSongs[StartIndex:]

	} else if StartIndex >= len(AlbumSongs) {

		// Current song is the last track; nothing to enqueue
		EnqueueSongs = []Tidal.Song{}

	}

	for _, Song := range EnqueueSongs {

		SongCopy := Song
		Guild.Queue.Add(&SongCopy, Event.User().ID.String())

	}
		
	QueueURL := fmt.Sprintf("%s/Queues/%s?View=Queue", strings.TrimRight(os.Getenv("DOMAIN"), "/"), GuildID.String())

	ViewQueueButton := discord.NewButton(discord.ButtonStyleLink, Localizations.Get("Embeds.Queue.View", Locale), "", QueueURL, snowflake.ID(0)).WithEmoji(discord.ComponentEmoji{

		ID: snowflake.MustParse(Icons.GetID(Icons.Albums)),

	})

	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Components.Album.Enqueued.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Success", Locale),

			Description: func() string {

				if StartIndex > 0 && Guild.Queue.Current != nil && len(EnqueueSongs) > 0 {

					return Localizations.GetFormat("Components.Album.Enqueued.DescriptionAfterCurrent", Locale, len(EnqueueSongs), Guild.Queue.Current.Title)
				
				}

				return Localizations.GetFormat("Components.Album.Enqueued.Description", Locale, len(EnqueueSongs))
			
			}(),

		})).
		AddActionRow(ViewQueueButton).
		Build())

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
				Color:       Utils.ERROR,

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
				Color:       Utils.ERROR,

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
				Color:       Utils.ERROR,

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
			Color:       Utils.PRIMARY,

		})).Build())

}