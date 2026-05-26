package Receive

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"Synthara-Redux/Utils"
)

const (

	envWakeSTTFallback = "VOICE_WAKE_STT_FALLBACK"

	envWakeProbeMS = "VOICE_WAKE_PROBE_MS"
	envWakeProbeMinMS  = "VOICE_WAKE_PROBE_MIN_MS"

	defaultWakeProbeInterval = 200 * time.Millisecond
	defaultWakeProbeMinPCM = TargetSampleRate * 2 * 20 / 100 // 0.20s @ 16kHz
	wakeProbeMaxPCM = TargetSampleRate * 2 * 2 // 2s cap

)

func wakeProbeInterval() time.Duration {

	if V := os.Getenv(envWakeProbeMS); V != "" {

		if MS, Err := strconv.Atoi(V); Err == nil && MS >= 80 {

			return time.Duration(MS) * time.Millisecond

		}

	}

	return defaultWakeProbeInterval

}

func wakeProbeMinPCM() int {

	if V := os.Getenv(envWakeProbeMinMS); V != "" {

		if MS, Err := strconv.Atoi(V); Err == nil && MS >= 80 {

			return TargetSampleRate * 2 * MS / 1000

		}

	}

	return defaultWakeProbeMinPCM

}

// sttWakeFallbackEnabled is true unless VOICE_WAKE_STT_FALLBACK=false.
func sttWakeFallbackEnabled() bool {

	return !strings.EqualFold(os.Getenv(envWakeSTTFallback), "false")

}

func (S *Session) appendProbePCM(PCM []int16) {

	Chunk := Int16ToBytesLE(AmplifyInt16ForKWS(PCM))
	S.probePCM = append(S.probePCM, Chunk...)

	if len(S.probePCM) > wakeProbeMaxPCM {

		S.probePCM = S.probePCM[len(S.probePCM)-wakeProbeMaxPCM:]

	}

}

// maybeSTTWakeProbe accumulates speech and periodically asks xAI if the user said Synthara.
func (S *Session) maybeSTTWakeProbe(PCM []int16) {

	if !sttWakeFallbackEnabled() || S.state.Load() != stateListening {

		return

	}

	if !FrameHasSpeechForWake(PCM) {

		return

	}

	S.appendProbePCM(PCM)

	MinPCM := wakeProbeMinPCM()

	if len(S.probePCM) < MinPCM || time.Since(S.lastWakeProbeAt) < wakeProbeInterval() {

		return

	}

	S.runSTTWakeProbeAsync()

}

// flushSTTWakeProbeOnSilence runs one last probe when speech ends (common miss window).
func (S *Session) flushSTTWakeProbeOnSilence() {

	if !sttWakeFallbackEnabled() || S.state.Load() != stateListening {

		return

	}

	if len(S.probePCM) < wakeProbeMinPCM() {

		S.probePCM = nil
		return

	}

	S.runSTTWakeProbeAsync()

}

func (S *Session) runSTTWakeProbeAsync() {

	if !S.wakeProbeBusy.CompareAndSwap(false, true) {

		return

	}

	S.lastWakeProbeAt = time.Now()
	PCMCopy := append([]byte(nil), PreparePCMForSTT(S.probePCM)...)

	go func() {

		defer S.wakeProbeBusy.Store(false)

		Ctx, Cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer Cancel()

		Text, Err := TranscribeRESTWake(Ctx, PCMCopy, TargetSampleRate)

		if Err != nil {

			if os.Getenv("KWS_DEBUG") != "" {

				Utils.Logger.Warn("Receive", fmt.Sprintf("Wake probe STT failed: %s", Err.Error()))

			}

			return

		}

		if !TranscriptHasWakeProbe(Text) {

			return

		}

		select {

		case S.wakeProbeHit <- Text:

		default:

		}

	}()

}

// drainWakeProbeHit handles async STT wake detections on the session worker.
func (S *Session) drainWakeProbeHit(_ string) {

	if S.state.Load() != stateListening {

		return

	}

	S.beginCapture()

}
