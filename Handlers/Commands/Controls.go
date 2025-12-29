package Commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func ControlsCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Details", os.Getenv("DOMAIN"), GuildID.String()) 

	Event.CreateMessage(discord.MessageCreate{

		Content: fmt.Sprintf("Get controls [here](%s)", Page),

	})

}