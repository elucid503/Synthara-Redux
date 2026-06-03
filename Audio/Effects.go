//go:build linux || darwin || windows
// +build linux darwin windows

package Audio

import (
	"math"
	"sync"
	"sync/atomic"
)

// EffectsProcessor holds per-session speed/reverb state for the mixer.
type EffectsProcessor struct {

	SpeedMilli atomic.Int32
	ReverbPercent atomic.Int32

	mu sync.Mutex

	reverb roomReverb

}

const (

	reverbInputHPHz = 140.0 // keeps mud/vocals mostly dry
	reverbWetLPHz = 11000.0
	reverbPreDelayMs = 28
	reverbCombFb = 0.88 // base feedback; scaled up with amount

	reverbWetLevelMax = 0.55 // Equal-power wet/dry at 100%: dry ~ 0.71, wet ~ 0.55 (audible space, not washed out)

)

func NewEffectsProcessor() *EffectsProcessor {

	processor := &EffectsProcessor{}

	processor.SpeedMilli.Store(1000)
	processor.ReverbPercent.Store(0)
	processor.reverb.init()

	return processor

}

func (effects *EffectsProcessor) SetSpeedMilli(speedMilli int) {

	effects.SpeedMilli.Store(int32(speedMilli))

}

func (effects *EffectsProcessor) SetReverbPercent(percent int) {

	old := effects.ReverbPercent.Swap(int32(percent))

	if (old == 0) != (int32(percent) == 0) {

		effects.mu.Lock()
		effects.reverb.reset()
		effects.mu.Unlock()

	}

}

func (effects *EffectsProcessor) SpeedRatio() float64 {

	ratio := float64(effects.SpeedMilli.Load()) / 1000.0

	if ratio <= 0 {

		return 1

	}

	return ratio

}

func (effects *EffectsProcessor) ApplyReverb(frame []float32) {

	if effects.ReverbPercent.Load() <= 0 {

		return

	}

	effects.mu.Lock()
	defer effects.mu.Unlock()

	mix := float32(effects.ReverbPercent.Load()) / 100.0

	dryGain := float32(math.Cos(float64(mix) * math.Pi / 2))
	wetGain := float32(math.Sin(float64(mix)*math.Pi/2)) * reverbWetLevelMax

	feedback := reverbCombFb + mix * 0.08 // longer tail at higher settings

	effects.reverb.setFeedback(feedback)
	effects.reverb.processFrame(frame, dryGain, wetGain)

	limitFramePeak(frame, 0.99)

}

// limitFramePeak scales the whole frame down if any sample clips (no harmonic distortion).
func limitFramePeak(frame []float32, ceiling float32) {

	peak := float32(0)

	for _, sample := range frame {

		abs := sample

		if abs < 0 {

			abs = -abs

		}

		if abs > peak {

			peak = abs

		}

	}

	if peak <= ceiling || peak == 0 {

		return

	}

	scale := ceiling / peak

	for i := range frame {

		frame[i] *= scale

	}

}

type biquad struct {

	b0, b1, b2, a1, a2 float32
	z1, z2 float32

}

func (filter *biquad) process(sample float32) float32 {

	out := filter.b0*sample + filter.z1

	filter.z1 = filter.b1 * sample - filter.a1 * out + filter.z2
	filter.z2 = filter.b2 * sample - filter.a2 * out

	return out

}

func makeHighPassBiquad(sampleRate int, freqHz, q float32) biquad {

	w0 := 2 * math.Pi * float64(freqHz) / float64(sampleRate)

	cosW0 := float32(math.Cos(w0))
	sinW0 := float32(math.Sin(w0))
	alpha := sinW0 / (2 * q)

	b0 := (1 + cosW0) / 2
	b1 := -(1 + cosW0)
	b2 := b0
	a0 := 1 + alpha
	a1 := -2 * cosW0
	a2 := 1 - alpha

	invA0 := 1 / a0

	return biquad{

		b0: b0 * invA0,
		b1: b1 * invA0,
		b2: b2 * invA0,
		a1: a1 * invA0,
		a2: a2 * invA0,

	}

}

func makeLowPassBiquad(sampleRate int, freqHz, q float32) biquad {

	w0 := 2 * math.Pi * float64(freqHz) / float64(sampleRate)

	cosW0 := float32(math.Cos(w0))
	sinW0 := float32(math.Sin(w0))
	alpha := sinW0 / (2 * q)

	b1 := 1 - cosW0
	b0 := b1 / 2
	b2 := b0
	a0 := 1 + alpha
	a1 := -2 * cosW0
	a2 := 1 - alpha

	invA0 := 1 / a0

	return biquad{

		b0: b0 * invA0,
		b1: b1 * invA0,
		b2: b2 * invA0,
		a1: a1 * invA0,
		a2: a2 * invA0,

	}

}

