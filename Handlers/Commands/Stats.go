package Commands

import (
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"runtime"

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
	EmbedBuilder.SetColor(0xFFFFFF)

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

	// Field 2: Data Streamed

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Streamer := Guild.Queue.PlaybackSession.Streamer
		BytesStreamed := Streamer.BytesStreamed

		// Convert bytes to KB/MB for display

		ProgressValue := ""

		if BytesStreamed > 1024*1024 {

			ProgressValue = fmt.Sprintf("%.2f MB", float64(BytesStreamed)/(1024*1024))
			
		} else {

			ProgressValue = fmt.Sprintf("%.2f KB", float64(BytesStreamed)/1024)
		}

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Progress", Locale), ProgressValue, true)

	} else {

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.Progress", Locale), Localizations.Get("Commands.Stats.Progress.NoData", Locale), true)

	}

	// Field 3: Goroutines

	GoroutinesValue := fmt.Sprintf("%d", runtime.NumGoroutine())

	EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.ActiveGoroutines", Locale), GoroutinesValue, true)

	// Field 4: Current Bitrate

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Streamer := Guild.Queue.PlaybackSession.Streamer
		ProgressMS := Streamer.Progress

		if ProgressMS > 0 {

			Bitrate := (Streamer.BytesStreamed * 8 * 1000) / ProgressMS
			BitrateValue := fmt.Sprintf("%d kb/s", Bitrate)

			EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.CurrentBitrate", Locale), BitrateValue, true)

		}

	}

	// Field 5: Frame Buffer

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		BufferSize := len(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)
		BufferCapacity := cap(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)

		BufferValue := fmt.Sprintf("%d / %d", BufferSize, BufferCapacity)

		EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.FrameBuffer", Locale), BufferValue, true)

	}

	// Field 6: Memory Usage

	var MemStats runtime.MemStats
	runtime.ReadMemStats(&MemStats)
	MemoryValue := fmt.Sprintf("%.2f MB", float64(MemStats.Alloc)/(1024*1024))

	EmbedBuilder.AddField(Localizations.Get("Commands.Stats.Fields.MemoryUsage", Locale), MemoryValue, true)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{EmbedBuilder.Build()},

	})

}