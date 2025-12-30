package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func QueueCommand(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := Event.GuildID()

	Page := fmt.Sprintf("%s/Queues/%s?View=Queue", os.Getenv("DOMAIN"), GuildID.String()) 

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.GetFormat("Commands.Queue.Success", Locale, Page),

	})

}