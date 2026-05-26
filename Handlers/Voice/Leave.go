package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Leave(GuildID, UserID snowflake.ID, _ string) {

	Guild, _ := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	Guild.ResetInactivityTimer()

	Guild.Cleanup(true)

	notifyLocalizedWithMember(Guild, UserID, "Commands.Leave.Success.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Notifications", Utils.PRIMARY)

}
