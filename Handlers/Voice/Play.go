package Voice

import (
	"fmt"
	"strconv"
	"strings"

	"Synthara-Redux/APIs"
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

func Play(GuildID, UserID snowflake.ID, Args string) {

	Args = strings.TrimSpace(Args)

	fmt.Printf("Play command invoked in guild %s by user %s with args: %s\n", GuildID, UserID, Args)

	Guild := Structs.GetGuild(GuildID, true)

	if Guild == nil {

		return

	}

	Locale := Guild.Locale.Code()

	Guild.ResetInactivityTimer()

	if Args == "" {

		notifyLocalized(Guild, "Commands.Play.Error.NoQuery.Title", "Commands.Play.Error.NoQuery.Description", "Embeds.Categories.Error", Utils.ERROR)

		return

	}

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, UserID)

	if !VoiceStateExists || VoiceState.ChannelID == nil {

		notifyLocalized(Guild, "Commands.Play.Error.NotInVoiceChannel.Title", "Commands.Play.Error.NotInVoiceChannel.Description", "Embeds.Categories.Error", Utils.ERROR)

		return

	}

	ErrorConnecting := Guild.Connect(*VoiceState.ChannelID, Guild.Channels.Text)

	if ErrorConnecting != nil {

		notify(Guild, Localizations.Get("Commands.Play.Error.FailedToConnect.Title", Locale), Localizations.GetFormat("Commands.Play.Error.FailedToConnect.Description", Locale, ErrorConnecting.Error()), Localizations.Get("Embeds.Categories.Error", Locale), Utils.ERROR)

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

	trackPlayRequest(UserID, Song, URI)

	notifyPlayResult(Guild, Song, Pos, UserID)

}

func trackPlayRequest(UserID snowflake.ID, Song *Tidal.Song, URI string) {

	if Song == nil {

		return

	}

	go func() {

		User, UserError := Structs.GetUser(UserID.String())

		if UserError != nil || User == nil {

			return

		}

		User.AddRecentSearch(Song.Title, URI)

		if Song.TidalID != 0 {

			SongURI := "Synthara-Redux:" + APIs.URITypeTidalSong + ":" + strconv.FormatInt(Song.TidalID, 10)

			User.AddFavorite(SongURI)

		}

		if Song.MixID != "" {

			User.SetMostRecentMix(Song.MixID)

		}

	}()

}
