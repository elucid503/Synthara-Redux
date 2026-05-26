package Receive

import (
	"fmt"
	"sync"

	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

type CommandHandler func(GuildID, UserID snowflake.ID, Args string)

var (

	commandRegistry   = make(map[string]CommandHandler)
	commandRegistryMu sync.RWMutex

)

// Register binds a handler to a voice command verb.
func Register(Command string, Handler CommandHandler) {

	if Command == "" || Handler == nil {

		return

	}

	commandRegistryMu.Lock()
	defer commandRegistryMu.Unlock()

	commandRegistry[Command] = Handler

}

func lookupHandler(Command string) CommandHandler {

	commandRegistryMu.RLock()
	defer commandRegistryMu.Unlock()

	return commandRegistry[Command]

}

type Dispatcher struct {

	GuildID snowflake.ID

}

func NewDispatcher(GuildID snowflake.ID) *Dispatcher {

	return &Dispatcher{GuildID: GuildID}

}

func (D *Dispatcher) Dispatch(GuildID, UserID snowflake.ID, Cmd ParsedCommand) {

	if D == nil {

		return

	}

	Handler := lookupHandler(Cmd.Command)

	if Handler == nil {

		return

	}

	go func() {

		defer func() {

			if r := recover(); r != nil {

				Utils.Logger.Error("Receive", fmt.Sprintf("Voice handler panic (cmd=%s): %v", Cmd.Command, r))

			}

		}()

		Handler(GuildID, UserID, Cmd.Args)

	}()

}
