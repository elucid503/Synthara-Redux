package Structs

import "fmt"

const (

	DefaultSpeedMilli = 1000
	SpeedStepMilli = 50

	DefaultReverb = 0
	ReverbStep = 15
	MaxReverbPercent = 75

)

var AllowedSpeedMilli = []int{850, 900, 950, 1000, 1050, 1100, 1150}
var AllowedReverbPercent = []int{0, 15, 30, 45, 60, 75}

func ClampSpeedMilli(speedMilli int) int {

	if speedMilli <= 0 {

		return DefaultSpeedMilli

	}

	return nearestAllowed(speedMilli, AllowedSpeedMilli, DefaultSpeedMilli)

}

func ClampReverb(reverb int) int {

	if reverb < 0 {

		return DefaultReverb

	}

	if reverb > MaxReverbPercent {

		reverb = MaxReverbPercent

	}

	return nearestAllowed(reverb, AllowedReverbPercent, DefaultReverb)

}

func nearestAllowed(value int, allowed []int, fallback int) int {

	if len(allowed) == 0 {

		return fallback

	}

	best := allowed[0]
	bestDist := absInt(value - best)

	for _, candidate := range allowed[1:] {

		if dist := absInt(value - candidate); dist < bestDist {

			best = candidate
			bestDist = dist

		}

	}

	return best
}

func absInt(value int) int {

	if value < 0 {

		return -value

	}

	return value

}

func EffectsProcessingEnabled(speedMilli, reverb int) bool {

	return speedMilli != DefaultSpeedMilli || reverb != DefaultReverb

}

func FormatSpeedLabel(speedMilli int) string {

	return fmt.Sprintf("%.2fx", float64(speedMilli)/1000.0)

}

func (guild *Guild) syncPlaybackEffects() {

	if guild.Queue.PlaybackSession == nil {

		return

	}

	effects := guild.Queue.PlaybackSession.Effects

	if effects == nil {

		return

	}

	effects.SetSpeedMilli(guild.Features.SpeedMilli)
	effects.SetReverbPercent(guild.Features.Reverb)
}

func (guild *Guild) SetSpeed(speedMilli int) int {

	guild.Features.SpeedMilli = ClampSpeedMilli(speedMilli)
	guild.syncPlaybackEffects()

	return guild.Features.SpeedMilli

}

func (guild *Guild) SetReverb(reverb int) int {

	guild.Features.Reverb = ClampReverb(reverb)
	guild.syncPlaybackEffects()

	return guild.Features.Reverb

}
