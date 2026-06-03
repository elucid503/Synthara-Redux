package Validation

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// GuildSessionError returns error embed if guild session doesn't exist
func GuildSessionError(Locale string) discord.Embed {

	return discord.Embed{

		Title: Localizations.Get("Embeds.Errors.NoActiveSession.Title", Locale),
		Author: &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
		Description: Localizations.Get("Embeds.Errors.NoActiveSession.Description", Locale),

		Color: Utils.ERROR,

	}

}

// VoiceStateError returns error embed if user is not in voice channel
func VoiceStateError(GuildID snowflake.ID, UserID snowflake.ID, Locale string) *discord.Embed {

	VoiceState, VoiceStateExists := Globals.DiscordClient.Caches.VoiceState(GuildID, UserID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		RestVoiceState, RestError := Globals.DiscordClient.Rest.GetUserVoiceState(GuildID, UserID)

		if RestError != nil || RestVoiceState == nil || RestVoiceState.ChannelID == nil {

			return &discord.Embed{

				Title: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Title", Locale),
				Author: &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
				Description: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Description", Locale),

				Color: Utils.ERROR,

			}

		}

	}

	return nil

}

// PlaybackError returns error embed if there is no active playback
func PlaybackError(Locale string) discord.Embed {

	return discord.Embed{

		Title: Localizations.Get("Embeds.Errors.NoActiveSession.Title", Locale),
		Author: &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
		Description: Localizations.Get("Embeds.Errors.NoActiveSession.Description", Locale),

		Color: Utils.ERROR,

	}

}
