package Globals

import (
	"context"
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

		return ErrorInitializing

	}

	DiscordClient = InitializedClient
	
	return nil

}

func ConnectDiscordClient() error {

	ContextToUse, CancelFunc := context.WithTimeout(context.TODO(), time.Second * 5); // 5s timeout
	defer CancelFunc() 

	ErrorConnecting := DiscordClient.OpenGateway(ContextToUse)

	if ErrorConnecting != nil {

		return ErrorConnecting
		
	}

	return nil

}