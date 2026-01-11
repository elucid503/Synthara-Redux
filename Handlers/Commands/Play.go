package Commands

import (
	"Synthara-Redux/APIs"
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Play(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// Get the search query from command options

	Data := Event.SlashCommandInteractionData()
	Query := Data.String("query")

	if Query == "" {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NoQuery.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NoQuery.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	// Check if user is in a voice channel

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInGuild.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInGuild.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	GuildID := *Event.GuildID()

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	ChannelID := VoiceState.ChannelID

	Guild := Structs.GetGuild(GuildID, true) // creates if not found

	// Connect to voice channel

	ErrorConnecting := Guild.Connect(*ChannelID, Event.Channel().ID())

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.FailedToConnect.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.FailedToConnect.Description", Locale, ErrorConnecting.Error()),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	// Route the input to a URI

	URI, ErrorRouting := APIs.Route(Query)

	if ErrorRouting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.InvalidInput.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.InvalidInput.Description", Locale, ErrorRouting.Error()),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	// Handle the URI

	SongFound, Pos, ErrorHandling := Guild.HandleURI(URI, Event.User().Mention())

	if ErrorHandling != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.FailedToHandle.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.FailedToHandle.Description", Locale, ErrorHandling.Error()),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	// Send response with current song info

	State := Innertube.QueueInfo{

		GuildID: GuildID,

		SongPosition: Pos,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{SongFound.Embed(State)},

	})

}