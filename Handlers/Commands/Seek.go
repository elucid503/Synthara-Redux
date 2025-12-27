package Commands

import (
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func SeekCommand(Event *events.ApplicationCommandInteractionCreate) {

	// Get the offset from command options

	Data := Event.SlashCommandInteractionData()
	Offset := Data.Int("offset")

	if Offset == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Content: "Please provide a seek offset in seconds (e.g., 10 to skip forward, -10 to go back)!",

		})

		return

	}

	// Check if user is in a guild

	if Event.Member() == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a guild to use this command!",

		})

		return

	}

	GuildID := *Event.GuildID()

	// Get the guild

	Guild := Structs.GetGuild(GuildID)

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No active playback session found!",

		})

		return

	}

	// Check if there's an active playback session

	if Guild.Queue.PlaybackSession == nil {

		Event.CreateMessage(discord.MessageCreate{

			Content: "No song is currently playing!",

		})

		return

	}

	// Check if user is in the same voice channel

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in a voice channel to use this command!",

		})

		return

	}

	if VoiceState.ChannelID == nil || *VoiceState.ChannelID != Guild.Channels.Voice {

		Event.CreateMessage(discord.MessageCreate{

			Content: "You must be in the same voice channel as the bot!",

		})

		return

	}

	// Perform the seek

	ErrorSeeking := Guild.Queue.PlaybackSession.Seek(int(Offset))

	if ErrorSeeking != nil {

		Utils.Logger.Error(fmt.Sprintf("Error seeking: %s", ErrorSeeking.Error()))

		Event.CreateMessage(discord.MessageCreate{

			Content: fmt.Sprintf("Failed to seek: %s", ErrorSeeking.Error()),

		})

		return

	}

	// Send success message

	Direction := "forward"
	AbsOffset := Offset

	if Offset < 0 {

		Direction = "backward"
		AbsOffset = -Offset

	}

	Event.CreateMessage(discord.MessageCreate{

		Content: fmt.Sprintf("Seeked %s by %d seconds!", Direction, AbsOffset),

	})

}
