package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Connect(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInGuild.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInGuild.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild := Structs.GetGuild(GuildID, true)

	ErrorConnecting := Guild.Connect(*VoiceState.ChannelID, Event.Channel().ID())

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.FailedToConnect.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.FailedToConnect.Description", Locale, ErrorConnecting.Error()),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild.StartInactivityTimer()

	Channel, RestError := Event.Client().Rest.GetChannel(*VoiceState.ChannelID)

	ChannelName := Localizations.Get("Commands.Connect.DefaultChannelName", Locale)

	if RestError == nil && Channel != nil {

		ChannelName = Channel.Name()

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Connect.Success.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
			Description: Localizations.GetFormat("Commands.Connect.Success.Description", Locale, ChannelName),
			Color:       Utils.PRIMARY,

		})},

	})

}
