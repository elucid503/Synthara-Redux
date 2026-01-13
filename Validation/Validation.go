package Validation

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// GuildSessionError returns error embed if guild session doesn't exist
func GuildSessionError(Locale string) discord.Embed {

	return discord.Embed{
		
		Title:       Localizations.Get("Commands.Pause.Error.Title", Locale),
		Author:      &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
		Description: Localizations.Get("Commands.Pause.Error.Description", Locale),
		Color:       0xFFB3BA,

	}

}

// VoiceStateError returns error embed if user is not in voice channel
func VoiceStateError(GuildID snowflake.ID, UserID snowflake.ID, Locale string) *discord.Embed {

	VoiceState, VoiceStateExists := Globals.DiscordClient.Caches.VoiceState(GuildID, UserID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		RestVoiceState, RestError := Globals.DiscordClient.Rest.GetUserVoiceState(GuildID, UserID)

		if RestError != nil || RestVoiceState == nil || RestVoiceState.ChannelID == nil {

			return &discord.Embed{
				
				Title:       Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Title", Locale),
				Author:      &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
				Description: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Description", Locale),
				Color:       0xFFB3BA,

			}

		}

	}

	return nil

}

// PlaybackError returns error embed if there is no active playback
func PlaybackError(Locale string) discord.Embed {

	return discord.Embed{
		
		Title:       Localizations.Get("Commands.Lyrics.Error.NoSong.Title", Locale),
		Author:      &discord.EmbedAuthor{Name: Localizations.Get("Embeds.Categories.Error", Locale)},
		Description: Localizations.Get("Commands.Lyrics.Error.NoSong.Description", Locale),
		Color:       0xFFB3BA,

	}

}
