package Utils

import "github.com/disgoorg/disgo/discord"

// Embed colors
const (
	WHITE = 0xFFFFFF
	RED   = 0xFFB3BA
)

type EmbedOptions struct {

	Title       string
	Description string

	Author      string
	Footer      string

	Color       int
	Thumbnail   string
	URL         string

}

func CreateEmbed(Options EmbedOptions) discord.Embed {

	EmbedBuilder := discord.NewEmbedBuilder()

	if Options.Title != "" {

		EmbedBuilder.SetTitle(Options.Title)

	}

	if Options.Description != "" {

		EmbedBuilder.SetDescription(Options.Description)

	}

	if Options.Author != "" {

		EmbedBuilder.SetAuthorName(Options.Author)

	}

	if Options.Footer != "" {

		EmbedBuilder.SetFooterText(Options.Footer)

	}

	if Options.Color != 0 {

		EmbedBuilder.SetColor(Options.Color)

	} else {

		EmbedBuilder.SetColor(WHITE) // Defaults to white

	}

	if Options.Thumbnail != "" {

		EmbedBuilder.SetThumbnail(Options.Thumbnail)

	}

	if Options.URL != "" {

		EmbedBuilder.SetURL(Options.URL)

	}
	
	return EmbedBuilder.Build()

}