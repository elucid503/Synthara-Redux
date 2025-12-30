package Commands

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayCommand(Event *events.ApplicationCommandInteractionCreate) {

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

	// Search for songs

	SearchResults := Innertube.SearchForSongs(Query)

	if len(SearchResults) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Play.Errors.NoResults", Locale),

		})

		return

	}

	Guild := Structs.GetGuild(GuildID, true) // creates if not found

	// Connect to voice channel

	ErrorConnecting := Guild.Connect(*ChannelID, Event.Channel().ID())

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.GetFormat("Commands.Play.Errors.FailedToConnect", Locale, ErrorConnecting.Error()),

		})

		return

	}
	
	// Play/Add result 

	Pos := Guild.Queue.Add(&SearchResults[0], Event.User().Mention())

	State := Innertube.QueueInfo{

		GuildID: GuildID,

		SongPosition: Pos,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{SearchResults[0].Embed(State)},
		
	})

	if Pos == 0 {
	
		Guild.Play(Guild.Queue.Current)

	}

}