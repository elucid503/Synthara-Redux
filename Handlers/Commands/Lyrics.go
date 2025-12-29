package Commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func LyricsCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Lyrics", os.Getenv("DOMAIN"), GuildID.String()) 

	Event.CreateMessage(discord.MessageCreate{

		Content: fmt.Sprintf("View the lyrics [here](%s)", Page),

	})

}