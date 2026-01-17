package Server

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"net/http"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (

	OperationPause  = "Pause"
	OperationResume = "Resume"

	OperationNext   = "Next"
	OperationLast   = "Last"

	OperationJump   = "Jump"
	OperationRemove = "Remove"
	OperationMove   = "Move"
	OperationReplay = "Replay"

)

var Upgrader = websocket.Upgrader{

	CheckOrigin: func(r *http.Request) bool {

		return true

	},

}

func HandleWSConnections(Context *gin.Context) {

	GuildIDStr := Context.Query("ID")

	if GuildIDStr == "" {

		Context.JSON(http.StatusBadRequest, gin.H{"Error": "Queue ID is required"})
		return

	}

	GuildID, ErrorParsing := snowflake.Parse(GuildIDStr)

	if ErrorParsing != nil {

		Context.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid Queue ID"})
		return

	}

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Context.JSON(http.StatusNotFound, gin.H{"Error": "Guild not found"})
		return

	}

	Socket, ErrorUpgrading := Upgrader.Upgrade(Context.Writer, Context.Request, nil)

	if ErrorUpgrading != nil {

		Utils.Logger.Error(fmt.Sprintf("Failed to upgrade websocket: %s", ErrorUpgrading.Error()))
		return

	}

	defer Socket.Close()

	// Register connection

	Guild.Queue.SocketMutex.Lock()
	Guild.Queue.WebSockets[Socket] = true
	Guild.Queue.SocketMutex.Unlock()

	// Send initial state

	Progress := int64(0)
	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Progress = Guild.Queue.PlaybackSession.Streamer.Progress

	}

	InitialState := map[string]interface{}{

		"Event": Structs.Event_Initial,

		"Data": map[string]interface{}{

			"Current": Guild.Queue.Current,
			"Previous": Guild.Queue.Previous,
			"Upcoming": Guild.Queue.Upcoming,

			"State": Guild.Queue.State,
			"Progress": Progress,

		},

	}

	Socket.WriteJSON(InitialState)

	// Keep connection alive and listen for close

	for {

		var Message map[string]interface{}
		
		ErrorReading := Socket.ReadJSON(&Message)

		if ErrorReading != nil {

			break

		}

		HandleWSMessage(Guild, Message)

	}

	// Unregisters connection; done

	Guild.Queue.SocketMutex.Lock()
	delete(Guild.Queue.WebSockets, Socket)
	Guild.Queue.SocketMutex.Unlock()

}

func HandleWSMessage(Guild *Structs.Guild, Message map[string]interface{}) {

	Operation, Ok := Message["Operation"].(string)

	if !Ok { return }

	// Check if guild is locked
	if Guild.Features.Locked {

		Guild.Queue.SendToWebsockets("ERROR", map[string]interface{}{

			"Message": "Web controls are locked. Use <code>/unlock</code> to enable.",

		})

		return

	}

	Locale := Guild.Locale.Code()

	switch Operation {

		case OperationPause:

			Guild.Queue.SetState(Structs.StatePaused)
			
			SendWebOperationMessage(Guild, "Commands.Pause.Title", "Commands.Pause.Description", Locale)

		case OperationResume:

			Guild.Queue.SetState(Structs.StatePlaying)

			SendWebOperationMessage(Guild, "Commands.Resume.Title", "Commands.Resume.Description", Locale)

		case OperationNext:

			Guild.Queue.Next()

		case OperationLast:

			Guild.Queue.Last()

			if Guild.Queue.Current != nil {

				SendWebOperationMessageWithSong(Guild, "Commands.Last.Title", "Commands.Last.Description", Locale, Guild.Queue.Current.Title)

			}

		case OperationJump:

			Index, Ok := Message["Index"].(float64)

			if !Ok { return }

			Guild.Queue.Jump(int(Index))
			
			if Guild.Queue.Current != nil {

				SendWebOperationMessageWithSong(Guild, "Web.Operations.Jump.Title", "Web.Operations.Jump.Description", Locale, Guild.Queue.Current.Title)

			}

		case OperationRemove:

			Index, Ok := Message["Index"].(float64)

			if !Ok { return }

			Guild.Queue.Remove(int(Index))

			SendWebOperationMessage(Guild, "Web.Operations.Remove.Title", "Web.Operations.Remove.Description", Locale)

		case OperationMove:

			FromIndex, FromOk := Message["FromIndex"].(float64)
			ToIndex, ToOk := Message["ToIndex"].(float64)

			if !FromOk || !ToOk { return }

			Guild.Queue.Move(int(FromIndex), int(ToIndex))

			SendWebOperationMessage(Guild, "Web.Operations.Move.Title", "Web.Operations.Move.Description", Locale)

		case OperationReplay:

			Index, Ok := Message["Index"].(float64)

			if !Ok { return }

			Guild.Queue.Replay(int(Index))
			
			if Guild.Queue.Current != nil {

				SendWebOperationMessageWithSong(Guild, "Web.Operations.Replay.Title", "Web.Operations.Replay.Description", Locale, Guild.Queue.Current.Title)

			}

	}

}

// SendWebOperationMessage sends a notification to Discord for web operations
func SendWebOperationMessage(Guild *Structs.Guild, TitleKey string, DescKey string, Locale string) {

	if Guild.Channels.Text == 0 {

		return // No text channel set

	}

	go func() {

		Prefix := Localizations.Get("Web.Prefix", Locale)
		Description := Prefix + " " + Localizations.Get(DescKey, Locale)

		_, _ = Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get(TitleKey, Locale),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
				Description: Description,

			})).
			Build())

	}()

}

// SendWebOperationMessageWithSong sends a notification with song name
func SendWebOperationMessageWithSong(Guild *Structs.Guild, TitleKey string, DescKey string, Locale string, SongTitle string) {

	if Guild.Channels.Text == 0 {

		return // No text channel set

	}

	go func() {

		Prefix := Localizations.Get("Web.Prefix", Locale)
		Description := Prefix + " " + Localizations.GetFormat(DescKey, Locale, SongTitle)

		_, _ = Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get(TitleKey, Locale),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
				Description: Description,

			})).
			Build())

	}()

}