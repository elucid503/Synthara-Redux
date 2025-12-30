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

	for i, Key := range Keys {

		if i == (len(Keys) - 1) {

			// Last key - should be the locale map

			if LocaleMap, Valid := Current[Key].(map[string]interface{}); Valid {

				// Tries requested locale

				if Value, Exists := LocaleMap[Locale].(string); Exists {

					return Value

				}

				// Fallbacks to 'default' locale

				if Value, Exists := LocaleMap[Default].(string); Exists {

					return Value

				}

				return fmt.Sprintf("[Missing: %s]", Path)

			}

			return fmt.Sprintf("[Invalid: %s]", Path)

		}

		// Attempts to traverse deeper

		if Next, Valid := Current[Key].(map[string]interface{}); Valid {

			Current = Next

		} else {

			return fmt.Sprintf("[NotFound: %s]", Path)

		}

	}

	return fmt.Sprintf("[Error: %s]", Path)

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