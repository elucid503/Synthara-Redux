package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"os"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Notify(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// This command is developer-only

	DeveloperIDs := os.Getenv("DEVELOPERS")
	UserID := Event.User().ID.String()

	if !strings.Contains(DeveloperIDs, UserID) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.CreateNotification.Unauthorized.Title", Locale),
				Description: Localizations.Get("Commands.CreateNotification.Unauthorized.Description", Locale),
				Color:       0xFFB3BA,

			})},

		})

		return

	}

	Data := Event.SlashCommandInteractionData()

	Title := Data.String("title")
	Description := Data.String("description")

	// Replaces literal escape sequences with actual newlines so Discord preserves line breaks in embeds.
	
	Description = strings.ReplaceAll(Description, "\\r\\n", "\n")
	Description = strings.ReplaceAll(Description, "\\n", "\n")

	var Expiry time.Time
	var HasExpiry bool

	ExpiryString, ExpiryExists := Data.OptString("expiry")

	if ExpiryExists && ExpiryString != "" {

		// Parses the date string as RFC3339 format (ex: 2006-01-02T15:04:05Z)

		ParsedExpiry, ParseError := time.Parse(time.RFC3339, ExpiryString)

		if ParseError != nil {

			// Tries also an alternative format (just date: 2006-01-02) for better UX

			ParsedExpiry, ParseError = time.Parse("2006-01-02", ExpiryString)

			if ParseError != nil {

				Event.CreateMessage(discord.MessageCreate{

					Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Commands.CreateNotification.InvalidDate.Title", Locale),
						Description: Localizations.Get("Commands.CreateNotification.InvalidDate.Description", Locale),
						Color:       0xFFB3BA,

					})},

				})

				return

			}

		}

		Expiry = ParsedExpiry
		HasExpiry = true

	}

	var Notification *Structs.Notification
	var CreateError error

	if HasExpiry {

		Notification, CreateError = Structs.CreateNotification(Title, Description, Expiry)

	} else {

		Notification, CreateError = Structs.CreateNotification(Title, Description, time.Time{})

	}

	if CreateError != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.CreateNotification.Error.Title", Locale),
				Description: Localizations.Get("Commands.CreateNotification.Error.Description", Locale),
				Color:       0xFFB3BA, // Red for error

			})},

		})

		return

	}

	SuccessDescription := Localizations.Get("Commands.CreateNotification.Success.Description", Locale)
	
	if HasExpiry {

		SuccessDescription += "\n" + Localizations.Get("Commands.CreateNotification.Success.Expiry", Locale) + " " + Expiry.Format("2006-01-02 15:04:05")

	}

	SuccessDescription += "\n" + Localizations.Get("Commands.CreateNotification.Success.ID", Locale) + " `" + Notification.ID + "`"

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.CreateNotification.Success.Title", Locale),
			Description: SuccessDescription,
			Color:       0xFFFFFF,

		})},

	})

}