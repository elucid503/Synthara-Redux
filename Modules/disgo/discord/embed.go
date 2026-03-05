package discord

import (
	"fmt"
	"time"
)

// EmbedType is the type of Embed
type EmbedType string

// Constants for EmbedType
const (
	EmbedTypeRich                  EmbedType = "rich"
	EmbedTypeImage                 EmbedType = "image"
	EmbedTypeVideo                 EmbedType = "video"
	EmbedTypeGifV                  EmbedType = "gifv"
	EmbedTypeArticle               EmbedType = "article"
	EmbedTypeLink                  EmbedType = "link"
	EmbedTypeAutoModerationMessage EmbedType = "auto_moderation_message"
	EmbedTypePollResult            EmbedType = "poll_result"
)

// NewEmbed returns a new Embed struct with no fields set.
func NewEmbedBuilder() Embed {
	return Embed{}
}

// Embed allows you to send embeds to discord
type Embed struct {
	Title       string         `json:"title,omitempty"`
	Type        EmbedType      `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url,omitempty"`
	Timestamp   *time.Time     `json:"timestamp,omitempty"`
	Color       int            `json:"color,omitempty"`
	Footer      *EmbedFooter   `json:"footer,omitempty"`
	Image       *EmbedResource `json:"image,omitempty"`
	Thumbnail   *EmbedResource `json:"thumbnail,omitempty"`
	Video       *EmbedResource `json:"video,omitempty"`
	Provider    *EmbedProvider `json:"provider,omitempty"`
	Author      *EmbedAuthor   `json:"author,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
}

// SetTitle sets the title of the Embed
func (e Embed) SetTitle(title string) Embed {
	e.Title = title
	return e
}

// SetTitlef sets the title of the Embed with format
func (e Embed) SetTitlef(title string, a ...any) Embed {
	return e.SetTitle(fmt.Sprintf(title, a...))
}

// SetDescription sets the description of the Embed
func (e Embed) SetDescription(description string) Embed {
	e.Description = description
	return e
}

// SetDescriptionf sets the description of the Embed with format
func (e Embed) SetDescriptionf(description string, a ...any) Embed {
	return e.SetDescription(fmt.Sprintf(description, a...))
}

// SetEmbedAuthor sets the author of the Embed using an EmbedAuthor struct
func (e Embed) SetEmbedAuthor(author *EmbedAuthor) Embed {
	e.Author = author
	return e
}

// SetAuthor sets the author of the Embed with all properties
func (e Embed) SetAuthor(name string, url string, iconURL string) Embed {
	if e.Author == nil {
		e.Author = &EmbedAuthor{}
	}
	e.Author.Name = name
	e.Author.URL = url
	e.Author.IconURL = iconURL
	return e
}

// SetAuthorName sets the author name of the Embed
func (e Embed) SetAuthorName(name string) Embed {
	if e.Author == nil {
		e.Author = &EmbedAuthor{}
	}
	e.Author.Name = name
	return e
}

// SetAuthorNamef sets the author name of the Embed with format
func (e Embed) SetAuthorNamef(name string, a ...any) Embed {
	return e.SetAuthorName(fmt.Sprintf(name, a...))
}

// SetAuthorURL sets the author URL of the Embed
func (e Embed) SetAuthorURL(url string) Embed {
	if e.Author == nil {
		e.Author = &EmbedAuthor{}
	}
	e.Author.URL = url
	return e
}

// SetAuthorURLf sets the author URL of the Embed with format
func (e Embed) SetAuthorURLf(url string, a ...any) Embed {
	return e.SetAuthorURL(fmt.Sprintf(url, a...))
}

// SetAuthorIcon sets the author icon of the Embed
func (e Embed) SetAuthorIcon(iconURL string) Embed {
	if e.Author == nil {
		e.Author = &EmbedAuthor{}
	}
	e.Author.IconURL = iconURL
	return e
}

// SetAuthorIconf sets the author icon of the Embed with format
func (e Embed) SetAuthorIconf(iconURL string, a ...any) Embed {
	return e.SetAuthorIcon(fmt.Sprintf(iconURL, a...))
}

// SetColor sets the color of the Embed
// The color should be an integer representation of a hexadecimal color code (e.g. 0xFF0000 for red)
func (e Embed) SetColor(color int) Embed {
	e.Color = color
	return e
}

// SetEmbedFooter sets the footer of the Embed
func (e Embed) SetEmbedFooter(footer *EmbedFooter) Embed {
	e.Footer = footer
	return e
}

// SetFooter sets the footer icon of the Embed
func (e Embed) SetFooter(text string, iconURL string) Embed {
	if e.Footer == nil {
		e.Footer = &EmbedFooter{}
	}
	e.Footer.Text = text
	e.Footer.IconURL = iconURL
	return e
}

// SetFooterText sets the footer text of the Embed
func (e Embed) SetFooterText(text string) Embed {
	if e.Footer == nil {
		e.Footer = &EmbedFooter{}
	}
	e.Footer.Text = text
	return e
}

// SetFooterTextf sets the footer text of the Embed with format
func (e Embed) SetFooterTextf(text string, a ...any) Embed {
	return e.SetFooterText(fmt.Sprintf(text, a...))
}

// SetFooterIcon sets the footer icon of the Embed
func (e Embed) SetFooterIcon(iconURL string) Embed {
	if e.Footer == nil {
		e.Footer = &EmbedFooter{}
	}
	e.Footer.IconURL = iconURL
	return e
}

// SetFooterIconf sets the footer icon of the Embed
func (e Embed) SetFooterIconf(iconURL string, a ...any) Embed {
	return e.SetFooterIcon(fmt.Sprintf(iconURL, a...))
}

// SetImage sets the image of the Embed
func (e Embed) SetImage(url string) Embed {
	if e.Image == nil {
		e.Image = &EmbedResource{}
	}
	e.Image.URL = url
	return e
}

// SetImagef sets the image of the Embed with format
func (e Embed) SetImagef(url string, a ...any) Embed {
	return e.SetImage(fmt.Sprintf(url, a...))
}

// SetThumbnail sets the thumbnail of the Embed
func (e Embed) SetThumbnail(url string) Embed {
	if e.Thumbnail == nil {
		e.Thumbnail = &EmbedResource{}
	}
	e.Thumbnail.URL = url
	return e
}

// SetThumbnailf sets the thumbnail of the Embed with format
func (e Embed) SetThumbnailf(url string, a ...any) Embed {
	return e.SetThumbnail(fmt.Sprintf(url, a...))
}

// SetURL sets the URL of the Embed
func (e Embed) SetURL(url string) Embed {
	e.URL = url
	return e
}

// SetURLf sets the URL of the Embed with format
func (e Embed) SetURLf(url string, a ...any) Embed {
	return e.SetURL(fmt.Sprintf(url, a...))
}

// SetTimestamp sets the timestamp of the Embed
func (e Embed) SetTimestamp(time time.Time) Embed {
	e.Timestamp = &time
	return e
}

// AddField adds a field to the Embed by name and value
func (e Embed) AddField(name string, value string, inline bool) Embed {
	e.Fields = append(e.Fields, EmbedField{Name: name, Value: value, Inline: &inline})
	return e
}

// SetField sets a field to the Embed by name and value
func (e Embed) SetField(i int, name string, value string, inline bool) Embed {
	if len(e.Fields) > i {
		e.Fields[i] = EmbedField{Name: name, Value: value, Inline: &inline}
	}
	return e
}

// AddFields adds multiple fields to the Embed
func (e Embed) AddFields(fields ...EmbedField) Embed {
	e.Fields = append(e.Fields, fields...)
	return e
}

// SetFields sets fields of the Embed
func (e Embed) SetFields(fields ...EmbedField) Embed {
	e.Fields = fields
	return e
}

// ClearFields removes all the fields from the Embed
func (e Embed) ClearFields() Embed {
	e.Fields = []EmbedField{}
	return e
}

// RemoveField removes a field from the Embed
func (e Embed) RemoveField(i int) Embed {
	if len(e.Fields) > i {
		e.Fields = append(e.Fields[:i], e.Fields[i+1:]...)
	}
	return e
}

func (e Embed) FindField(fieldFindFunc func(field EmbedField) bool) (EmbedField, bool) {
	for _, field := range e.Fields {
		if fieldFindFunc(field) {
			return field, true
		}
	}
	return EmbedField{}, false
}

func (e Embed) FindAllFields(fieldFindFunc func(field EmbedField) bool) []EmbedField {
	var fields []EmbedField
	for _, field := range e.Fields {
		if fieldFindFunc(field) {
			fields = append(fields, field)
		}
	}
	return fields
}

// The EmbedResource of an Embed.Image/Embed.Thumbnail/Embed.Video
type EmbedResource struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

// The EmbedProvider of an Embed
type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// The EmbedAuthor of an Embed
type EmbedAuthor struct {
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// The EmbedFooter of an Embed
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// EmbedField (s) of an Embed
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline *bool  `json:"inline,omitempty"`
}

type EmbedFieldPollResult string

const (
	EmbedFieldPollResultQuestionText              EmbedFieldPollResult = "poll_question_text"
	EmbedFieldPollResultVictorAnswerVotes         EmbedFieldPollResult = "victor_answer_votes"
	EmbedFieldPollResultTotalVotes                EmbedFieldPollResult = "total_votes"
	EmbedFieldPollResultVictorAnswerID            EmbedFieldPollResult = "victor_answer_id"
	EmbedFieldPollResultVictorAnswerText          EmbedFieldPollResult = "victor_answer_text"
	EmbedFieldPollResultVictorAnswerEmojiID       EmbedFieldPollResult = "victor_answer_emoji_id"
	EmbedFieldPollResultVictorAnswerEmojiName     EmbedFieldPollResult = "victor_answer_emoji_name"
	EmbedFieldPollResultVictorAnswerEmojiAnimated EmbedFieldPollResult = "victor_answer_emoji_animated"
)
