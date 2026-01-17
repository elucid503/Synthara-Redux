package Handlers

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Handlers/Autocomplete"
	"Synthara-Redux/Handlers/Commands"
	"Synthara-Redux/Handlers/Components"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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

	Contexts []discord.InteractionContextType `json:"contexts,omitempty"`

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
			Contexts:                 Command.Contexts,

		}

	}

	_, ErrorSetting := Globals.DiscordClient.Rest.SetGlobalCommands(Globals.DiscordClient.ApplicationID, CommandsToRegister)

	if ErrorSetting != nil {

		Utils.Logger.Error("Failed to set global slash commands: " + ErrorSetting.Error())
		return

	}

	Utils.Logger.Info("Slash commands updated.")

}

func InitializeHandlers() {
	
	// Ready

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.Ready) {

		Utils.Logger.Info("Discord Client is ready!")

	}))

	// Command Interactions

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.ApplicationCommandInteractionCreate) {

		go func ()  {
			
			Guild := Structs.GetGuild(*Event.GuildID(), false)

			if Guild != nil {

				// Reset inactivity timer on activity

				Guild.ResetInactivityTimer()

			}

		}()

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

				case "jump":

					Commands.Jump(Event)

				case "replay":

					Commands.Replay(Event)

				case "repeat":

					Commands.Repeat(Event)

				case "shuffle":

					Commands.Shuffle(Event)

				case "autoplay":

					Commands.Autoplay(Event)

				case "lyrics":

					Commands.Lyrics(Event)

				case "controls":

					Commands.Controls(Event)

				case "queue":

					Commands.Queue(Event)

				case "move":

					Commands.Move(Event)

				case "lock":

					Commands.Lock(Event)

				case "unlock":

					Commands.Unlock(Event)

				case "stats":

					Commands.Stats(Event)

				case "album":

					Commands.Album(Event)

				case "leave":

					Commands.Leave(Event)

			case "notify":

				Commands.Notify(Event)

			}

			// Check for unseen notifications after command execution
			CheckAndDisplayNotification(Event, Event.User().ID)

		}()

	}))

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.AutocompleteInteractionCreate) {

		go func() {

			switch Event.Data.CommandName {

				case "play":

					Autocomplete.PlayAutocomplete(Event)

				case "jump":

					Autocomplete.JumpAutocomplete(Event)

				case "replay":

					Autocomplete.ReplayAutocomplete(Event)

				case "move":

					Autocomplete.MoveAutocomplete(Event)

			}

		}()

	}))

	// Component Interactions

	Globals.DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event *events.ComponentInteractionCreate) {

		go func() {

			CustomID := Event.Data.CustomID()

			// Parses custom ID for arguments (ex: "RemoveSong:YouTubeID")

			Parts := strings.Split(CustomID, ":")
			BaseID := Parts[0]
			
			switch BaseID {

				case "Last":

					Components.Last(Event)

				case "Lyrics":

					Components.Lyrics(Event)

				case "Play":

					Components.Resume(Event)

				case "Pause":

					Components.Pause(Event)

				case "Queue":

					Components.Queue(Event)

				case "Next":

					Components.Next(Event)

				case "RemoveSong":

					if len(Parts) > 1 {

						TidalID, ParseErr := strconv.ParseInt(Parts[1], 10, 64)

						if ParseErr == nil {

							Components.RemoveSong(Event, TidalID)

						}

					}

				case "JumpToSong":

					if len(Parts) > 1 {

						TidalID, ParseErr := strconv.ParseInt(Parts[1], 10, 64)

						if ParseErr == nil {

							Components.JumpToSong(Event, TidalID)

						}

					}

				case "Reconnect":

					Components.Reconnect(Event)

				case "AutoPlay":

					Components.Autoplay(Event)

				case "AlbumEnqueue":

					Components.AlbumEnqueue(Event)

				case "AlbumPlay":

					Components.AlbumPlay(Event)

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

				ReconnectButton := discord.NewButton(discord.ButtonStyleSecondary, Localizations.Get("Buttons.Reconnect", Guild.Locale.Code()), "Reconnect", "", 0).WithEmoji(discord.ComponentEmoji{

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

func CheckAndDisplayNotification(Event *events.ApplicationCommandInteractionCreate, UserID snowflake.ID) {

	User, UserError := Structs.GetUser(UserID.String())

	if UserError != nil {

		return

	}

	LatestNotification, NotifError := Structs.GetLatestNotification()

	if NotifError != nil { return } // No notifications found, probably
	
	if User.LastNotificationSeen == LatestNotification.ID { return }

	// Mark as seen

	User.SetLastNotificationSeen(LatestNotification.ID)

	// Create notification embed

	NotificationEmbed := Utils.CreateEmbed(Utils.EmbedOptions{

		Title:       LatestNotification.Title,
		Description: LatestNotification.Description,
		Color:       0xFFA500, // Orange color for notifications

	})

	// Sends as follow-up

	Event.Client().Rest.CreateFollowupMessage(Event.ApplicationID(), Event.Token(), discord.MessageCreate{

		Embeds: []discord.Embed{NotificationEmbed},

	})

}