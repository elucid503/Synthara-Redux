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

		notifyLocalized(Guild, "Embeds.Errors.NoActiveSession.Title", "Embeds.Errors.NoActiveSession.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "Playback is already paused.")

		return

	}

	Guild.Queue.SetState(Structs.StatePaused)

	notifyLocalizedWithMember(Guild, UserID, "Commands.Pause.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)

}
