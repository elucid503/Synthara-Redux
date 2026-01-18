package Commands

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Restrict(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// This command is developer-only

	DeveloperIDs := os.Getenv("DEVELOPERS")
	UserID := Event.User().ID.String()

	if !strings.Contains(DeveloperIDs, UserID) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Restrict.Unauthorized.Title", Locale),
				Description: Localizations.Get("Commands.Restrict.Unauthorized.Description", Locale),
				Color:       Utils.RED,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Data := Event.SlashCommandInteractionData()

	Enabled := Data.Bool("enabled")
	CustomMessage, HasCustomMessage := Data.OptString("message")

	Globals.ServiceRestrictionMutex.Lock()
	Globals.ServiceRestricted = Enabled
	
	if HasCustomMessage && CustomMessage != "" {
		
		Globals.ServiceRestrictionMessage = CustomMessage
		
	} else {
		
		Globals.ServiceRestrictionMessage = ""
		
	}
	Globals.ServiceRestrictionMutex.Unlock()

	var ResponseTitle, ResponseDescription string

	if Enabled {

		ResponseTitle = Localizations.Get("Commands.Restrict.Enabled.Title", Locale)
		ResponseDescription = Localizations.Get("Commands.Restrict.Enabled.Description", Locale)

	} else {

		ResponseTitle = Localizations.Get("Commands.Restrict.Disabled.Title", Locale)
		ResponseDescription = Localizations.Get("Commands.Restrict.Disabled.Description", Locale)

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       ResponseTitle,
			Description: ResponseDescription,
			Color:       Utils.WHITE,

		})},

	})

}
