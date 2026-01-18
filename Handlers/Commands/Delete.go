package Commands

import (
	"Synthara-Redux/Globals"
	"Synthara-Redux/Globals/Localizations"
	"Synthara-Redux/Structs"
	"Synthara-Redux/Utils"
	"fmt"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func Delete(Event *events.ApplicationCommandInteractionCreate) {

	Locale := Event.Locale().Code()

	// This command is developer-only

	DeveloperIDs := os.Getenv("DEVELOPERS")
	UserID := Event.User().ID.String()

	if !strings.Contains(DeveloperIDs, UserID) {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Delete.Unauthorized.Title", Locale),
				Description: Localizations.Get("Commands.Delete.Unauthorized.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Data := Event.SlashCommandInteractionData()

	GuildIDString := Data.String("guild")
	Message := Data.String("message")

	// Handle "ALL" case

	if strings.ToUpper(GuildIDString) == "ALL" {

		Structs.GuildStoreMutex.Lock()
		GuildsToDelete := make([]*Structs.Guild, 0, len(Structs.GuildStore))

		for _, Guild := range Structs.GuildStore {

			GuildsToDelete = append(GuildsToDelete, Guild)

		}
		Structs.GuildStoreMutex.Unlock()

		DeletedCount := 0

		for _, Guild := range GuildsToDelete {

			// Send message to text channel

			if Guild.Channels.Text != 0 {

				_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
					AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

						Title:       Localizations.Get("Commands.Delete.Message.Title", Guild.Locale.Code()),
						Author:      Localizations.Get("Embeds.Categories.Notifications", Guild.Locale.Code()),
						Description: Message,
						Color:       Utils.ERROR,

					})).
					Build())

				if ErrorSending != nil {

					Utils.Logger.Error("Command", fmt.Sprintf("Error sending delete message to guild %s: %s", Guild.ID, ErrorSending.Error()))

				}

			}

			// Clear the queue

			Guild.Queue.Clear()

			// Cleanup the guild

			Guild.Cleanup(true)

			DeletedCount++

		}

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Delete.Success.AllTitle", Locale),
				Description: fmt.Sprintf(Localizations.Get("Commands.Delete.Success.AllDescription", Locale), DeletedCount),
				Color:       Utils.PRIMARY,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Handle specific guild case

	GuildID, ParseErr := snowflake.Parse(GuildIDString)

	if ParseErr != nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Delete.Error.InvalidGuild.Title", Locale),
				Description: Localizations.Get("Commands.Delete.Error.InvalidGuild.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	Guild := Structs.GetGuild(GuildID, false)

	if Guild == nil {

		Event.CreateMessage(discord.MessageCreate{

			Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Delete.Error.NoGuild.Title", Locale),
				Description: Localizations.Get("Commands.Delete.Error.NoGuild.Description", Locale),
				Color:       Utils.ERROR,

			})},

			Flags: discord.MessageFlagEphemeral,

		})

		return

	}

	// Get guild name for response

	GuildName := Guild.ID.String()

	CachedGuild, ExistsInCache := Globals.DiscordClient.Caches.GuildCache().Get(Guild.ID)

	if ExistsInCache {

		GuildName = CachedGuild.Name

	}

	// Send message to text channel

	if Guild.Channels.Text != 0 {

		_, ErrorSending := Globals.DiscordClient.Rest.CreateMessage(Guild.Channels.Text, discord.NewMessageCreateBuilder().
			AddEmbeds(Utils.CreateEmbed(Utils.EmbedOptions{

				Title:       Localizations.Get("Commands.Delete.Message.Title", Guild.Locale.Code()),
				Author:      Localizations.Get("Embeds.Categories.Notifications", Guild.Locale.Code()),
				Description: Message,
				Color:       Utils.ERROR,

			})).
			Build())

		if ErrorSending != nil {

			Utils.Logger.Error("Command", fmt.Sprintf("Error sending delete message to guild %s: %s", Guild.ID, ErrorSending.Error()))

		}

	}

	// Clear the queue

	Guild.Queue.Clear()

	// Cleanup the guild

	Guild.Cleanup(true)

	Event.CreateMessage(discord.MessageCreate{

		Embeds: []discord.Embed{Utils.CreateEmbed(Utils.EmbedOptions{

			Title:       Localizations.Get("Commands.Delete.Success.Title", Locale),
			Description: fmt.Sprintf(Localizations.Get("Commands.Delete.Success.Description", Locale), GuildName),
			Color:       Utils.PRIMARY,

		})},

		Flags: discord.MessageFlagEphemeral,

	})

}
