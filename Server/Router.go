package Server

import (
	"Synthara-Redux/Globals"
)

func InitializeRoutes() {

	Globals.WebServer.GET("/Queue", HandleSocket)

}