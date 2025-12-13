package main

import (
	"Synthara-Redux/Handlers"
	"Synthara-Redux/Utils"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load(".env")

	InitErr := Utils.InitDiscordClient()

	if InitErr != nil {

		os.Exit(1)

	}

	ConnectErr := Utils.ConnectDiscordClient()

	if ConnectErr != nil {

		os.Exit(1)

	}

	WaitingErr := Utils.WaitUntilDiscordClientReady()

	if WaitingErr != nil {

		os.Exit(1)

	}

	if (os.Getenv("REFRESH_COMMANDS") == "true") {

		Handlers.InitializeCommands()

	}
	
	Handlers.InitializeHandlers()

	InnerTubeError := Utils.InitInnerTubeClient();

	if InnerTubeError != nil {

		os.Exit(1);

	}
	
	Utils.Hang()

}