package Server

import (
	"Synthara-Redux/Globals"

	"github.com/gin-gonic/gin"
)

func InitializeRoutes() {

	Globals.WebServer.GET("/Queues/:ID", HandleQueuePage)
	Globals.WebServer.GET("/API/Queue", HandleWSConnections)

}

func HandleQueuePage(Context *gin.Context) {

	Context.File("./Web/dist/index.html")

}