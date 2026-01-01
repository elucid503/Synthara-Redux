package Globals

import (
	"context"
	"os"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/gin-gonic/gin"
)

var DiscordClient *bot.Client
var WebServer *gin.Engine

func InitDiscordClient() error {

	InitializedClient, ErrorInitializing := disgo.New(os.Getenv("DISCORD_TOKEN"), bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsNonPrivileged, gateway.IntentGuildVoiceStates)))

	if ErrorInitializing != nil {

		return ErrorInitializing

	}

	DiscordClient = InitializedClient
	
	return nil

}

func InitWebServer() {

	gin.SetMode(gin.ReleaseMode)
	
	WebServer = gin.Default()

	WebServer.Static("/assets", "./Web/dist/assets")

	// CORS responses 

	WebServer.Use(func(Context *gin.Context) {

		Context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		Context.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH")
		Context.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
	
	})

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