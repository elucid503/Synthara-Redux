package Handlers

import (
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func InitializeCommands() {

	// Ping Command

	Command := discord.SlashCommandCreate{

		Name:        "ping",
		Description: "Replies with Pong!",

	}

	Utils.DiscordClient.Rest().SetGlobalCommands(Utils.DiscordClient.ApplicationID(), []discord.ApplicationCommandCreate{Command})

	Utils.Logger.Info("Slash commands initialized.")

}

func InitializeHandlers() {

	Utils.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event bot.Event) {

		switch E := Event.(type) {

		case *events.ApplicationCommandInteractionCreate:

			switch E.Data.CommandName() {

				case "ping":

					Commands.PingCommand(E)

			}

			Utils.Logger.Info("Received and handled command: " + E.Data.CommandName());

		}
					
	}))

	Utils.Logger.Info("Event handlers initialized.")

}