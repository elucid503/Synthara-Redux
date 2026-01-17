package Components

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Pause(Event *events.ComponentInteractionCreate) {

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

	Guild.Queue.SetState(Structs.StatePaused)

	// Update the original message with the new pause/play button
	if Guild.Queue.Current != nil {

		State := Tidal.QueueInfo{

			Playing: false, // Now paused

			GuildID: GuildID,

			SongPosition: 0,

			TotalPrevious: len(Guild.Queue.Previous),
			TotalUpcoming: len(Guild.Queue.Upcoming),

			Locale: Locale,

		}

		Event.UpdateMessage(discord.NewMessageUpdateBuilder().
			AddEmbeds(Guild.Queue.Current.Embed(State)).
			AddActionRow(Guild.Queue.Current.Buttons(State)...).
			Build())

	} else {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Pause.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Playback", Locale),
				Description: Localizations.Get("Commands.Pause.Description", Locale),

			})},

			Flags: discord.MessageFlagEphemeral,

		})

	}

}
