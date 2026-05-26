package Voice

import (
	"fmt"

	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func guildAndLocale(GuildID snowflake.ID) (*Structs.Guild, string) {

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		return nil, "en"

	}

	return Guild, Guild.Locale.Code()

}

func notify(Guild *Structs.Guild, Title, Description, Author string, Color int) {

	if Guild == nil || Guild.Channels.Text == 0 {

		return

	}

	Embed := Utils.CreateEmbed(Utils.EmbedOptions{

		Title:       Title,
		Description: Description,
		Author:      Author,
		Color:       Color,

	})

	_, ErrSend := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreate().AddEmbeds(Embed))

	if ErrSend != nil {

		Utils.Logger.Error("Voice", fmt.Sprintf("Failed to send voice notification to guild %s: %s", Guild.ID, ErrSend.Error()))

	}

}

func notifyLocalized(Guild *Structs.Guild, TitleKey, DescriptionKey, AuthorKey string, Color int) {

	if Guild == nil {

		return

	}

	Locale := Guild.Locale.Code()

	notify(Guild,

		Localizations.Get(TitleKey, Locale),
		Localizations.Get(DescriptionKey, Locale),
		Localizations.Get(AuthorKey, Locale),
		Color,
	)

}

func notifyNowPlaying(Guild *Structs.Guild, Song *Tidal.Song, Pos int, Locale string) {

	if Guild == nil || Song == nil || Guild.Channels.Text == 0 {

		return

	}

	State := Tidal.QueueInfo{

		Playing: true,

		GuildID: Guild.ID,

		SongPosition:  Pos,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Embed := Song.Embed(State)

	Description := Localizations.Get("Embeds.NowPlaying.AddedViaVoice", Locale)

	NewDescription := Embed.Description

	NewDescription += "\n\n" + Description

	Embed.Description = NewDescription

	_, ErrSend := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text,

		discord.NewMessageCreate().AddEmbeds(Embed).AddActionRow(Song.Buttons(State)...),
	)

	if ErrSend != nil {

		Utils.Logger.Error("Voice", fmt.Sprintf("Failed to send now playing to guild %s: %s", Guild.ID, ErrSend.Error()))

	}

}
