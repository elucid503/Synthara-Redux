package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Reconnect(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, true)

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Failed to create guild session.",
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Get the user's voice state to find which channel they're in
	VoiceState, VoiceStateExists := Event.Client().Caches.VoiceState(*Event.GuildID(), Event.User().ID)

	var ChannelID *snowflake.ID

	if VoiceStateExists && VoiceState.ChannelID != nil {

		ChannelID = VoiceState.ChannelID

	} else {

		RestVoiceState, RestError := Event.Client().Rest.GetUserVoiceState(*Event.GuildID(), Event.User().ID)

		if RestError != nil || RestVoiceState == nil || RestVoiceState.ChannelID == nil {

			Event.CreateMessage(discord.MessageCreate{

				Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

					Title:       Localizations.Get("Embeds.Categories.Error", Locale),
					Author:      Localizations.Get("Embeds.Categories.Error", Locale),
					Description: Localizations.Get("Commands.Play.Error.VoiceChannel.Description", Locale),
					Color:       Utils.ERROR,

				})},

				Flags: discord.MessageFlagEphemeral,

			})

			return
		}

		ChannelID = RestVoiceState.ChannelID

	}

	Guild.Channels.Voice = *ChannelID
	Guild.Channels.Text = Event.Channel().ID()

	ErrorConnecting := Event.Client().UpdateVoiceState(context.Background(), *Event.GuildID(), ChannelID, false, false)

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Failed to connect to voice channel.",
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Channel, RestError := Event.Client().Rest.GetChannel(*ChannelID)

	var ChannelName string

	if RestError == nil && Channel != nil {

		ChannelName = Channel.Name()

	} else {

		ChannelName = "voice channel"

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

		Title:       Localizations.Get("Embeds.Notifications.Reconnect.Title", Locale),
		Author:      Localizations.Get("Embeds.Categories.Notifications", Locale),
		Description: Localizations.GetFormat("Embeds.Notifications.Reconnect.Description", Locale, ChannelName),

	})).Build())

}