package Commands

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

func Inspect(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// This command is developer-only

	DeveloperIDs := os.Getenv("DEVELOPERS")
	UserID := Event.User().ID.String()

	if !strings.Contains(DeveloperIDs, UserID) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Inspect.Unauthorized.Title", Locale),
				Description: Localizations.Get("Commands.Inspect.Unauthorized.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Data := Event.SlashCommandInteractionData()

	GuildIDString, HasGuildSelection := Data.OptString("guild")

	// If no guild specified, show overview of all active guilds

	if !HasGuildSelection || GuildIDString == "" {

		ShowGuildOverview(Event, Locale)

	} else {

		// Show detailed inspection for specific guild

		ShowGuildDetails(Event, Locale, GuildIDString)

	}

}

func ShowGuildOverview(Event *events.ApplicationCommandInteractionCreate, Locale string) {

	Structs.GuildStoreMutex.Lock()
	ActiveGuilds := make([]*Structs.Guild, 0, len(Structs.GuildStore))
	
	for _, Guild := range Structs.GuildStore {

		ActiveGuilds = append(ActiveGuilds, Guild)

	}
	Structs.GuildStoreMutex.Unlock()

	if len(ActiveGuilds) == 0 {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Inspect.Overview.NoGuilds.Title", Locale),
				Description: Localizations.Get("Commands.Inspect.Overview.NoGuilds.Description", Locale),
				Color:       0xFFFFFF,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	EmbedBuilder := discord.NewEmbedBuilder()

	EmbedBuilder.SetTitle(Localizations.Get("Commands.Inspect.Overview.Title", Locale))
	EmbedBuilder.SetColor(0xFFFFFF)

	// System Stats

	var MemStats runtime.MemStats
	runtime.ReadMemStats(&MemStats)
	
	guildWord := Localizations.Get("Common.Guild", Locale)

	if len(ActiveGuilds) != 1 {

		guildWord = Localizations.Get("Common.Guilds", Locale)

	}
	
	SystemStats := fmt.Sprintf(Localizations.Get("Commands.Inspect.Overview.SystemStats", Locale), len(ActiveGuilds), guildWord, runtime.NumGoroutine(), float64(MemStats.Alloc)/(1024*1024))

	EmbedBuilder.SetDescription(SystemStats)

	// Add field for each active guild (limited to 25 fields max)

	MaxFields := 25
	GuildsToShow := len(ActiveGuilds)
	
	if GuildsToShow > MaxFields {
		
		GuildsToShow = MaxFields
		
	}

	for i := 0; i < GuildsToShow; i++ {

		Guild := ActiveGuilds[i]
		
		// Build status string

		StateLabel := Localizations.Get("Commands.Stats.Status.Idle", Locale)
		
		switch Guild.Queue.State {
			
			case Structs.StatePlaying:
				
				StateLabel = Localizations.Get("Commands.Stats.Status.Playing", Locale)
				
			case Structs.StatePaused:
				
				StateLabel = Localizations.Get("Commands.Stats.Status.Paused", Locale)
				
		}

		// Queue info

		QueueSize := len(Guild.Queue.Upcoming)
		WebSocketCount := len(Guild.Queue.WebSockets)
		
		var CurrentSong string
		
		if Guild.Queue.Current != nil {
			
			CurrentSong = fmt.Sprintf("%s", Guild.Queue.Current.Title)
			
		} else {
			
			CurrentSong = Localizations.Get("Commands.Inspect.Overview.NoSong", Locale)
			
		}

		UpcomingPlural := ""

		if QueueSize != 1 {

			UpcomingPlural = "s"

		}

		clientWord := Localizations.Get("Common.Client", Locale)

		if WebSocketCount != 1 {

			clientWord = Localizations.Get("Common.Clients", Locale)

		}

		FieldValue := fmt.Sprintf( Localizations.Get("Commands.Inspect.Overview.GuildField", Locale), StateLabel, CurrentSong, QueueSize, UpcomingPlural, WebSocketCount, clientWord, )
		FieldName := fmt.Sprintf("%s", Guild.ID.String())

		EmbedBuilder.AddField(FieldName, FieldValue, false)

	}

	// Add note if there are more guilds

	if len(ActiveGuilds) > MaxFields {

		RemainingCount := len(ActiveGuilds) - MaxFields
		
		FooterText := fmt.Sprintf(
			Localizations.Get("Commands.Inspect.Overview.MoreGuilds", Locale),
			RemainingCount,
		)
		
		EmbedBuilder.SetFooter(FooterText, "")

	}

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{EmbedBuilder.Build()},

	})

}

