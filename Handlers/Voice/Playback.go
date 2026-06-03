package Voice

import (
	"fmt"

	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

// The Speed and Reverb commands are very similar, so they both use the same helper function to avoid code duplication.

func Speed(guildID, userID snowflake.ID, args string) {

	runVoiceIntSetting(guildID, userID, args, ParseSpeedMilli, (*Structs.Guild).SetSpeed, func(g *Structs.Guild) int { return g.Features.SpeedMilli }, "Commands.Speed.Title",

		func(g *Structs.Guild) string {

			return fmt.Sprintf("Playback speed is %s.", Structs.FormatSpeedLabel(g.Features.SpeedMilli))

		},

		func(g *Structs.Guild) string {

			return fmt.Sprintf("Playback speed set to %s.", Structs.FormatSpeedLabel(g.Features.SpeedMilli))

		},

	)

}

func Reverb(guildID, userID snowflake.ID, args string) {

	runVoiceIntSetting(guildID, userID, args, ParseReverbPercent, (*Structs.Guild).SetReverb, func(g *Structs.Guild) int { return g.Features.Reverb }, "Commands.Reverb.Title",

		func(g *Structs.Guild) string {

			return fmt.Sprintf("Reverb is %d percent.", g.Features.Reverb)

		},

		func(g *Structs.Guild) string {

			return fmt.Sprintf("Reverb set to %d percent.", g.Features.Reverb)

		},

	)
}

func runVoiceIntSetting(guildID, userID snowflake.ID, args string, parse func(string, int) (int, bool), apply func(*Structs.Guild, int) int, current func(*Structs.Guild) int, titleKey string, statusLine, confirmLine func(*Structs.Guild) string) {

	guild, locale := guildAndLocale(guildID)

	if guild == nil || !requireVoice(guild, guildID, userID, locale) {

		return

	}

	guild.ResetInactivityTimer()

	level, ok := parse(args, current(guild))

	if !ok {

		voiceRespond(guildID, statusLine(guild))
		return

	}

	apply(guild, level)
	notifyLocalizedWithMember(guild, userID, titleKey, "Embeds.NowPlaying.AddedByMemberViaVoice", "Embeds.Categories.Playback", Utils.PRIMARY)

	voiceRespond(guildID, confirmLine(guild))

}
