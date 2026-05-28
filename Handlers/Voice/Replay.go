package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Replay(GuildID, UserID snowflake.ID, Args string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Position := ParseReplayPosition(Args)

	if Position < 0 || Position >= len(Guild.Queue.Previous) {

		notifyLocalized(Guild, "Commands.Replay.Error.InvalidPosition.Title", "Commands.Replay.Error.InvalidPosition.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "That song isn't in your history.")

		return

	}

	ReplayIndex := len(Guild.Queue.Previous) - 1 - Position

	if !Guild.Queue.Replay(ReplayIndex) {

		notifyLocalized(Guild, "Commands.Replay.Error.InvalidPosition.Title", "Commands.Replay.Error.InvalidPosition.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "That song isn't in your history.")

		return

	}

	notifyCurrentSongWithMember(Guild, UserID)

}
