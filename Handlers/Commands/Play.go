package Commands

import (
	"Synthara-Redux/APIs"
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func Play(Event *events.ApplicationCommandInteractionCreate) {

	DeferDone := make(chan struct{})

	go func() {

		Event.DeferCreateMessage(false)
		close(DeferDone)

	}()

	Locale := Event.Locale().Code()

	// Get the search query from command options

	Data := Event.SlashCommandInteractionData()
	Query := Data.String("query")

	if Query == "" {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NoQuery.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NoQuery.Description", Locale),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	// Check if user is in a voice channel

	if Event.Member() == nil {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInGuild.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInGuild.Description", Locale),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	GuildID := *Event.GuildID()

	VoiceState, VoiceStateExists := Utils.GetVoiceState(GuildID, Event.User().ID)

	if !VoiceStateExists {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.Get("Commands.Play.Error.NotInVoiceChannel.Description", Locale),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	ChannelID := VoiceState.ChannelID

	Guild := Structs.GetGuild(GuildID, true) // creates if not found

	// Connect to voice channel

	ErrorConnecting := Guild.Connect(*ChannelID, Event.Channel().ID())

	if ErrorConnecting != nil {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.FailedToConnect.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.FailedToConnect.Description", Locale, ErrorConnecting.Error()),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	// Route the input to a URI

	URI, ErrorRouting := APIs.Route(Query)

	if ErrorRouting != nil {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.InvalidInput.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.InvalidInput.Description", Locale, ErrorRouting.Error()),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	// Handle the URI

	SongFound, Pos, ErrorHandling := Guild.HandleURI(URI, Event.User().Mention())

	if ErrorHandling != nil {

		Utils.WaitFor(DeferDone)
		Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.MessageUpdate{

			Embeds: &[]discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Play.Error.FailedToHandle.Title", Locale),
				Author:      Localizations.Get("Embeds.Categories.Error", Locale),
				Description: Localizations.GetFormat("Commands.Play.Error.FailedToHandle.Description", Locale, ErrorHandling.Error()),
				Color:       Utils.ERROR,

			})},

		})

		return

	}

	// User persistence updates

	go func() {

		User, UserError := Structs.GetUser(Event.User().ID.String())

		if UserError == nil {

			if (User.FirstUse) {

				User.SetFirstUse(false)

			}
			
			User.AddRecentSearch(SongFound.Title, URI)

			// Track Favorites 

			if SongFound.TidalID != 0 {
				
				SongURI := "Synthara-Redux:" + APIs.URITypeTidalSong + ":" + strconv.FormatInt(SongFound.TidalID, 10)
				User.AddFavorite(SongURI)

			}

			// Track Mix

			if SongFound.MixID != "" {

				User.SetMostRecentMix(SongFound.MixID)

			}

		}

	}()

	// Send response with current song info

	State := Tidal.QueueInfo{

		Playing: true, // Forced here

		GuildID: GuildID,

		SongPosition: Pos,

		TotalPrevious: len(Guild.Queue.Previous),
		TotalUpcoming: len(Guild.Queue.Upcoming),

		Locale: Locale,

	}

	Utils.WaitFor(DeferDone)
	Event.Client().Rest.UpdateInteractionResponse(Event.ApplicationID(), Event.Token(), discord.NewMessageUpdateBuilder().AddEmbeds(SongFound.Embed(State)).AddActionRow(SongFound.Buttons(State)...).Build())

}