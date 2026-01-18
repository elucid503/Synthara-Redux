package Components

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Autoplay(Event *events.ComponentInteractionCreate) {

	Locale := Event.Locale().Code()
	GuildID := *Event.GuildID()

	Guild := Structs.GetGuild(GuildID, false)

	// Validate guild session
	if Guild == nil {

		ErrorEmbed := Validation.GuildSessionError(Locale)
		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	// Validate user is in voice
	if ErrorEmbed := Validation.VoiceStateError(GuildID, Event.User().ID, Locale); ErrorEmbed != nil {

		Event.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{*ErrorEmbed}, Flags: discord.MessageFlagEphemeral})
		return

	}

	// Toggle autoplay
	Guild.Features.Autoplay = !Guild.Features.Autoplay

	var StatusKey string

	if Guild.Features.Autoplay {

		StatusKey = "Commands.AutoPlay.Enabled"

		// Generate initial suggestions when enabling autoplay

		if len(Guild.Queue.Suggestions) == 0 {

			Guild.Queue.RegenerateSuggestions()

		}
		
		// Check if playback should start from suggestions

		if Guild.Queue.Current == nil && len(Guild.Queue.Upcoming) == 0 && len(Guild.Queue.Suggestions) > 0 {

			Utils.Logger.Info("AutoPlay", "Queue is empty, starting playback from suggestions")
			
			// Take first suggestion and start playing
			
			Guild.Queue.Next(true)

		}

	} else {

		StatusKey = "Commands.AutoPlay.Disabled"

	}

	Event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.AutoPlay.Title", Locale),
			Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
			Description: Localizations.Get(StatusKey, Locale),

		})).
		Build())

}
