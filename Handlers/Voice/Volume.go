package Voice

import (
	"fmt"

	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Volume(GuildID, UserID snowflake.ID, Args string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Level, OK := ParseVolumeLevel(Args, Guild.Features.Volume)

	if !OK {

		voiceRespond(GuildID, fmt.Sprintf("Volume is %d percent.", Guild.Features.Volume))
		return

	}

	Guild.SetVolume(Level)

	notifyLocalizedWithMember(Guild, UserID, "Commands.Volume.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)
	voiceRespond(GuildID, fmt.Sprintf("Volume set to %d percent.", Guild.Features.Volume))

}
