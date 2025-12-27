package Handlers

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Handlers/Autocomplete"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func InitializeCommands() {

	// Ping Command

	PingCommand := discord.SlashCommandCreate{

		Name:        "ping",
		Description: "Replies with Pong!",

	}

	// Play Command

	PlayCommand := discord.SlashCommandCreate{

		Name:        "play",
		Description: "Search and play a song",

		Options: []discord.ApplicationCommandOption{

			discord.ApplicationCommandOptionString{

				Name:        "query",
				Description: "The song to search for",
				Required:    true,
				Autocomplete: true,

			},

		},

	}

	// Pause Command

	PauseCommand := discord.SlashCommandCreate{

		Name:        "pause",
		Description: "Pauses the currently playing song",

	}

	// Resume Command 

	ResumeCommand := discord.SlashCommandCreate{

		Name:        "resume",
		Description: "Resumes the currently paused song",

	}

	// Next Command

	NextCommand := discord.SlashCommandCreate{

		Name:        "next",
		Description: "Skips to the next song in the queue",

	}

	// Last Command

	LastCommand := discord.SlashCommandCreate{

		Name:        "last",
		Description: "Plays the previously played song",

	}

	// Seek Command

	SeekCommand := discord.SlashCommandCreate{

		Name:        "seek",
		Description: "Seek forward or backward in the current song",

		Options: []discord.ApplicationCommandOption{

			discord.ApplicationCommandOptionInt{

				Name:        "offset",
				Description: "Seconds to seek (positive = forward, negative = backward)",
				Required:    true,

			},

		},

	}

	Globals.DiscordClient.Rest.SetGlobalCommands(Globals.DiscordClient.ApplicationID, []discord.ApplicationCommandCreate{PingCommand, PlayCommand, PauseCommand, ResumeCommand, NextCommand, LastCommand, SeekCommand})

	Utils.Logger.Info("Slash commands initialized.")

}

func InitializeHandlers() {
	
	// Ready

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.Ready) {

		Utils.Logger.Info("Discord Client is ready!")

	}))

	// Command Interactions

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.ApplicationCommandInteractionCreate) {

		go func ()  {
			
			switch Event.Data.CommandName() {

				case "ping":

					Commands.PingCommand(Event)

				case "play":

					Commands.PlayCommand(Event)

				case "pause":

					Commands.PauseCommand(Event)

				case "resume": 

					Commands.ResumeCommand(Event)

				case "next":

					Commands.NextCommand(Event)

				case "last":

					Commands.LastCommand(Event)

				case "seek":

					Commands.SeekCommand(Event)

			}

			Utils.Logger.Info("Received and handled command: " + Event.Data.CommandName());

		}()
					
	}))

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.AutocompleteInteractionCreate) {

		go func() {

			switch Event.Data.CommandName {

				case "play":

					Autocomplete.PlayAutocomplete(Event)

			}

		}()

	}))

	// Voice State Updates

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.GuildVoiceStateUpdate) {

		if (Event.VoiceState.UserID != Globals.DiscordClient.ApplicationID) {

			return; // Not our bot

		}

		Guild := Structs.GetGuild(Event.VoiceState.GuildID)

		if (Event.VoiceState.ChannelID == nil && !Guild.Internal.Disconnecting) { // we do not want to call this if Disconnect() was already called...

			// Disconnected from voice channel, we should clean up the voice connection

			Guild.Disconnect(false)

			go func() { 

				_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.MessageCreate{

					Embeds: []discord.Embed{{

						Title: "Manually Disconnected",
						Description: "The Queue has been reset.",
						Color: 0xFFFFFF, // White
						
						Author: &discord.EmbedAuthor{

							Name: "Notifications",

						},

					}},

				})

				if ErrorSending != nil {

					Utils.Logger.Error(fmt.Sprintf("Error sending manual disconnect message to guild %s: %s", Guild.ID, ErrorSending.Error()))
				
				}

			}()

		}
		
	}))

	Utils.Logger.Info("Event handlers initialized.")

}