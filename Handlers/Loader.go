package Handlers

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Handlers/Autocomplete"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"encoding/json"
	"fmt"
	"os"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

type CommandEntry struct {

	Name string `json:"name"`
	NameLocalizations map[discord.Locale]string `json:"name_localizations,omitempty"`
	
	Description string `json:"description"`
	DescriptionLocalizations map[discord.Locale]string `json:"description_localizations,omitempty"`
	
	Options []discord.UnmarshalApplicationCommandOption `json:"options,omitempty"`

}

func InitializeCommands() {

	File, ErrorReading := os.ReadFile("./Handlers/Commands.json")

	if ErrorReading != nil {

		Utils.Logger.Error("Failed to read Commands.json: " + ErrorReading.Error())

	}

	var Manifest []CommandEntry

	ErrorUnmarshaling := json.Unmarshal(File, &Manifest)

	if ErrorUnmarshaling != nil {

		Utils.Logger.Error("Failed to unmarshal Commands.json: " + ErrorUnmarshaling.Error())

	}

	CommandsToRegister := make([]discord.ApplicationCommandCreate, len(Manifest))

	for Index, Command := range Manifest {

		// Converts UnmarshalApplicationCommandOption to ApplicationCommandOption

		Options := make([]discord.ApplicationCommandOption, len(Command.Options))

		for i, opt := range Command.Options {

			Options[i] = opt.ApplicationCommandOption
			
		}

		CommandsToRegister[Index] = discord.SlashCommandCreate{

			Name:                     Command.Name,
			NameLocalizations:        Command.NameLocalizations,
			Description:              Command.Description,
			DescriptionLocalizations: Command.DescriptionLocalizations,
			Options:                  Options,

		}

	}

	Globals.DiscordClient.Rest.SetGlobalCommands(Globals.DiscordClient.ApplicationID, CommandsToRegister)

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

			// Switching for each command; not really a better way to do this
			
			switch Event.Data.CommandName() {

				case "ping":

					Commands.Ping(Event)

				case "play":

					Commands.Play(Event)

				case "pause":

					Commands.Pause(Event)

				case "resume": 

					Commands.Resume(Event)

				case "next":

					Commands.Next(Event)

				case "last":

					Commands.Last(Event)

				case "lyrics":

					Commands.Lyrics(Event)

				case "controls":

					Commands.Conrols(Event)

				case "queue":

					Commands.Queue(Event)

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

		Guild := Structs.GetGuild(Event.VoiceState.GuildID, false) // does not create if not found

		if (Guild == nil) {

			return; // No active guild session

		}

		if (Event.VoiceState.ChannelID == nil && !Guild.Internal.Disconnecting) { // we do not want to call this if Cleanup() was already called...

			// Disconnected from voice channel

			Guild.Cleanup(false)

			go func() { 

				ReconnectButton := discord.NewButton(discord.ButtonStylePrimary, Localizations.Get("Buttons.Reconnect", Guild.Locale.Code()), "Reconnect", "", 0).WithEmoji(discord.ComponentEmoji{

					ID: snowflake.MustParse(Icons.GetID(Icons.Call)),

				})

			_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
				AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Notifications.ManualDisconnect.Title", Guild.Locale.Code()),
					Author:      Localizations.Get("Embeds.Categories.Notifications", Guild.Locale.Code()),
					Description: Localizations.Get("Embeds.Notifications.ManualDisconnect.Description", Guild.Locale.Code()),

				})).
				AddActionRow(ReconnectButton).
				Build())

			if ErrorSending != nil {

				Utils.Logger.Error(fmt.Sprintf("Error sending manual disconnect message to guild %s: %s", Guild.ID, ErrorSending.Error()))
			
			}

		}()

		}
		
	}))

	Utils.Logger.Info("Event handlers initialized.")

}