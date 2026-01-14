package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
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
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Get the user's voice state to find which channel they're in
	VoiceState, VoiceStateExists := Event.Client().Caches.VoiceState(*Event.GuildID(), Event.User().ID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.VoiceChannel.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild.Channels.Voice = *VoiceState.ChannelID
	Guild.Channels.Text = Event.Channel().ID()

	ErrorConnecting := Event.Client().UpdateVoiceState(context.Background(), *Event.GuildID(), VoiceState.ChannelID, false, false)

	if ErrorConnecting != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Embeds.Categories.Error", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: "Failed to connect to voice channel.",
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Embeds.Categories.Playback", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.Get("Embeds.Notifications.Reconnect.Description", Locale),

		})).
		Build())

}
