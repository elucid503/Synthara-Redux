package Server

import (
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"net/http"

	"github.com/disgoorg/snowflake/v2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (

	OperationPause  = "Pause"
	OperationResume = "Resume"

	OperationNext   = "Next"
	OperationLast   = "Last"

	OperationSeek   = "Seek"

)

var Upgrader = websocket.Upgrader{

	CheckOrigin: func(r *http.Request) bool {

		return true

	},

}

func HandleSocket(Context *gin.Context) {

	GuildIDStr := Context.Query("guild_id")

	if GuildIDStr == "" {

		Context.JSON(http.StatusBadRequest, gin.H{"error": "guild_id is required"})
		return

	}

	GuildID, ErrorParsing := snowflake.Parse(GuildIDStr)

	if ErrorParsing != nil {

		Context.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild_id"})
		return

	}

	Guild := Structs.GetGuild(GuildID)

	if Guild == nil {

		Context.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
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

	InitialState := map[string]interface{}{

		"Event": Structs.Event_Initial,

		"Data": map[string]interface{}{

			"Current": Guild.Queue.Current,
			"State": Guild.Queue.State,

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

	switch Operation {

	case OperationPause:

		Guild.Queue.SetState(Structs.StatePaused)

	case OperationResume:

		Guild.Queue.SetState(Structs.StatePlaying)

	case OperationNext:

		Guild.Queue.Next()

	case OperationLast:

		Guild.Queue.Last()

	case OperationSeek:

		if Offset, Ok := Message["Offset"].(float64); Ok {

			if Guild.Queue.PlaybackSession != nil {

				Guild.Queue.PlaybackSession.Seek(int(Offset))

			}

		}

	}

}