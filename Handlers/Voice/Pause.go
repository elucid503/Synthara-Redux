package Voice

import (
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Pause(GuildID, UserID snowflake.ID, _ string) {

	Guild, _ := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	Guild.ResetInactivityTimer()

	if Guild.Queue.State != Structs.StatePlaying {

		notifyLocalized(Guild,

			"Commands.Pause.Error.Title",
			"Commands.Pause.Error.Description",
			"Embeds.Categories.Error",
			Utils.ERROR,
		)

		return

	}

	Guild.Queue.SetState(Structs.StatePaused)

	notifyLocalized(Guild,

		"Commands.Pause.Title",
		"Commands.Pause.Description",
		"Embeds.Categories.Playback",
		Utils.PRIMARY,
	)

}
