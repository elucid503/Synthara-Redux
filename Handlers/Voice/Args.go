package Voice

import (
	"strconv"
	"strings"

	"Synthara-Redux/Structs"
)

func ParseShuffleEnabled(args string, current bool) bool {

	args = strings.TrimSpace(strings.ToLower(args))

	if args == "" {
		return !current
	}

	switch args {

	case "on", "enable", "enabled", "true", "yes":
		return true

	case "off", "disable", "disabled", "false", "no":
		return false

	default:
		return !current

	}

}

func ParseRepeatMode(args string, current int) int {

	args = strings.TrimSpace(strings.ToLower(args))

	if args == "" {
		return CycleRepeatMode(current)
	}

	token, ok := firstToken(args)

	if !ok {
		return CycleRepeatMode(current)
	}

	switch token {

	case "0", "off", "none", "disable", "disabled":
		return Structs.RepeatOff

	case "1", "one", "single", "song":
		return Structs.RepeatOne

	case "2", "all", "queue", "on":
		return Structs.RepeatAll

	}

	if value, err := strconv.Atoi(token); err == nil {
		switch value {

		case Structs.RepeatOff, Structs.RepeatOne, Structs.RepeatAll:
			return value

		}
	}

	return CycleRepeatMode(current)

}

func CycleRepeatMode(current int) int {

	switch current {

	case Structs.RepeatOff:
		return Structs.RepeatOne

	case Structs.RepeatOne:
		return Structs.RepeatAll

	default:
		return Structs.RepeatOff

	}

}

func ParseReplayPosition(args string) int {

	args = strings.TrimSpace(args)

	if args == "" {
		return 0
	}

	token, ok := firstToken(args)

	if !ok {
		return 0
	}

	if value, err := strconv.Atoi(token); err == nil && value >= 0 {
		return value
	}

	return 0

}

func ParseAutoplayEnabled(args string, current bool) bool {
	return ParseShuffleEnabled(args, current)
}

func ParseVolumeLevel(args string, current int) (int, bool) {

	token, ok := firstToken(args)

	if !ok {
		return current, false
	}

	switch token {

	case "low", "down", "quieter", "lower", "decrease":
		return Structs.ClampVolume(current - Structs.VolumeStep), true

	case "high", "up", "louder", "higher", "increase":
		return Structs.ClampVolume(current + Structs.VolumeStep), true

	}

	if value, err := strconv.Atoi(strings.TrimSuffix(token, "%")); err == nil {
		return Structs.ClampVolume(value), true
	}

	return current, false

}

func ParseSpeedMilli(args string, current int) (int, bool) {
	token, ok := firstToken(args)
	if !ok {
		return current, false
	}

	switch token {
	case "faster", "speedup", "speed-up", "quicker", "up":
		return Structs.ClampSpeedMilli(current + Structs.SpeedStepMilli), true
	case "slower", "slowdown", "slow-down", "down":
		return Structs.ClampSpeedMilli(current - Structs.SpeedStepMilli), true
	case "normal", "default", "reset", "1", "1.0", "1.00", "1.00x", "1.0x", "1x":
		return Structs.DefaultSpeedMilli, true
	}

	token = strings.TrimSuffix(token, "x")
	if multiplier, err := strconv.ParseFloat(token, 64); err == nil {
		return Structs.ClampSpeedMilli(int(multiplier * 1000)), true
	}

	if millis, err := strconv.Atoi(token); err == nil {
		return Structs.ClampSpeedMilli(millis), true
	}

	return current, false
}

func ParseReverbPercent(args string, current int) (int, bool) {
	return parseStepLevel(args, current, Structs.ReverbStep, Structs.DefaultReverb,
		Structs.AllowedReverbPercent[len(Structs.AllowedReverbPercent)-1],
		Structs.ClampReverb,
		[]string{"more", "wet", "higher", "up", "increase"},
		[]string{"less", "lower", "down", "decrease", "dry"},
		[]string{"off", "none", "zero", "0"},
		[]string{"max", "full", "maximum", "75", "100"},
	)
}

func parseStepLevel(
	args string,
	current, step, defaultVal, maxVal int,
	clamp func(int) int,
	up, down, off, max []string,
) (int, bool) {
	token, ok := firstToken(args)
	if !ok {
		return current, false
	}

	for _, word := range up {
		if token == word {
			return clamp(current + step), true
		}
	}

	for _, word := range down {
		if token == word {
			return clamp(current - step), true
		}
	}

	for _, word := range off {
		if token == word {
			return defaultVal, true
		}
	}

	for _, word := range max {
		if token == word {
			return maxVal, true
		}
	}

	if value, err := strconv.Atoi(strings.TrimSuffix(token, "%")); err == nil {
		return clamp(value), true
	}

	return current, false
}

func firstToken(args string) (string, bool) {

	args = strings.TrimSpace(strings.ToLower(args))

	if args == "" {
		return "", false
	}

	return strings.Fields(args)[0], true

}