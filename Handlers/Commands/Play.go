package Commands

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Structs"
	"fmt"

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

	VoiceState, VoiceStateExists := Globals.DiscordClient.Caches.VoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		VoiceState, VoiceStateError := Event.Client().Rest.GetUserVoiceState(GuildID, Event.User().ID);
		
		if VoiceStateError != nil || VoiceState.ChannelID == nil {

			VoiceStateExists = false

		} else {

			VoiceStateExists = true

		}
		
	}

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a voice channel to use this command!",

		})

		return

	}

	ChannelID := *VoiceState.ChannelID

	// Search for songs

	SearchResults := Innertube.SearchForSongs(Query)

	if len(SearchResults) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No results were found for your query!",

		})

		return

	}

	Guild := Structs.GetOrCreateGuild(GuildID);

	// Connect to voice channel

	ErrorConnecting := Guild.Connect(ChannelID)

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "Failed to connect to voice channel: " + ErrorConnecting.Error(),

		})

		return

	}
	
	// Play/Add result 

	IsCurrent := Guild.Queue.Add(SearchResults[0])

	if IsCurrent {
		
		Event.CreateMessage(discord.MessageCreate{

			Content: fmt.Sprintf("Now playing %s by %s", SearchResults[0].Title, SearchResults[0].Artists[0]),

		})

	}
	
	

}