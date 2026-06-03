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

		notifyLocalized(Guild, "Embeds.Errors.NoActiveSession.Title", "Embeds.Errors.NoActiveSession.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "There's nothing to resume.")

		return

	}

	Guild.Queue.SetState(Structs.StatePlaying)

	notifyLocalizedWithMember(Guild, UserID, "Commands.Resume.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)

}
