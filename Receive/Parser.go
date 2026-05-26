package Receive

import (
	"strings"
	"unicode"
)

// Map of supported command verbs; allows fuzzy matching
const (

	CommandPlay = "play"
	CommandPause = "pause"
	CommandResume = "resume"

)

// ParsedCommand is the structured result of parsing a transcribed utterance.
type ParsedCommand struct {

	Prefix string
	Command string
	Args string

}

// Parse takes the raw text from xAI and pulls out the prefix, command and args if possible.
func Parse(Text string) (ParsedCommand, bool) {

	Cleaned := strings.ToLower(strings.TrimSpace(Text))

	// Drop punctuation but keep spaces

	Cleaned = stripPunct(Cleaned)

	if Cleaned == "" {

		return ParsedCommand{}, false

	}

	Tokens := strings.Fields(Cleaned)

	if len(Tokens) == 0 {

		return ParsedCommand{}, false

	}

	// xAI sometimes prepends conversational fillers ("hey", "okay", "uh"), so we scan for the wake word in the first few tokens

	PrefixIdx := -1

	for i := 0; i < len(Tokens) && i < 3; i++ {

		if fuzzyMatchSynthara(Tokens[i]) {

			PrefixIdx = i
			break

		}

	}

	// If the wake word didn't survive transcription, we can still continue...

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

	// TODO: generalize this for other commands if we add more; some commands may want to ignore args if they look like garbage. perhaps a field of the command struct?

	if Cmd == CommandPause || Cmd == CommandResume {

		Args = ""

	}

	if !transcriptLooksEnglish(Text) {

		return ParsedCommand{}, false

	}

	return ParsedCommand{

		Prefix:  "Synthara", // hardcoded

		Command: Cmd,
		Args: Args,

	}, true

}

// TranscriptHasWake reports whether transcribed text contains the wake word.
func TranscriptHasWake(Text string) bool {

	if !transcriptLooksEnglish(Text) {

		return false

	}

	Cleaned := strings.ToLower(strings.TrimSpace(stripPunct(Text)))

	if Cleaned == "" {

		return false

	}

	for _, Tok := range strings.Fields(Cleaned) {

		if fuzzyMatchSynthara(Tok) {

			return true

		}

	}

	Compact := strings.ReplaceAll(Cleaned, " ", "")

	for _, Sub := range wakeSubstringHints() {

		if strings.Contains(Compact, Sub) {

			return true

		}

	}

	return false

}

// TranscriptHasWakeProbe is a looser check used only for STT wake fallback.
func TranscriptHasWakeProbe(Text string) bool {

	if TranscriptHasWake(Text) {

		return true

	}

	if !transcriptLooksEnglish(Text) {

		return false

	}

	Compact := strings.ReplaceAll(strings.ToLower(stripPunct(Text)), " ", "")

	if len(Compact) < 3 {

		return false

	}

	for _, Sub := range wakeSubstringHints() {

		if strings.Contains(Compact, Sub) {

			return true

		}

	}

	return false

}

func wakeSubstringHints() []string {

	return []string{

		"synthar", "synara", "sinthar", "cynthar", "synther", "synthra", "synth",
		"syntha", "santha", "santor", "centaur", "cintra", "sintera", "syntera",
		"tara", "dara", "nara", "thara", "sentara", "syntara",

	}

}

// transcriptLooksEnglish rejects CJK and similar scripts xAI sometimes emits with language=en.
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

// fuzzyMatchSynthara checks whether a token is plausibly "Synthara"
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

	// Same starting letter as synthara (s/c) with generous edit distance.
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

	case "play", "plays", "played":

		return CommandPlay

	case "pause", "paused", "pausing":

		return CommandPause

	case "resume", "unpause", "continue":

		return CommandResume

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

// levenshtein implements the standard edit distance with O(n) memory.
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
