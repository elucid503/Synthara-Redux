package Localizations

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var Manifest map[string]interface{}

const Default = "en-US"

func Initialize() error {

	ManifestData, ReadError := os.ReadFile("./Globals/Localizations/Manifest.json")

	if ReadError != nil {

		return fmt.Errorf("failed to read localizations manifest.json: %w", ReadError)

	}

	ParseError := json.Unmarshal(ManifestData, &Manifest)

	if ParseError != nil {

		return fmt.Errorf("failed to parse localizations manifest.json: %w", ParseError)

	}

	return nil

}

// GetLocalized retrieves a localized string for the given path and locale code.
func Get(Path string, Locale string) string {

	Keys := strings.Split(Path, ".")

	Current := Manifest

	if len(Keys) == 0 {

		return fmt.Sprintf("[Error: %s]", Path)

	}

	// Traverses to parent of final key

	for _, Key := range Keys[:len(Keys)-1] { // All but last key

		Next, Valid := Current[Key].(map[string]interface{})

		if !Valid {

			return fmt.Sprintf("[NotFound: %s]", Path) // Key not found at this level

		}

		Current = Next

	}

	// Final key should be locale map

	Last := Keys[len(Keys)-1]

	LocaleMap, Valid := Current[Last].(map[string]interface{})

	if !Valid {

		return fmt.Sprintf("[Invalid: %s]", Path) // Key not found at final level

	}

	if Value, Exists := LocaleMap[Locale].(string); Exists {

		return Value // Found specific locale

	}

	if Value, Exists := LocaleMap[Default].(string); Exists {

		return Value // Fallback to default locale

	}

	return fmt.Sprintf("[Missing: %s]", Path) // Nothing found

}

// GetLocalizedFormat retrieves a localized format string and applies the provided arguments.
func GetFormat(Path string, Locale string, Args ...interface{}) string {

	Format := Get(Path, Locale)

	if len(Args) == 0 {

		return Format

	}

	return fmt.Sprintf(Format, Args...)

}

// Pluralize is a utility function to return the singular or plural form of a word based on count.
func Pluralize(Word string, Count int, Locale string) string {

	if Count == 1 {

		return Get(fmt.Sprintf("Common.%s", Word), Locale)

	}

	return Get(fmt.Sprintf("Common.%ss", Word), Locale)

}