package Utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
)

var DiscordClient bot.Client

func InitDiscordClient() error {

	InitializedClient, ErrorInitializing := disgo.New(os.Getenv("DISCORD_TOKEN"), bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuildVoiceStates)))

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

func WaitUntilDiscordClientReady() error {

    ContextToUse, CancelFunc := context.WithTimeout(context.TODO(), 10 * time.Second) // 10s timeout
    defer CancelFunc()

    WaitChannel := make(chan struct{}, 1)

    DiscordClient.AddEventListeners(bot.NewListenerFunc(func(Event bot.Event) {

        if _, EventIsReady := Event.(*events.Ready); EventIsReady {

            select {

				case WaitChannel <- struct{}{}:

            default:

            }

        }
		
    }))

    select {

		case <-WaitChannel:

			Logger.Info("Discord client is ready.")
			return nil // Successfully received Ready event

		case <-ContextToUse.Done():

			Logger.Error("Timeout waiting for Discord client to be ready.")
			return ContextToUse.Err() // Timeout or cancellation

		}

}