package main

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals"
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

	Utils.Logger.Info("Starting Synthara-Redux...")

	LocalizationErr := Localizations.Initialize()

	if LocalizationErr != nil {

		Utils.Logger.Error(fmt.Sprintf("Failed to initialize/read localizations: %s", LocalizationErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Localizations loaded.")

	InitErr := Globals.InitDiscordClient()

	if InitErr != nil {

		Utils.Logger.Error(fmt.Sprintf("Failed to initialize Discord client: %s", InitErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Connecting to Discord...")

	ConnectErr := Globals.ConnectDiscordClient()

	if ConnectErr != nil {

		Utils.Logger.Error(fmt.Sprintf("Failed to connect to Discord: %s", ConnectErr.Error()))
		os.Exit(1)

	}

	Utils.Logger.Info("Connected to Discord!")

	if (os.Getenv("REFRESH_COMMANDS") == "true") {

		Handlers.InitializeCommands()

	}
	
	Handlers.InitializeHandlers()

	Globals.InitWebServer()
	Server.InitializeRoutes()
	
	go Globals.WebServer.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))

	Utils.Logger.Info(fmt.Sprintf("Web server running on port %s", os.Getenv("PORT")))

	InnerTubeError := Innertube.InitClient();

	if InnerTubeError != nil {

		Utils.Logger.Error(fmt.Sprintf("Failed to initialize Innertube client: %s", InnerTubeError.Error()))
		os.Exit(1);

	}
	
	Utils.Hang()

}