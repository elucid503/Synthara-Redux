package Commands

import (
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func PlayCommand(Event *events.ApplicationCommandInteractionCreate) {

	// Defer initial response
	
	ErrorDeferring := Event.DeferCreateMessage(false)

	if ErrorDeferring != nil {

		Utils.Logger.Error("Error deferring message: " + ErrorDeferring.Error())
		return

	}

	// Get the search query from command options

	Data := Event.SlashCommandInteractionData()
	Query := Data.String("query")

	if Query == "" {

		Msg := "Please provide a search query!"

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

		})

		return

	}

	// Check if user is in a voice channel

	if Event.Member() == nil {

		Msg := "You must be in a guild to use this command!"

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

		})

		return

	}

	GuildID := *Event.GuildID()

	VoiceState, VoiceStateError := Event.Client().Rest.GetUserVoiceState(GuildID, Event.User().ID);

	if VoiceStateError != nil || VoiceState.ChannelID == nil {

		Msg := "You must be in a voice channel to use this command!"

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

		})

		return

	}

	ChannelID := *VoiceState.ChannelID

	// Search for songs

	SearchResults := Utils.SearchInnerTubeSongs(Query)

	if len(SearchResults) == 0 {

		Msg := "No results found for your query!"

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

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

			Msg := "Failed to connect to voice channel: " + ErrorConnecting.Error()

			Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

				Content: &Msg,

			})

			return

		}

	}

	// Add song to queue

	Guild.AddToQueue(Song)

	// If nothing is currently playing, start playback

	if !Guild.HasCurrentSong() {

		Msg := fmt.Sprintf("Now playing: %s", Song.Title)

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

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

			// Disconnect after queue is empty

			Guild.DisconnectVoice()

		}()

	} else {

		Msg := fmt.Sprintf("Added to Queue: **%s** (Position: %d)", Song.Title, len(Guild.Queue.Next))

		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Content: &Msg,

		})

	}

}
