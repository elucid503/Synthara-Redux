package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Shuffle(GuildID, UserID snowflake.ID, Args string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Enabled := ParseShuffleEnabled(Args, Guild.Features.Shuffle)

	Guild.Features.Shuffle = Enabled

	if Enabled {

		notifyLocalizedWithMember(Guild, UserID, "Commands.Shuffle.Enabled.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)

		return

	}

	notifyLocalizedWithMember(Guild, UserID, "Commands.Shuffle.Disabled.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)

}
