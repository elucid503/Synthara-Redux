package Voice

import (
	"fmt"
	"strings"

	"Synthara-Redux/APIs"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Play(GuildID, UserID snowflake.ID, Args string) {

	Args = strings.TrimSpace(Args)

	Guild, Locale := guildAndLocale(GuildID)

	if Guild == nil {

		return

	}

	Guild.ResetInactivityTimer()

	if Args == "" {

		notifyLocalized(Guild, "Commands.Play.Error.NoQuery.Title", "Commands.Play.Error.NoQuery.Description", "Embeds.Categories.Error", Utils.ERROR)

		return

	}

	URI, ErrRoute := APIs.Route(Args)

	if ErrRoute != nil {

		notify(Guild, Localizations.Get("Commands.Play.Error.InvalidInput.Title", Locale), Localizations.GetFormat("Commands.Play.Error.InvalidInput.Description", Locale, ErrRoute.Error()), Localizations.Get("Embeds.Categories.Error", Locale), Utils.ERROR)

		return

	}

	Mention := fmt.Sprintf("<@%s>", UserID)

	Song, Pos, ErrHandle := Guild.HandleURI(URI, Mention)

	if ErrHandle != nil {

		notify(Guild, Localizations.Get("Commands.Play.Error.FailedToHandle.Title", Locale), Localizations.GetFormat("Commands.Play.Error.FailedToHandle.Description", Locale, ErrHandle.Error()), Localizations.Get("Embeds.Categories.Error", Locale), Utils.ERROR)

		return

	}

	notifyNowPlaying(Guild, Song, Pos, Locale)

}