type delayLine struct {

	buf []float32
	idx int

}

func (delay *delayLine) process(sample float32) float32 {

	if len(delay.buf) == 0 {
		return sample
	}

	out := delay.buf[delay.idx]
	delay.buf[delay.idx] = sample

	delay.idx++

	if delay.idx >= len(delay.buf) {
		delay.idx = 0
	}

	return out

}

func (delay *delayLine) clear() {

	for i := range delay.buf {
		delay.buf[i] = 0
	}

	delay.idx = 0

}

type feedbackComb struct {

	buf []float32
	idx int

	feedback float32

}

func (comb *feedbackComb) process(sample float32) float32 {

	out := comb.buf[comb.idx]
	comb.buf[comb.idx] = sample + out*comb.feedback

	comb.idx++

	if comb.idx >= len(comb.buf) {

		comb.idx = 0

	}

	return out

}

func (comb *feedbackComb) clear() {

	for i := range comb.buf {

		comb.buf[i] = 0

	}

	comb.idx = 0

}

func (comb *feedbackComb) setFeedback(fb float32) {

	if fb < 0.5 {

		fb = 0.5

	}

	if fb > 0.97 {

		fb = 0.97

	}

	comb.feedback = fb

}

// roomReverb is a stereo send/return reverb with filtered wet and explicit dry/wet mix.
type roomReverb struct {

	preL, preR delayLine
	combsL [4]feedbackComb
	combsR [4]feedbackComb

	inputHPF [Channels]biquad
	wetLPF [Channels]biquad

}

// Comb delay lengths in samples at 48 kHz
var combDelaySamples48k = [4]int{

	1807, 2251, 2833, 3527,

}

func (room *roomReverb) init() {

	preLen := SampleRate * reverbPreDelayMs / 1000

	room.preL = delayLine{buf: make([]float32, preLen)}
	room.preR = delayLine{buf: make([]float32, preLen)}

	for i, base := range combDelaySamples48k {

		scaled := base * SampleRate / 48000

		if scaled < 1 {
			scaled = 1
		}

		room.combsL[i] = feedbackComb{buf: make([]float32, scaled), feedback: reverbCombFb}
		room.combsR[i] = feedbackComb{buf: make([]float32, scaled+17), feedback: reverbCombFb}

	}

	hpf := makeHighPassBiquad(SampleRate, reverbInputHPHz, 0.707)
	lpf := makeLowPassBiquad(SampleRate, reverbWetLPHz, 0.707)

	for ch := range room.inputHPF {
		room.inputHPF[ch] = hpf
		room.wetLPF[ch] = lpf
	}

}

func (room *roomReverb) setFeedback(fb float32) {

	for i := range room.combsL {
		room.combsL[i].setFeedback(fb)
		room.combsR[i].setFeedback(fb)
	}

}

func (room *roomReverb) reset() {

	room.preL.clear()
	room.preR.clear()

	for i := range room.combsL {

		room.combsL[i].clear()
		room.combsR[i].clear()

	}

	for ch := range room.inputHPF {

		room.inputHPF[ch].z1 = 0
		room.inputHPF[ch].z2 = 0
		room.wetLPF[ch].z1 = 0
		room.wetLPF[ch].z2 = 0

	}

}

func (room *roomReverb) wetSample(leftIn, rightIn float32) (float32, float32) {

	sendL := room.inputHPF[0].process(leftIn)
	sendR := room.inputHPF[1].process(rightIn)

	sendL = room.preL.process(sendL)
	sendR = room.preR.process(sendR)

	var wetL, wetR float32

	for i := range room.combsL {

		wetL += room.combsL[i].process(sendL)
		wetR += room.combsR[i].process(sendR)

	}

	wetL *= 0.22
	wetR *= 0.22

	wetL = room.wetLPF[0].process(wetL)
	wetR = room.wetLPF[1].process(wetR)

	return wetL, wetR

}

func (room *roomReverb) processFrame(pcm []float32, dryGain, wetGain float32) {

	for frameIdx := 0; frameIdx < FrameSize; frameIdx++ {

		idx := frameIdx * Channels

		dryL := pcm[idx]
		dryR := pcm[idx+1]

		wetL, wetR := room.wetSample(dryL, dryR)

		pcm[idx] = dryL*dryGain + wetL*wetGain
		pcm[idx+1] = dryR*dryGain + wetR*wetGain

	}

}

func floatToPCM(in []float32, out []int16) {

	for i, v := range in {

		if v > 1 {

			v = 1

		} else if v < -1 {

			v = -1

		}

		out[i] = int16(v * 32767)

	}

}

func clearFloats(buf []float32) {

	for i := range buf {

		buf[i] = 0

	}

}
