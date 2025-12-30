package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Seek(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// Get the offset from command options

	Data := Event.SlashCommandInteractionData()
	Offset := Data.Int("offset")

	if Offset == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NoOffset", Locale),

		})

		return

	}

	// Check if user is in a guild

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NotInGuild", Locale),

		})

		return

	}

	GuildID := *Event.GuildID()

	// Get the guild

	Guild := Structs.GetGuild(GuildID, false) // does not create if not found

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NoSession", Locale),

		})

		return

	}

	// Check if there's an active playback session

	if Guild.Queue.PlaybackSession == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NoSongPlaying", Locale),

		})

		return

	}

	// Check if user is in the same voice channel

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NotInVoiceChannel", Locale),

		})

		return

	}

	if VoiceState.ChannelID == nil || *VoiceState.ChannelID != Guild.Channels.Voice {

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.Get("Commands.Seek.Errors.NotInSameChannel", Locale),

		})

		return

	}

	// Perform the seek

	ErrorSeeking := Guild.Queue.PlaybackSession.Seek(int(Offset))

	if ErrorSeeking != nil {

		Utils.Logger.Error(fmt.Sprintf("Error seeking: %s", ErrorSeeking.Error()))

		Event.CreateMessage(discord.MessageCreate{

			Content: Localizations.GetFormat("Commands.Seek.Errors.SeekFailed", Locale, ErrorSeeking.Error()),

		})

		return

	}

	// Send success message

	Direction := Localizations.Get("Commands.Seek.Directions.Forward", Locale)
	AbsOffset := Offset

	if Offset < 0 {

		Direction = Localizations.Get("Commands.Seek.Directions.Backward", Locale)
		AbsOffset = -Offset

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: Localizations.GetFormat("Commands.Seek.Success", Locale, Direction, AbsOffset),

	})

}