package main

import (
	"Synthara-Redux/APIs/Innertube"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Handlers"
	"Synthara-Redux/Utils"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load(".env")

	InitErr := Globals.InitDiscordClient()

	if InitErr != nil {

		os.Exit(1)

	}

	ConnectErr := Globals.ConnectDiscordClient()

	if ConnectErr != nil {

		os.Exit(1)

	}

	if (os.Getenv("REFRESH_COMMANDS") == "true") {

		Handlers.InitializeCommands()

	}
	
	Handlers.InitializeHandlers()

	InnerTubeError := Innertube.InitClient();

	if InnerTubeError != nil {

		os.Exit(1);

	}
	
	Utils.Hang()

}