package Receive

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

// FeedbackCueKind identifies capture lifecycle audio feedback.
type FeedbackCueKind int

const (
	FeedbackCueCaptureStart FeedbackCueKind = iota
	FeedbackCueCaptureEnd
)

type FeedbackCueHandler func(GuildID snowflake.ID, Kind FeedbackCueKind)

type CaptureDuckHandler func(GuildID snowflake.ID, Duck bool)

type VoiceCommandOptOutChecker func(UserID snowflake.ID) bool

var (
	feedbackCueMu sync.RWMutex
	feedbackCueFn FeedbackCueHandler

	captureDuckMu sync.RWMutex
	captureDuckFn CaptureDuckHandler

	voiceOptOutMu sync.RWMutex
	voiceOptOutFn VoiceCommandOptOutChecker
)

func SetFeedbackCueHandler(fn FeedbackCueHandler) {

	feedbackCueMu.Lock()
	feedbackCueFn = fn
	feedbackCueMu.Unlock()

}

func SetCaptureDuckHandler(fn CaptureDuckHandler) {

	captureDuckMu.Lock()
	captureDuckFn = fn
	captureDuckMu.Unlock()

}

func SetVoiceCommandOptOutChecker(fn VoiceCommandOptOutChecker) {

	voiceOptOutMu.Lock()
	voiceOptOutFn = fn
	voiceOptOutMu.Unlock()

}

func voiceCommandOptOut(UserID snowflake.ID) bool {

	voiceOptOutMu.RLock()
	Fn := voiceOptOutFn
	voiceOptOutMu.RUnlock()

	if Fn == nil {

		return false

	}

	return Fn(UserID)

}

func emitFeedbackCue(GuildID snowflake.ID, Kind FeedbackCueKind) {

	feedbackCueMu.RLock()
	fn := feedbackCueFn
	feedbackCueMu.RUnlock()

	if fn != nil {

		fn(GuildID, Kind)

	}

}

func emitCaptureDuck(GuildID snowflake.ID, Duck bool) {

	captureDuckMu.RLock()
	fn := captureDuckFn
	captureDuckMu.RUnlock()

	if fn != nil {

		fn(GuildID, Duck)

	}

}
