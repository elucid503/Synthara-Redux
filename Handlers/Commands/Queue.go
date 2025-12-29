package Commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func QueueCommand(Event *events.ApplicationCommandInteractionCreate) {

	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Queue", os.Getenv("DOMAIN"), GuildID.String()) 

	Event.CreateMessage(discord.MessageCreate{

		Content: fmt.Sprintf("View/Edit the Queue [here](%s)", Page),

	})

}