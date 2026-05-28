package Voice

import (
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Next(GuildID, UserID snowflake.ID, _ string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Advanced, Ended := Guild.Queue.Next(false)

	if Ended {

		notifyLocalizedWithMember(Guild, UserID, "Embeds.Notifications.QueueEnded.Title", "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Notifications", Utils.PRIMARY)
		voiceRespond(GuildID, "The queue has ended.")

		return

	}

	if !Advanced {

		notifyLocalized(Guild, "Commands.Next.Error.NoNextSong.Title", "Commands.Next.Error.NoNextSong.Description", "Embeds.Categories.Error", Utils.ERROR)
		voiceRespond(GuildID, "There's nothing next in the queue.")

		return

	}

	notifyCurrentSongWithMember(Guild, UserID)

}
