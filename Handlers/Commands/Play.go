package Commands

import (
	"Synthara-Redux/Utils"
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

	VoiceState, VoiceStateError := Event.Client().Rest.GetUserVoiceState(GuildID, Event.User().ID);

	if VoiceStateError != nil || VoiceState.ChannelID == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a voice channel to use this command!",

		})

		return

	}

	ChannelID := *VoiceState.ChannelID

	// Search for songs

	SearchResults := Utils.SearchInnerTubeSongs(Query)

	if len(SearchResults) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No results found for your query!",

		})

		return

	}

	// Get the first result

	Song := SearchResults[0]

	Guild := Utils.GetOrCreateGuild(GuildID)

	Guild.SetTextChannel(Event.Channel().ID())

	// Connect to voice channel if not already connected

	if Guild.VoiceConnection == nil || Guild.Channels.Voice != ChannelID {

		ErrorConnecting := Utils.ConnectToVoiceChannel(GuildID, ChannelID)

		if ErrorConnecting != nil {

			Event.CreateMessage(discord.MessageCreate{

				Content: "Failed to connect to voice channel: " + ErrorConnecting.Error(),

			})

			return

		}

	}

	// Add song to queue

	Guild.AddToQueue(Song)

	// If nothing is currently playing, start playback

	if !Guild.HasCurrentSong() {

		Event.CreateMessage(discord.MessageCreate{

			Content: fmt.Sprintf("Now playing: %s", Song.Title),

		})

		go func() {

			for Guild.AdvanceQueue() {

				CurrentSong := Guild.Queue.Current

				if CurrentSong == nil {

					break

				}

				ErrorPlaying := Utils.PlaySongInGuild(Guild, *CurrentSong)

				if ErrorPlaying != nil {

					Utils.Logger.Error("Error playing song: " + ErrorPlaying.Error())
					break

				}

			}

			// Disconnects after queue is empty

			Guild.Queue.Current = nil
			Guild.DisconnectVoice()

		}()

	} else {

		Event.CreateMessage(discord.MessageCreate{

			Content: fmt.Sprintf("Added to Queue: %s (Position: %d)", Song.Title, len(Guild.Queue.Next)),

		})

	}

}
