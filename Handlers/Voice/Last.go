package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Last(GuildID, UserID snowflake.ID, _ string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	if !Guild.Queue.Last(false) {

		notifyLocalized(Guild, "Commands.Last.Error.Title", "Commands.Last.Error.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "There's no previous song.")

		return

	}

	notifyCurrentSongWithMember(Guild, UserID)

}
