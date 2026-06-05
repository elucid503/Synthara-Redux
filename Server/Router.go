package Server

import (
	"Synthara-Redux/Globals"

	"github.com/gin-gonic/gin"
)

func InitializeRoutes() {

	Globals.WebServer.GET("/Queues/:ID", RateLimitMiddleware(RateLimitPage), HandleQueuePage)

	Globals.WebServer.GET("/API/Queue", RateLimitMiddleware(RateLimitWSConnect), HandleWSConnections)

	Globals.WebServer.GET("/API/Suggestions", RateLimitMiddleware(RateLimitSuggestions), HandleSuggestions)

	Globals.WebServer.GET("/API/Auth/Login", RateLimitMiddleware(RateLimitAuthLogin), HandleAuthLogin)
	Globals.WebServer.GET("/API/Auth/Callback", RateLimitMiddleware(RateLimitAuthCallback), HandleAuthCallback)
	Globals.WebServer.GET("/API/Auth/Me", RateLimitMiddleware(RateLimitAuthMe), HandleAuthMe)
	Globals.WebServer.POST("/API/Auth/Logout", RateLimitMiddleware(RateLimitAuthMe), HandleAuthLogout)

}

func HandleQueuePage(Context *gin.Context) {

	Context.File("./Web/dist/index.html")

}
