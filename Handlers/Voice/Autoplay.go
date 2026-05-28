package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Autoplay(GuildID, UserID snowflake.ID, Args string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Enabled := ParseAutoplayEnabled(Args, Guild.Features.Autoplay)

	Guild.Features.Autoplay = Enabled

	if Enabled {

		if len(Guild.Queue.Suggestions) == 0 {

			Guild.Queue.RegenerateSuggestions()

		}

		Guild.StartInactivityTimer()

		if Guild.Queue.Current == nil && len(Guild.Queue.Upcoming) == 0 && len(Guild.Queue.Suggestions) > 0 {

			Guild.Queue.Next(true)

		}

		notifyLocalizedWithMember(Guild, UserID, "Commands.AutoPlay.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)
		voiceRespond(GuildID, "Autoplay is on.")

		return

	}

	Guild.StartInactivityTimer()

	notifyLocalizedWithMember(Guild, UserID, "Commands.AutoPlay.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)
	voiceRespond(GuildID, "Autoplay is off.")

}
