package main

import (
	"Synthara-Redux/APIs/Apple"
	"Synthara-Redux/APIs/Spotify"
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/APIs/YouTube"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Icons"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Handlers"
	"Synthara-Redux/Server"
	"Synthara-Redux/Utils"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load(".env")

	Utils.Logger.Info("Startup", "Starting Synthara-Redux...")

	LocalizationErr := Localizations.Initialize()

	if LocalizationErr != nil {

		Utils.Logger.Error("Initialization", fmt.Sprintf("Failed to initialize/read localizations: %s", LocalizationErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Initialization", "Localizations loaded.")

	IconsErr := Icons.Initialize()

	if IconsErr != nil {

		Utils.Logger.Error("Initialization", fmt.Sprintf("Failed to initialize/read icons: %s", IconsErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Initialization", "Icons loaded.")

	MongoErr := Globals.InitMongoDB()

	if MongoErr != nil {

		Utils.Logger.Error("Database", fmt.Sprintf("Failed to initialize MongoDB: %s", MongoErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Database", "Connected to MongoDB.")

	InitErr := Globals.InitDiscordClient()

	if InitErr != nil {

		Utils.Logger.Error("Discord", fmt.Sprintf("Failed to initialize Discord client: %s", InitErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Discord", "Connecting to Discord...")

	ConnectErr := Globals.ConnectDiscordClient()

	if ConnectErr != nil {

		Utils.Logger.Error("Discord", fmt.Sprintf("Failed to connect to Discord: %s", ConnectErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Discord", "Connected to Discord!")

	if (os.Getenv("REFRESH_COMMANDS") == "true") {

		Handlers.InitializeCommands()

	}
	
	Handlers.InitializeHandlers()

	Globals.InitWebServer()
	Server.InitializeRoutes()
	
	go Globals.WebServer.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))

	Utils.Logger.Info("Web Server", fmt.Sprintf("Web server running on port %s", os.Getenv("PORT")))

	// Tidal Initialization

	Tidal.Init()

	Utils.Logger.Info("API", "Tidal client initialized.")

	// Spotify Initialization

	Spotify.Initialize(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	
	Utils.Logger.Info("API", "Spotify client initialized.")

	// Apple Music Initialization

	Apple.Initialize(os.Getenv("APPLE_JWT"))

	Utils.Logger.Info("API", "Apple Music client initialized.")

	// YT Initialization

	YouTube.Init()
	
	Utils.Logger.Info("API", "YouTube client initialized.")
	
	// Done with setup; now we wait for events

	Utils.Hang()

}