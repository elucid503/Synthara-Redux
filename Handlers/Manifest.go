package Handlers

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Utils"

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

			},

		},

	}

	Globals.DiscordClient.Rest.SetGlobalCommands(Globals.DiscordClient.ApplicationID, []discord.ApplicationCommandCreate{PingCommand, PlayCommand})

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

			}

			Utils.Logger.Info("Received and handled command: " + Event.Data.CommandName());

		}()
					
	}))

	Utils.Logger.Info("Event handlers initialized.")

}