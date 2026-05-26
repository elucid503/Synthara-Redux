package Voice

import (
	"strconv"
	"strings"

	"Synthara-Redux/Structs"
)

func ParseShuffleEnabled(Args string, Current bool) bool {

	Args = strings.TrimSpace(strings.ToLower(Args))

	if Args == "" {

		return !Current

	}

	switch Args {

	case "on", "enable", "enabled", "true", "yes":

		return true

	case "off", "disable", "disabled", "false", "no":

		return false

	default:

		return !Current

	}

}

func ParseRepeatMode(Args string, Current int) int {

	Args = strings.TrimSpace(strings.ToLower(Args))

	if Args == "" {

		return CycleRepeatMode(Current)

	}

	Tokens := strings.Fields(Args)

	if len(Tokens) > 0 {

		Args = Tokens[0]

	}

	switch Args {

	case "0", "off", "none", "disable", "disabled":

		return Structs.RepeatOff

	case "1", "one", "single", "song":

		return Structs.RepeatOne

	case "2", "all", "queue", "on":

		return Structs.RepeatAll

	}

	if Value, ErrParse := strconv.Atoi(Args); ErrParse == nil {

		switch Value {

		case Structs.RepeatOff, Structs.RepeatOne, Structs.RepeatAll:

			return Value

		}

	}

	return CycleRepeatMode(Current)

}

func CycleRepeatMode(Current int) int {

	switch Current {

	case Structs.RepeatOff:

		return Structs.RepeatOne

	case Structs.RepeatOne:

		return Structs.RepeatAll

	default:

		return Structs.RepeatOff

	}

}

func ParseReplayPosition(Args string) int {

	Args = strings.TrimSpace(Args)

	if Args == "" {

		return 0

	}

	Tokens := strings.Fields(Args)

	if len(Tokens) == 0 {

		return 0

	}

	if Value, ErrParse := strconv.Atoi(Tokens[0]); ErrParse == nil && Value >= 0 {

		return Value

	}

	return 0

}

func ParseAutoplayEnabled(Args string, Current bool) bool {

	return ParseShuffleEnabled(Args, Current)

}
