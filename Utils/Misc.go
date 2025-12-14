package Utils

import (
	"Synthara-Redux/Globals"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func Hang() {

	SignalChan := make(chan os.Signal, 1)

	signal.Notify(SignalChan, syscall.SIGINT, syscall.SIGTERM) // Listens for termination signals

	<-SignalChan // Blocks until a signal is received

	Logger.Info("Shutting down and disconnecting Discord client...")

	Globals.DiscordClient.Close(context.TODO())
	
}

func GetNestedValue(Data interface{}, Keys ...string) (interface{}, bool) {

	Current := Data

	for _, Key := range Keys {

		if m, KeyValid := Current.(map[string]interface{}); KeyValid {

			Current, KeyValid = m[Key]

			if !KeyValid {

				return nil, false

			}

		} else {

			return nil, false

		}

	}

	return Current, true

}

func ExtractSongThumbnail(Renderer map[string]interface{}) string {

	ThumbnailsVal, ThumbnailsValExists := GetNestedValue(Renderer, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")

	if !ThumbnailsValExists || len(ThumbnailsVal.([]interface{})) == 0 {

		return "https://cdn.discordapp.com/embed/avatars/1.png" // Default 'thumbnail'

	}

	LastThumbnail, LastThumbnailExists := ThumbnailsVal.([]interface{})[len(ThumbnailsVal.([]interface{}))-1].(map[string]interface{})

	if !LastThumbnailExists {

		return "https://cdn.discordapp.com/embed/avatars/1.png"

	}

	URL, LastThumbnailURLExists := LastThumbnail["url"].(string)

	if !LastThumbnailURLExists {

		return "https://cdn.discordapp.com/embed/avatars/1.png"

	}

	return URL

}

func ParseFormattedDuration(FormattedDuration string) int {

	if FormattedDuration == "" {

		return 0
		
	}

	var Minutes, Seconds int

	if _, ParseError := fmt.Sscanf(FormattedDuration, "%d:%d", &Minutes, &Seconds); ParseError == nil {

		return Minutes*60 + Seconds

	}

	return 0

}