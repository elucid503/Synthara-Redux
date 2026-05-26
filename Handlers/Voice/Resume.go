package Voice

import (
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Resume(GuildID, UserID snowflake.ID, _ string) {

	Guild, _ := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	Guild.ResetInactivityTimer()

	if Guild.Queue.State != Structs.StatePaused {

		notifyLocalized(Guild,

			"Commands.Resume.Error.Title",
			"Commands.Resume.Error.Description",
			"Embeds.Categories.Error",
			Utils.ERROR,
		)

		return

	}

	Guild.Queue.SetState(Structs.StatePlaying)

	notifyLocalized(Guild,

		"Commands.Resume.Title",
		"Commands.Resume.Description",
		"Embeds.Categories.Playback",
		Utils.PRIMARY,
	)

}
