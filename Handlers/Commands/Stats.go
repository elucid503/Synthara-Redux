package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Stats(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	Guild := Structs.GetGuild(*Event.GuildID(), false)

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Lock.Error.Title", Locale),
				Description: Localizations.Get("Commands.Lock.Error.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}
	
	EmbedBuilder := discord.NewEmbedBuilder()
	EmbedBuilder.SetTitle(Localizations.Get("Commands.Stats.Title", Locale))
	EmbedBuilder.SetColor(0x5865F2)

	// Field 1: Playback Status

	var StatusValue string

	switch Guild.Queue.State {

	case Structs.StatePlaying:

		StatusValue = Localizations.Get("Commands.Stats.Status.Playing", Locale)

	case Structs.StatePaused:

		StatusValue = Localizations.Get("Commands.Stats.Status.Paused", Locale)

	default:

		StatusValue = Localizations.Get("Commands.Stats.Status.Idle", Locale)

	}

	EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Status", Locale), StatusValue, true)

	// Field 2: Segment Progress

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Streamer := Guild.Queue.PlaybackSession.Streamer
		CurrentSegment := Streamer.CurrentIndex
		TotalSegments := Streamer.TotalSegments

		Percentage := 0
		
		if TotalSegments > 0 {

			Percentage = (CurrentSegment * 100) / TotalSegments

		}

		ProgressValue := fmt.Sprintf(Localizations.Get("Commands.Stats.Progress.Format", Locale), CurrentSegment, TotalSegments, Percentage)

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Progress", Locale), ProgressValue, true)

	} else {

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Progress", Locale), Localizations.Get("Commands.Stats.Progress.NoData", Locale), true)

	}

	// Field 3: Queue Status

	UpcomingCount := len(Guild.Queue.Upcoming)
	PreviousCount := len(Guild.Queue.Previous)

	var QueueValue string

	if UpcomingCount == 1 {

		QueueValue = fmt.Sprintf(Localizations.Get("Commands.Stats.Queue.Singular", Locale), UpcomingCount)

	} else {

		QueueValue = fmt.Sprintf(Localizations.Get("Commands.Stats.Queue.Plural", Locale), UpcomingCount)

	}

	if PreviousCount > 0 {

		if PreviousCount == 1 {

			QueueValue += "\n" + fmt.Sprintf(Localizations.Get("Commands.Stats.Queue.PreviousSingular", Locale), PreviousCount)

		} else {

			QueueValue += "\n" + fmt.Sprintf(Localizations.Get("Commands.Stats.Queue.PreviousPlural", Locale), PreviousCount)
		}
	}

	EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Queue", Locale), QueueValue, true)

	// Field 4: Buffer Status

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		BufferSize := len(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)
		BufferCapacity := cap(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)
		BufferPercent := (BufferSize * 100) / BufferCapacity

		var BufferValue string

		if BufferSize == 1 {

			BufferValue = fmt.Sprintf(Localizations.Get("Commands.Stats.Buffer.Singular", Locale), BufferSize, BufferCapacity, BufferPercent)

		} else {

			BufferValue = fmt.Sprintf(Localizations.Get("Commands.Stats.Buffer.Plural", Locale), BufferSize, BufferCapacity, BufferPercent)

		}

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Buffer", Locale), BufferValue, true)

	} else {

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Buffer", Locale), Localizations.Get("Commands.Stats.Buffer.NoData", Locale), true)

	}

	// Field 5: Playback Duration

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		ProgressMS := Guild.Queue.PlaybackSession.Streamer.Progress
		TotalSeconds := ProgressMS / 1000
		Minutes := TotalSeconds / 60
		Seconds := TotalSeconds % 60

		DurationValue := fmt.Sprintf(Localizations.Get("Commands.Stats.Duration.Format", Locale), Minutes, Seconds)

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Duration", Locale), DurationValue, true)

	} else {

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Duration", Locale), Localizations.Get("Commands.Stats.Duration.NoData", Locale), true)
	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{EmbedBuilder.Build()},

	})

}