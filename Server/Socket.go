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

type WebIdentifier struct {
	Name string
}

const (
	OperationPause  = "Pause"
	OperationResume = "Resume"

	OperationNext = "Next"
	OperationLast = "Last"

	OperationJump    = "Jump"
	OperationRemove  = "Remove"
	OperationMove    = "Move"
	OperationReplay  = "Replay"
	OperationEnqueue = "Enqueue"
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

		Utils.Logger.Error("WebSocket", fmt.Sprintf("Failed to upgrade websocket: %s", ErrorUpgrading.Error()))
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

			"Current":  Guild.Queue.Current,
			"Previous": Guild.Queue.Previous,
			"Upcoming": Guild.Queue.Upcoming,

			"State":    Guild.Queue.State,
			"Progress": Progress,

			"OAuthEnabled":   OAuthEnabled(),
			"Authenticated":  WebAuthenticated(Context.Request),
			"ControlsLocked": WebControlsLocked(Guild.Features.Locked, Context.Request),
			"GuildLocked":    Guild.Features.Locked,
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

		HandleWSMessage(Guild, Context.Request, Message)

	}

	// Unregisters connection; done

	Guild.Queue.SocketMutex.Lock()
	delete(Guild.Queue.WebSockets, Socket)
	Guild.Queue.SocketMutex.Unlock()

}

func HandleWSMessage(Guild *Structs.Guild, Request *http.Request, Message map[string]interface{}) {

	Operation, Ok := Message["Operation"].(string)

	if !Ok {
		return
	}

	Identifier := WebIdentifier{Name: WebUserForControls(Request)}

	if WebControlsLocked(Guild.Features.Locked, Request) {

		Guild.Queue.SendToWebsockets("ERROR", map[string]interface{}{

			"Message": WebControlsLockMessage(Guild.Features.Locked, Request),
		})

		return

	}

	Locale := Guild.Locale.Code()

	switch Operation {

	case OperationPause:

		Guild.Queue.SetState(Structs.StatePaused)

		SendWebOperationMessage(Guild, "Commands.Pause.Title", "Web.Operations.Pause.Description", Locale, Identifier)

	case OperationResume:

		Guild.Queue.SetState(Structs.StatePlaying)

		SendWebOperationMessage(Guild, "Commands.Resume.Title", "Web.Operations.Resume.Description", Locale, Identifier)

	case OperationNext:

		Advanced, Ended := Guild.Queue.Next(true)

		if Ended {

			SendWebOperationMessage(Guild, "Embeds.Notifications.QueueEnded.Title", "Embeds.Notifications.QueueEnded.Description", Locale, Identifier)

		} else if Advanced && Guild.Queue.Current != nil {

			SendWebOperationMessageWithSong(Guild, "Commands.Next.Title", "Web.Operations.Next.Description", Locale, Identifier, Guild.Queue.Current.Title)

		}

	case OperationLast:

		Guild.Queue.Last(true)

		if Guild.Queue.Current != nil {

			SendWebOperationMessageWithSong(Guild, "Commands.Last.Title", "Web.Operations.Last.Description", Locale, Identifier, Guild.Queue.Current.Title)

		}

	case OperationJump:

		Index, Ok := Message["Index"].(float64)

		if !Ok {

			return

		}

		Guild.Queue.Jump(int(Index))

		if Guild.Queue.Current != nil {

			SendWebOperationMessageWithSong(Guild, "Web.Operations.Jump.Title", "Web.Operations.Jump.Description", Locale, Identifier, Guild.Queue.Current.Title)

		}

	case OperationRemove:

		Index, Ok := Message["Index"].(float64)

		if !Ok {

			return

		}

		Guild.Queue.Remove(int(Index))

		SendWebOperationMessage(Guild, "Web.Operations.Remove.Title", "Web.Operations.Remove.Description", Locale, Identifier)

	case OperationMove:

		FromIndex, FromOk := Message["FromIndex"].(float64)
		ToIndex, ToOk := Message["ToIndex"].(float64)

		if !FromOk || !ToOk {

			return

		}

		Guild.Queue.Move(int(FromIndex), int(ToIndex))

		SendWebOperationMessage(Guild, "Web.Operations.Move.Title", "Web.Operations.Move.Description", Locale, Identifier)

	case OperationReplay:

		Index, Ok := Message["Index"].(float64)

		if !Ok {

			return

		}

		Guild.Queue.Replay(int(Index))

		if Guild.Queue.Current != nil {

			SendWebOperationMessageWithSong(Guild, "Web.Operations.Replay.Title", "Web.Operations.Replay.Description", Locale, Identifier, Guild.Queue.Current.Title)

		}

	case OperationEnqueue:

		TidalID, Ok := Message["TidalID"].(float64)

		if !Ok {

			return

		}

		URI := fmt.Sprintf("Synthara-Redux:Song:%d", int64(TidalID))
		SongFound, _, ErrorHandling := Guild.HandleURI(URI, Identifier.Name)

		if ErrorHandling != nil {

			Guild.Queue.SendToWebsockets("ERROR", map[string]interface{}{

				"Message": "Failed to add song to queue.",

			})

			return

		}

		if SongFound != nil {

			SendWebOperationMessageWithSong(Guild, "Web.Operations.Enqueue.Title", "Web.Operations.Enqueue.Description", Locale, Identifier, SongFound.Title)

		}

	}

}

// SendWebOperationMessage sends a notification to Discord for web operations
func SendWebOperationMessage(Guild *Structs.Guild, TitleKey string, DescKey string, Locale string, Identifier WebIdentifier) {

	if Guild.Channels.Text == 0 {

		return // No text channel set

	}

	go func() {

		Description := Localizations.GetFormat(DescKey, Locale, Identifier.Name)

		_, _ = Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreate().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get(TitleKey, Locale),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
				Description: Description,

			})))

	}()

}

// SendWebOperationMessageWithSong sends a notification with song name
func SendWebOperationMessageWithSong(Guild *Structs.Guild, TitleKey string, DescKey string, Locale string, Identifier WebIdentifier, SongTitle string) {

	if Guild.Channels.Text == 0 {

		return // No text channel set

	}

	go func() {

		Description := Localizations.GetFormat(DescKey, Locale, Identifier.Name, SongTitle)

		_, _ = Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreate().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get(TitleKey, Locale),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
				Description: Description,

			})))

	}()

}
