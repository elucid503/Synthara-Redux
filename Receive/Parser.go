package Receive

import (
	"strings"
	"unicode"
)

const (

	CommandPlay = "play"
	CommandPause = "pause"
	CommandResume = "resume"
	CommandNext = "next"
	CommandLast = "last"
	CommandLeave = "leave"
	CommandShuffle = "shuffle"
	CommandRepeat = "repeat"
	CommandReplay = "replay"
	CommandAutoplay = "autoplay"
	CommandVolume = "volume"

)

type ParsedCommand struct {

	Prefix string
	Command string
	Args string

}

func Parse(Text string) (ParsedCommand, bool) {

	Cleaned := strings.ToLower(strings.TrimSpace(Text))

	Cleaned = stripPunct(Cleaned)

	if Cleaned == "" {

		return ParsedCommand{}, false

	}

	Tokens := strings.Fields(Cleaned)

	if len(Tokens) == 0 {

		return ParsedCommand{}, false

	}

	PrefixIdx := -1

	for i := 0; i < len(Tokens); i++ {

		if fuzzyMatchSynthara(Tokens[i]) {

			PrefixIdx = i
			break

		}

	}

	var Rest []string

	if PrefixIdx >= 0 {

		Rest = Tokens[PrefixIdx+1:]

	} else {

		Rest = Tokens

	}

	Cmd, Args, OK := firstCommandInTokens(Rest)

	if !OK {

		return ParsedCommand{}, false

	}

	Args = strings.TrimSpace(Args)

	if CommandClearsTrailingArgs(Cmd) {

		Args = ""

	}

	if !transcriptLooksEnglish(Text) {

		return ParsedCommand{}, false

	}

	return ParsedCommand{

		Prefix: "Synthara",

		Command: Cmd,
		Args: Args,

	}, true

}

func CommandClearsTrailingArgs(Command string) bool {

	switch Command {

	case CommandPause, CommandResume, CommandNext, CommandLast, CommandLeave:

		return true

	default:

		return false

	}

}

// CommandNeedsMultiWordArgs returns true for commands whose argument is a free-form phrase (e.g. a song name) that may span several words.
func CommandNeedsMultiWordArgs(Command string) bool {

	switch Command {

	case CommandPlay:

		return true

	default:

		return false

	}

}

func CommandDispatchesImmediately(Command, Args string) bool {

	switch Command {

	case CommandPause, CommandResume, CommandNext, CommandLast, CommandLeave, CommandShuffle, CommandRepeat, CommandReplay, CommandAutoplay:

		return true

	case CommandVolume:

		return strings.TrimSpace(Args) != "" // volume command can be dispatched immediately if it has any argument at all

	default:

		return false

	}

}

func transcriptLooksEnglish(Text string) bool {

	if strings.TrimSpace(Text) == "" {

		return false

	}

	for _, R := range Text {

		if unicode.Is(unicode.Han, R) || unicode.Is(unicode.Hiragana, R) || unicode.Is(unicode.Katakana, R) || unicode.Is(unicode.Hangul, R) {

			return false

		}

	}

	return true

}

func fuzzyMatchSynthara(Token string) bool {

	Token = strings.ToLower(Token)
	Target := "synthara"

	if Token == Target {

		return true

	}

	switch Token {

	case "tara", "dara", "nara", "thara", "sara", "syntha", "synth", "syntara", "synara", "sinthara", "cynthara", "synthera", "centaur", "santoro", "santara", "sentara", "cintra", "sintera", "syntera", "synthia":

		return true

	}

	if len(Token) >= 3 && (Token[0] == 's' || Token[0] == 'c') {

		if levenshtein(Token, Target) <= 4 {

			return true

		}

	}

	if len(Token) >= 4 && levenshtein(Token, Target) <= 3 {

		return true

	}

	return false

}

func firstCommandInTokens(Tokens []string) (string, string, bool) {

	for i, Tok := range Tokens {

		if Cmd := normalizeCommand(Tok); Cmd != "" {

			Args := ""

			if i+1 < len(Tokens) {

				Args = strings.Join(Tokens[i+1:], " ")

			}

			return Cmd, strings.TrimSpace(Args), true

		}

	}

	return "", "", false

}

func normalizeCommand(Token string) string {

	switch Token {

	case "play", "plays", "played", "enqueue", "add", "queue":

		return CommandPlay

	case "pause", "paused", "pausing":

		return CommandPause

	case "resume", "unpause", "continue":

		return CommandResume

	case "next", "skip", "forward":

		return CommandNext

	case "last", "previous", "back", "prev":

		return CommandLast

	case "leave", "disconnect", "dc", "quit":

		return CommandLeave

	case "shuffle", "shuffled", "shuffling":

		return CommandShuffle

	case "repeat", "loop", "looped":

		return CommandRepeat

	case "replay", "again":

		return CommandReplay

	case "autoplay", "auto", "radio":

		return CommandAutoplay

	case "volume", "vol", "loudness":

		return CommandVolume

	}

	return ""

}

func stripPunct(S string) string {

	var B strings.Builder

	for _, R := range S {

		if unicode.IsLetter(R) || unicode.IsDigit(R) || R == ' ' || R == '\'' {

			B.WriteRune(R)

		} else {

			B.WriteRune(' ')

		}

	}

	return B.String()

}

// Levenshtein distance implementation adapted from https://en.wikipedia.org/wiki/Levenshtein_distance#Iterative_with_two_matrix_rows
func levenshtein(A, B string) int {

	if A == B {

		return 0

	}

	if len(A) == 0 {

		return len(B)

	}

	if len(B) == 0 {

		return len(A)

	}

	Prev := make([]int, len(B)+1)
	Curr := make([]int, len(B)+1)

	for j := 0; j <= len(B); j++ {

		Prev[j] = j

	}

	for i := 1; i <= len(A); i++ {

		Curr[0] = i

		for j := 1; j <= len(B); j++ {

			Cost := 1

			if A[i-1] == B[j-1] {

				Cost = 0

			}

			Min := Prev[j] + 1

			if Curr[j-1]+1 < Min {

				Min = Curr[j-1] + 1

			}

			if Prev[j-1]+Cost < Min {

				Min = Prev[j-1] + Cost

			}

			Curr[j] = Min

		}

		Prev, Curr = Curr, Prev

	}

	return Prev[len(B)]

}
