package Utils

import (
	"Synthara-Redux/Globals"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func Hang() {

	SignalChan := make(chan os.Signal, 1)

	signal.Notify(SignalChan, syscall.SIGINT, syscall.SIGTERM) // Listens for termination signals

	<-SignalChan // Blocks until a signal is received

	Logger.Info("Shutting down and disconnecting Discord client...")

	Globals.DiscordClient.Close(context.TODO())
	
}

func GetNestedValue(Data interface{}, Keys ...string) (interface{}, bool) {

	Current := Data

	for _, Key := range Keys {

		if m, KeyValid := Current.(map[string]interface{}); KeyValid {

			Current, KeyValid = m[Key]

			if !KeyValid {

				return nil, false

			}

		} else {

			return nil, false

		}

	}

	return Current, true

}

func GetVoiceState(GuildID snowflake.ID, UserID snowflake.ID) (*discord.VoiceState, bool) {

	VoiceState, VoiceStateExists := Globals.DiscordClient.Caches.VoiceState(GuildID, UserID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		RestVoiceState, RestError := Globals.DiscordClient.Rest.GetUserVoiceState(GuildID, UserID)

		if RestError != nil || RestVoiceState == nil || RestVoiceState.ChannelID == nil {

			return nil, false

		}

		return RestVoiceState, true

	}

	return &VoiceState, true

}

func GetURI(Type string, ID string) string {

	return fmt.Sprintf("Synthara-Redux:%s:%s", Type, ID)
 
}

func Pluralize(Word string, Count int) string {

	if Count == 1 {

		return Word

	}

	return Word + "s"

}