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

			Content: Localizations.Get("Commands.Play.Errors.NoQuery", Locale),

		})

		return

	}

	// Check if user is in a voice channel

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Play.Errors.NotInGuild", Locale),

		})

		return

	}

	GuildID := *Event.GuildID()

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Play.Errors.NotInVoiceChannel", Locale),

		})

		return

	}

	ChannelID := VoiceState.ChannelID

	Guild := Structs.GetGuild(GuildID, true) // creates if not found

	// Connect to voice channel

	ErrorConnecting := Guild.Connect(*ChannelID, Event.Channel().ID())

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.GetFormat("Commands.Play.Errors.FailedToConnect", Locale, ErrorConnecting.Error()),

		})

		return

	}

	// Route the input to a URI

	URI, ErrorRouting := APIs.Route(Query)

	if ErrorRouting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.GetFormat("Commands.Play.Errors.InvalidInput", Locale, ErrorRouting.Error()),

		})

		return

	}

	// Handle the URI

	SongFound, Pos, ErrorHandling := Guild.HandleURI(URI, Event.User().Mention())

	if ErrorHandling != nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.GetFormat("Commands.Play.Errors.FailedToHandle", Locale, ErrorHandling.Error()),

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