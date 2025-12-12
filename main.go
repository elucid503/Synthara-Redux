package main

import (
	"Synthara-Redux/Utils"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load(".env")

	Utils.InitDiscordClient()

	Utils.ConnectDiscordClient()

	Utils.WaitUntilDiscordClientReady()

	// Connect events, eventually
	
	Utils.Hang()

}