func ShowGuildDetails(Event *events.ApplicationCommandInteractionCreate, Locale string, GuildIDString string) {

	GuildID, ParseError := snowflake.Parse(GuildIDString)

	if ParseError != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Inspect.Details.InvalidGuildID.Title", Locale),
				Description: Localizations.Get("Commands.Inspect.Details.InvalidGuildID.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Structs.GuildStoreMutex.Lock()
	Guild, Exists := Structs.GuildStore[GuildID]
	Structs.GuildStoreMutex.Unlock()

	if !Exists {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Inspect.Details.GuildNotFound.Title", Locale),
				Description: Localizations.Get("Commands.Inspect.Details.GuildNotFound.Description", Locale),
				Color:       0xFFB3BA,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Get guild name via REST API

	GuildName := Guild.ID.String()
	
	RestGuild, RestError := Globals.DiscordClient.Rest.GetGuild(Guild.ID, false)
	
	if RestError == nil {
		
		GuildName = RestGuild.Name
		
	} else {
		
		CachedGuild, ExistsInCache := Globals.DiscordClient.Caches.GuildCache().Get(Guild.ID)
		
		if ExistsInCache {
			
			GuildName = CachedGuild.Name
			
		}
		
	}

	// NOTE: Localizations aren't used here. This is for developers, and won't really be user-facing.	

	EmbedBuilder := discord.NewEmbedBuilder()

	EmbedBuilder.SetTitle(GuildName)
	EmbedBuilder.SetColor(0xFFFFFF) // White

	// Locale Field 

	EmbedBuilder.AddField("Locale", Guild.Locale.Code(), true)

	// Status 

	var Status string

	switch Guild.Queue.State {

		case Structs.StatePlaying:

			Status = "Playing"

		case Structs.StatePaused:

			Status = "Paused"

		default:

			Status = "Idle"

	}

	EmbedBuilder.AddField("Status", Status, true)

	// Current Song

	if Guild.Queue.Current != nil {

		EmbedBuilder.AddField("Current", fmt.Sprintf("%s", Guild.Queue.Current.Title), true)

	} else {

		EmbedBuilder.AddField("Current", "None", true)

	}
	
	// Previous Len 

	EmbedBuilder.AddField("Previous", fmt.Sprintf("%d %s", len(Guild.Queue.Previous), Utils.Pluralize("Song", len(Guild.Queue.Previous))), true)

	// Upcoming Len

	EmbedBuilder.AddField("Upcoming", fmt.Sprintf("%d %s", len(Guild.Queue.Upcoming), Utils.Pluralize("Song", len(Guild.Queue.Upcoming))), true)

	// Suggestions Len 

	EmbedBuilder.AddField("Suggestions", fmt.Sprintf("%d %s", len(Guild.Queue.Suggestions), Utils.Pluralize("Song", len(Guild.Queue.Suggestions))), true)

	// WebSocket Count

	EmbedBuilder.AddField("WebSockets", fmt.Sprintf("%d %s", len(Guild.Queue.WebSockets), Utils.Pluralize("Client", len(Guild.Queue.WebSockets))), true)

	// Data Streamed 

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

		EmbedBuilder.AddField("Data Streamed", ProgressValue, true)

	} else {

		EmbedBuilder.AddField("Data Streamed", "N/A", true)

	}

	// Bitrate 

	var BitrateAdded bool = false

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Streamer := Guild.Queue.PlaybackSession.Streamer
		ProgressMS := Streamer.Progress

		if ProgressMS > 0 {

			Bitrate := (Streamer.BytesStreamed * 8 * 1000) / ProgressMS
			BitrateValue := fmt.Sprintf("%d kb/s", Bitrate)

			EmbedBuilder.AddField("Bitrate", BitrateValue, true)
			BitrateAdded = true

		}

	}

	if !BitrateAdded {
		
		EmbedBuilder.AddField("Bitrate", "N/A", true)

	}

	// Buffer 

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		BufferSize := len(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)
		BufferCapacity := cap(Guild.Queue.PlaybackSession.Streamer.OpusFrameChan)

		BufferValue := fmt.Sprintf("%d / %d", BufferSize, BufferCapacity)

		EmbedBuilder.AddField("Buffer", BufferValue, true)

	} else {

		EmbedBuilder.AddField("Buffer", "N/A", true)

	}

	// Playback Time 

	if Guild.Queue.PlaybackSession != nil && Guild.Queue.PlaybackSession.Streamer != nil {

		Streamer := Guild.Queue.PlaybackSession.Streamer
		ProgressMS := Streamer.Progress

		PlaybackTimeValue := fmt.Sprintf("%s", time.Duration(ProgressMS) * time.Millisecond)

		EmbedBuilder.AddField("Playback Time", PlaybackTimeValue, true)

	} else {

		EmbedBuilder.AddField("Playback Time", "N/A", true)

	}

	// Inactivity Timer 

	Timer := Guild.Internal.InactivityTimer

	if Timer != nil {

		EmbedBuilder.AddField("Inactivity Timer", "Active", true)

	} else {

		EmbedBuilder.AddField("Inactivity Timer", "Inactive", true)

	}

	// Flags 

	var RepeatStatus string

	switch Guild.Features.Repeat {

		case Structs.RepeatOff:
			
			RepeatStatus = "Off"

		case Structs.RepeatOne:

			RepeatStatus = "One"

		case Structs.RepeatAll:

			RepeatStatus = "All"

	}

	EmbedBuilder.AddField("Repeat", RepeatStatus, true)

	// Shuffle

	EmbedBuilder.AddField("Shuffle", BoolToVal(Guild.Features.Shuffle), true)

	// Autoplay

	EmbedBuilder.AddField("Autoplay", BoolToVal(Guild.Features.Autoplay), true)
	
	// Locked 

	EmbedBuilder.AddField("Locked", BoolToVal(Guild.Features.Locked), true)
	
	// Ping 

	EmbedBuilder.AddField("Voice Ping", fmt.Sprintf("%d ms", Guild.VoiceConnection.Gateway().Latency().Milliseconds()), true)

	// Voice Connection Status 

	StatusMap := map[voice.Status]string{

		voice.StatusUnconnected: "Unconnected",
		voice.StatusConnecting: "Connecting",
		voice.StatusWaitingForHello: "WaitingForHello",
		voice.StatusIdentifying: "Identifying",
		voice.StatusResuming: "Resuming",
		voice.StatusWaitingForReady: "WaitingForReady",
		voice.StatusReady: "Ready",
		voice.StatusDisconnected: "Disconnected",

	}

	VoiceStatus, Exists := StatusMap[Guild.VoiceConnection.Gateway().Status()]

	if !Exists {

		VoiceStatus = "Unknown"

	}

	EmbedBuilder.AddField("Voice Status", VoiceStatus, true)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{EmbedBuilder.Build()},

	})
	
}

func BoolToVal(value bool) string {

	if value {
		
		return "Yes"
		
	}
	
	return "No"

}