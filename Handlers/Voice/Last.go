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

	if !Guild.Queue.Last() {

		notifyLocalized(Guild, "Commands.Last.Error.Title", "Commands.Last.Error.Description", "Embeds.Categories.Error", Utils.ERROR)

		return

	}

	notifyCurrentSongWithMember(Guild, UserID)

}
