package Voice

import (
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Repeat(GuildID, UserID snowflake.ID, Args string) {

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	if !requireVoice(Guild, GuildID, UserID, Locale) {

		return

	}

	Guild.ResetInactivityTimer()

	Mode := ParseRepeatMode(Args, Guild.Features.Repeat)

	Guild.Features.Repeat = Mode

	var TitleKey string
	var TTSResponse string

	switch Mode {

	case Structs.RepeatOne:

		TitleKey = "Commands.Repeat.One.Title"
		TTSResponse = "Repeating this song."

	case Structs.RepeatAll:

		TitleKey = "Commands.Repeat.All.Title"
		TTSResponse = "Repeating the queue."

	default:

		TitleKey = "Commands.Repeat.Off.Title"
		TTSResponse = "Repeat is off."
		Guild.Features.Repeat = Structs.RepeatOff

	}

	notifyLocalizedWithMember(Guild, UserID, TitleKey, "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)
	voiceRespond(GuildID, TTSResponse)

}
