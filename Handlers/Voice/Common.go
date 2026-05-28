package Voice

import (
	"fmt"

	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Receive"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"Synthara-Redux/Validation"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func guildAndLocale(GuildID snowflake.ID) (*Structs.Guild, string) {

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		return nil, Localizations.Default

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

	notify(Guild, Localizations.Get(TitleKey, Locale), Localizations.Get(DescriptionKey, Locale), Localizations.Get(AuthorKey, Locale), Color)

}

func notifyLocalizedWithMember(Guild *Structs.Guild, UserID snowflake.ID, TitleKey, DescriptionKey, AuthorKey string, Color int) {

	if Guild == nil {

		return

	}

	Locale := Guild.Locale.Code()

	Mention := fmt.Sprintf("<@%s>", UserID)

	notify(Guild, Localizations.Get(TitleKey, Locale), Localizations.GetFormat(DescriptionKey, Locale, Mention), Localizations.Get(AuthorKey, Locale), Color)

}

func notifyValidationEmbed(Guild *Structs.Guild, Embed *discord.Embed) {

	if Guild == nil || Embed == nil {

		return

	}

	Author := ""

	if Embed.Author != nil {

		Author = Embed.Author.Name

	}

	notify(Guild, Embed.Title, Embed.Description, Author, Embed.Color)

}

func requireVoice(Guild *Structs.Guild, GuildID, UserID snowflake.ID, Locale string) bool {

	if Embed := Validation.VoiceStateError(GuildID, UserID, Locale); Embed != nil {

		notifyValidationEmbed(Guild, Embed)
		voiceRespond(GuildID, "You're not in a voice channel.")

		return false

	}

	return true

}

func voiceRespond(GuildID snowflake.ID, text string) {

	Receive.EmitVoiceResponse(GuildID, text)

}

func notifyCurrentSongWithMember(Guild *Structs.Guild, UserID snowflake.ID) {

	if Guild == nil || Guild.Queue.Current == nil || Guild.Channels.Text == 0 {

		return

	}

	Locale := Guild.Locale.Code()

	Song := Guild.Queue.Current

	Playing := Guild.Queue.State != Structs.StatePaused

	State := Tidal.QueueInfo{

		Playing: Playing,

		GuildID: Guild.ID,

		SongPosition: 0,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Embed := Song.Embed(State)

	Mention := fmt.Sprintf("<@%s>", UserID)

	Embed.Description += "\n" + Localizations.GetFormat("Embeds.NowPlaying.AddedByMemberViaVoice", Locale, Mention)

	_, ErrSend := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text,

		discord.NewMessageCreate().AddEmbeds(Embed).AddActionRow(Song.Buttons(State)...),
	)

	if ErrSend != nil {

		Utils.Logger.Error("Voice", fmt.Sprintf("Failed to send current song to guild %s: %s", Guild.ID, ErrSend.Error()))

	}

}

func notifyPlayResult(Guild *Structs.Guild, Song *Tidal.Song, Pos int, UserID snowflake.ID) {

	if Guild == nil || Song == nil || Guild.Channels.Text == 0 {

		return

	}

	Locale := Guild.Locale.Code()

	Playing := Guild.Queue.State != Structs.StatePaused

	State := Tidal.QueueInfo{

		Playing: Playing,

		GuildID: Guild.ID,

		SongPosition: Pos,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Embed := Song.Embed(State)

	Mention := fmt.Sprintf("<@%s>", UserID)

	VoiceLine := Localizations.GetFormat("Embeds.NowPlaying.AddedByMemberViaVoice", Locale, Mention)

	Embed.Description += "\n" + VoiceLine

	_, ErrSend := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text,

		discord.NewMessageCreate().AddEmbeds(Embed).AddActionRow(Song.Buttons(State)...),
	)

	if ErrSend != nil {

		Utils.Logger.Error("Voice", fmt.Sprintf("Failed to send play result to guild %s: %s", Guild.ID, ErrSend.Error()))

	}

}
