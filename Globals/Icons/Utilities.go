package Icons

import (
	"encoding/json"
	"fmt"
	"os"
)

type IconName string

const (

	Albums           IconName = "Albums"
	Call             IconName = "Call"
	ChatBubbles      IconName = "ChatBubbles"
	Pause            IconName = "Pause"
	Play             IconName = "Play"
	PlaySkipBack     IconName = "PlaySkipBack"
	PlaySkipForward  IconName = "PlaySkipForward"
	Repeat           IconName = "Repeat"
	Shuffle          IconName = "Shuffle"
	Sparkles         IconName = "Sparkles"
	Star             IconName = "Star"
	Trash            IconName = "Trash"
	
)

type Icon struct {

	Name string `json:"Name"`
	ID   string `json:"ID"`

}

var Manifest map[string]Icon

func Initialize() error {

	ManifestData, ReadError := os.ReadFile("./Globals/Icons/Manifest.json")

	if ReadError != nil {

		return fmt.Errorf("failed to read icons manifest.json: %w", ReadError)

	}

	ParseError := json.Unmarshal(ManifestData, &Manifest)

	if ParseError != nil {

		return fmt.Errorf("failed to parse icons manifest.json: %w", ParseError)

	}

	return nil

}

// Get retrieves an icon by its human-readable name.
func Get(Name IconName) Icon {

	if IconData, Exists := Manifest[string(Name)]; Exists {

		return IconData

	}

	return Icon{

		Name: "unknown",
		ID:   "",

	}

}

// GetID retrieves an icon ID by its human-readable name.
func GetID(Name IconName) string {

	return Get(Name).ID

}

// GetName retrieves an icon name by its human-readable name.
func GetName(Name IconName) string {

	return Get(Name).Name

}