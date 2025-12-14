package Commands

import (
	"Synthara-Redux/APIs/Innertube"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayCommand(Event *events.ApplicationCommandInteractionCreate) {

	// Get the search query from command options

	Data := Event.SlashCommandInteractionData()
	Query := Data.String("query")

	if Query == "" {

		Event.CreateMessage(discord.MessageCreate{

			Content:"Please provide a search query!",

		})

		return

	}

	// Check if user is in a voice channel

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a guild to use this command!",

		})

		return

	}

	GuildID := *Event.GuildID()

	VoiceState, VoiceStateError := Event.Client().Rest.GetUserVoiceState(GuildID, Event.User().ID);

	if VoiceStateError != nil || VoiceState.ChannelID == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a voice channel to use this command!",

		})

		return

	}

	// ChannelID := *VoiceState.ChannelID

	// Search for songs

	SearchResults := Innertube.SearchForSongs(Query)

	if len(SearchResults) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No results found for your query!",

		})

		return

	}


	
}
