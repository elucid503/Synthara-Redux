package Utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
)

var DiscordClient *bot.Client

func InitDiscordClient() error {

	InitializedClient, ErrorInitializing := disgo.New(os.Getenv("DISCORD_TOKEN"), bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsNonPrivileged, gateway.IntentGuildVoiceStates)))

	if ErrorInitializing != nil {

		Logger.Error(fmt.Sprintf("Error initializing Discord client: %s", ErrorInitializing.Error()))
		return ErrorInitializing

	}

	DiscordClient = InitializedClient
	
	Logger.Info("Discord client initialized successfully")
	return nil

}

func ConnectDiscordClient() error {

	ContextToUse, CancelFunc := context.WithTimeout(context.TODO(), time.Second * 5); // 5s timeout
	defer CancelFunc() 

	ErrorConnecting := DiscordClient.OpenGateway(ContextToUse)

	if ErrorConnecting != nil {

		Logger.Error(fmt.Sprintf("Error connecting Discord client: %s", ErrorConnecting.Error()))

	}

	Logger.Info("Discord client connected successfully")
	return nil

}