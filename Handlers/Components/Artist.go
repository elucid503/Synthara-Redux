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

func ArtistEnqueue(Event *events.ComponentInteractionCreate) {

	// Defer response since fetching artist tracks may take time
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

				Title: Localizations.Get("Components.Artist.InvalidID.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.InvalidID.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	ArtistIDStr := Parts[1]
	ArtistID, ParseErr := strconv.ParseInt(ArtistIDStr, 10, 64)

	if ParseErr != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Components.Artist.InvalidID.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.InvalidID.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	ArtistSongs, ErrorFetching := Tidal.FetchArtistTopTracks(ArtistID)

	if ErrorFetching != nil || len(ArtistSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Components.Artist.FetchError.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.FetchError.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Determine if the current song is by this artist and find its position
	StartIndex := 0

	if Guild.Queue.Current != nil && Guild.Queue.Current.ArtistID == ArtistID {

		CurrentID := Guild.Queue.Current.TidalID

		for i, s := range ArtistSongs {

			if s.TidalID == CurrentID {

				StartIndex = i + 1 // enqueue songs AFTER the current track
				break

			}

		}

	}

	EnqueueSongs := ArtistSongs

	if StartIndex > 0 && StartIndex < len(ArtistSongs) {

		EnqueueSongs = ArtistSongs[StartIndex:]

	} else if StartIndex >= len(ArtistSongs) {

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

	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdate().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:  Localizations.Get("Components.Artist.Enqueued.Title", Locale),
			Author: Localizations.Get("Embeds.Categories.Success", Locale),

			Description: func() string {

				if StartIndex > 0 && Guild.Queue.Current != nil && len(EnqueueSongs) > 0 {

					return Localizations.GetFormat("Components.Artist.Enqueued.DescriptionAfterCurrent", Locale, len(EnqueueSongs), Guild.Queue.Current.Title)

				}

				return Localizations.GetFormat("Components.Artist.Enqueued.Description", Locale, len(EnqueueSongs))

			}(),

		})).
		AddActionRow(ViewQueueButton))

}

func ArtistPlay(Event *events.ComponentInteractionCreate) {

	// Defer response since fetching artist tracks may take time
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

				Title: Localizations.Get("Components.Artist.InvalidID.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.InvalidID.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	ArtistIDStr := Parts[1]
	ArtistID, ParseErr := strconv.ParseInt(ArtistIDStr, 10, 64)

	if ParseErr != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Components.Artist.InvalidID.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.InvalidID.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Fetch artist top tracks from Tidal

	ArtistSongs, ErrorFetching := Tidal.FetchArtistTopTracks(ArtistID)

	if ErrorFetching != nil || len(ArtistSongs) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title: Localizations.Get("Components.Artist.FetchError.Title", Locale),
				Author: Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Components.Artist.FetchError.Description", Locale),
				Color: Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild.Queue.Clear()

	for _, Song := range ArtistSongs {

		SongCopy := Song
		Guild.Queue.Add(&SongCopy, Event.User().ID.String())

	}

	if len(Guild.Queue.Upcoming) > 0 {

		Guild.Queue.Play()

	}

	Event.Client().Rest.UpdateInteractionResponse(Event.Client().ApplicationID, Event.Token(), discord.NewMessageUpdate().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title: Localizations.Get("Components.Artist.Playing.Title", Locale),
			Author: Localizations.Get("Embeds.Categories.Success", Locale),
			Description: Localizations.GetFormat("Components.Artist.Playing.Description", Locale, len(ArtistSongs)),
			Color: Utils.PRIMARY,

		})))

}
