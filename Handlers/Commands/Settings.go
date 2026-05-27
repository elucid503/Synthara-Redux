package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

const SettingsCategoryVoiceCommandOptOut = "voice_command_opt_out"

func Settings(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	Data := Event.SlashCommandInteractionData()

	Category := Data.String("category")
	Value := Data.Bool("value")

	UserData, ErrorGettingUser := Structs.GetUser(Event.User().ID.String())

	if ErrorGettingUser != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Settings.Error.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Settings.Error.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	switch Category {

	case SettingsCategoryVoiceCommandOptOut:

		ErrorSaving := UserData.SetVoiceCommandOptOut(Value)

		if ErrorSaving != nil {

			Event.CreateMessage(discord.MessageCreate{

				Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Commands.Settings.Error.Title", Locale),
					Author:      Localizations.Get("Embeds.Categories.Error", Locale),
					Description: Localizations.Get("Commands.Settings.Error.Description", Locale),
					Color:       Utils.ERROR,

				})},

				Flags: discord.MessageFlagEphemeral,

			})

			return

		}

		var TitleKey, DescriptionKey string

		if Value {

			TitleKey = "Commands.Settings.VoiceCommandOptOut.Enabled.Title"
			DescriptionKey = "Commands.Settings.VoiceCommandOptOut.Enabled.Description"

		} else {

			TitleKey = "Commands.Settings.VoiceCommandOptOut.Disabled.Title"
			DescriptionKey = "Commands.Settings.VoiceCommandOptOut.Disabled.Description"

		}

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get(TitleKey, Locale),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
				Description: Localizations.Get(DescriptionKey, Locale),

			})},

			Flags: discord.MessageFlagEphemeral,

		})

	default:

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Settings.Error.UnknownCategory.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Settings.Error.UnknownCategory.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

	}

}
