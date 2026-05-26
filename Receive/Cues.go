package Receive

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

// VoiceCueKind matches Audio.CueKind without importing Audio (avoids cycles).
type VoiceCueKind int

const (
	VoiceCueWake VoiceCueKind = iota
	VoiceCueEnd // 1

)

// VoiceCueHandler plays wake/end feedback for a guild (registered from Handlers/Structs).
type VoiceCueHandler func(GuildID snowflake.ID, Kind VoiceCueKind)

// VoiceCaptureDuckHandler ducks or restores music for the full capture window.
type VoiceCaptureDuckHandler func(GuildID snowflake.ID, Start bool)

var (

	voiceCueHandlerMu sync.RWMutex
	voiceCueHandlerFn VoiceCueHandler

	voiceCaptureDuckHandlerMu sync.RWMutex
	voiceCaptureDuckHandlerFn VoiceCaptureDuckHandler

)

// SetVoiceCueHandler registers playback feedback for wake/capture-end (e.g. from Structs).
func SetVoiceCueHandler(fn VoiceCueHandler) {

	voiceCueHandlerMu.Lock()
	voiceCueHandlerFn = fn
	voiceCueHandlerMu.Unlock()

}

// SetVoiceCaptureDuckHandler registers music duck/restore for voice command capture.
func SetVoiceCaptureDuckHandler(fn VoiceCaptureDuckHandler) {

	voiceCaptureDuckHandlerMu.Lock()
	voiceCaptureDuckHandlerFn = fn
	voiceCaptureDuckHandlerMu.Unlock()

}

func emitVoiceCue(GuildID snowflake.ID, Kind VoiceCueKind) {

	voiceCueHandlerMu.RLock()
	fn := voiceCueHandlerFn
	voiceCueHandlerMu.RUnlock()

	if fn != nil {

		fn(GuildID, Kind)

	}

}

func emitCaptureDuck(GuildID snowflake.ID, Start bool) {

	voiceCaptureDuckHandlerMu.RLock()
	fn := voiceCaptureDuckHandlerFn
	voiceCaptureDuckHandlerMu.RUnlock()

	if fn != nil {

		fn(GuildID, Start)

	}

}
