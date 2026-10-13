package sound

// The PC speaker routines of OSISND.LIB (SOUND1-4.ASM, DELAY2.ASM, PLAYNOTE.C) and the game's
// effect dispatcher MUS_0065, rendered as square waves. The originals busy-wait in loops
// calibrated against the timer (OSI's "computer speed", loop passes per 1/750 s); here each
// wait becomes queued samples and the functions return how long their sound lasts, for the
// caller to host.Wait where the original blocked.

import (
	"math/rand/v2"
	"time"
)

const (
	// delayPassesPerSecond is how many passes of the two-instruction delay loop run per
	// second: the speed test counts three-instruction passes, scaled to 750 per second.
	delayPassesPerSecond = 750 * 3 / 2
	pitClockHz           = 1193182 // the PIT input clock; OSI_sound divides it by the frequency
	wavedStepDivisor     = 24      // OSI_playWavedNote waits speed/24 passes a step
	noiseStepShift       = 4       // OSI_playNoise waits speed>>4 passes a step
	speakerAmplitude     = 6000
)

// Effect numbers, as the game passes them to MUS_0065. The names describe the sound.
const (
	EffectSilence     = 0  // speaker off
	EffectNote        = 1  // a 10-unit note of param Hz (blocked move: 311 Hz)
	EffectWavedLow    = 2  // waved note at half of param (instrument)
	EffectWaved       = 3  // waved note at param (instrument)
	EffectNoise       = 4  // a short noise burst
	EffectProjectile  = 5  // flying missile, param the distance
	EffectCrackle     = 6  // sparse random clicks, param the distance
	EffectImpact      = 7  // missile impact, param the pitch (negative: noise)
	EffectRumble      = 8  // low random tones
	EffectSweepShort  = 9  // rising sweep, hit on a monster of class 8
	EffectHiss        = 10 // high noise
	EffectSplatter    = 11 // twenty noise bursts, hit on a monster of class 0
	EffectSweep       = 12 // rising sweep (miss, ring vanishes, monster of class 23)
	EffectFailure     = 13 // the same rising sweep: "Failed", karma loss, refusals
	EffectSuccess     = 14 // a short 2000 Hz note: "Success"
	EffectWhoosh      = 15 // rising noise
	EffectMissile     = 16 // flying missile of another kind, param the distance
	EffectSwing       = 17 // falling slow sweep: a melee attack
	EffectPluck       = 18 // plucked note at half of param (musicians, instruments)
	EffectLowNote     = 19 // a long note at a quarter of param (instrument)
	EffectChime       = 20 // note then decaying waved note at param (instrument)
	EffectFanfare     = 21 // fifty rising waved notes then a long one
	EffectSqueak      = 22 // high sweep: "Squeak!"
	effectDefaultNote = 100
)

// beeper is the speaker: an on/off gate with a tone, turned into samples as time passes.
type beeper struct {
	hz      float64 // 0: speaker off
	phase   float64 // of the tone, in periods
	owed    float64 // fractional sample time carried between waits
	lowpass float64
	pcm     []float32
	noise   uint16 // OSI_playNoise's generator state
}

func newBeeper() *beeper { return &beeper{noise: 30308} }

// wait holds the current state for seconds.
func (b *beeper) wait(seconds float64) {
	n := seconds*outputRate + b.owed
	count := int(n)
	b.owed = n - float64(count)
	for range count {
		target := 0.0
		if b.hz > 0 {
			target = speakerAmplitude
			if b.phase >= 0.5 {
				target = -speakerAmplitude
			}
			b.phase += b.hz / outputRate
			b.phase -= float64(int(b.phase))
		}
		b.lowpass += (target - b.lowpass) / 2 // the cone is no ideal square
		b.pcm = append(b.pcm, float32(b.lowpass/32768))
	}
}

// level holds the cone fully on or off (the waved notes' pulse width modulation).
func (b *beeper) level(on bool, seconds float64) {
	b.hz = 0
	n := seconds*outputRate + b.owed
	count := int(n)
	b.owed = n - float64(count)
	target := -speakerAmplitude / 2.0
	if on {
		target = speakerAmplitude / 2.0
	}
	for range count {
		b.lowpass += (target - b.lowpass) / 2
		b.pcm = append(b.pcm, float32(b.lowpass/32768))
	}
}

// C: OSI_sound (SOUND3.ASM)
func (b *beeper) sound(hz int) {
	if hz <= 0 {
		return
	}
	b.hz = float64(pitClockHz / (pitClockHz / hz)) // the PIT divisor truncates
}

// C: OSI_nosound (SOUND2.ASM)
func (b *beeper) nosound() { b.hz = 0 }

// delayShift is DELAY2.ASM's speed divisor for a scale: it indexes its word table by bytes,
// so odd scales read a high byte of 0.
var delayShiftTable = [...]int{0, 0, 1, 1, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4}

// C: OSI_sDelay (DELAY2.ASM)
func (b *beeper) sDelay(scale, count int) {
	shift := 0
	if scale%2 == 0 && scale/2 < len(delayShiftTable) {
		shift = delayShiftTable[scale/2]
	}
	b.wait(float64(count) / float64(int(1)<<shift) / delayPassesPerSecond)
}

// C: OSI_playNote (PLAYNOTE.C)
func (b *beeper) playNote(hz, duration int) {
	b.sound(hz)
	b.sDelay(1, duration)
	b.nosound()
}

// playNoise plays random tones of 100 to maxHz Hz, step units apiece, for length units.
// C: OSI_playNoise (SOUND4.ASM)
func (b *beeper) playNoise(step, length, maxHz int) {
	if step <= 0 {
		return
	}
	for elapsed := 0; elapsed < length; elapsed += step {
		const scramble = 37448
		b.noise += scramble
		b.noise = b.noise>>3 | b.noise<<13 // rotate right by 3
		b.noise ^= scramble
		b.noise += 17
		span := maxHz - 100 + 1
		if span <= 0 {
			span = 1
		}
		b.sound(int(b.noise)%span + 100)
		b.wait(float64(step) / (1 << noiseStepShift) / delayPassesPerSecond)
	}
	b.nosound()
}

// playWavedNote is a pulse width modulated note: a 16-bit phase accumulator advancing by
// step each of steps steps, the speaker on while it is above a threshold that starts at
// width and moves by widthDelta a step.
// C: OSI_playWavedNote (SOUND1.ASM)
func (b *beeper) playWavedNote(step, stepUnits, steps, width, widthDelta int) {
	stepSeconds := float64(stepUnits) / wavedStepDivisor / delayPassesPerSecond
	var phase, threshold uint16 = 0, uint16(width)
	for range steps {
		phase += uint16(step)
		b.level(phase > threshold, stepSeconds)
		threshold += uint16(widthDelta)
	}
	b.nosound()
}

// slide is a stepped tone from fromHz towards toHz.
// C: MUS_0008
func (b *beeper) slide(fromHz, toHz, stepUnits, length int) {
	if length > 0 {
		inc := (toHz - fromHz) * stepUnits / length
		hz := fromHz
		for elapsed := 0; elapsed < length; elapsed += stepUnits {
			b.sound(hz)
			b.sDelay(1, stepUnits)
			hz += inc
		}
	}
	b.nosound()
}

// randRange is OSI_rand: lo to hi inclusive.
func randRange(lo, hi int) int {
	if hi < lo {
		return lo
	}
	return lo + rand.IntN(hi-lo+1)
}

// randomTones is the shared loop of the crackling effects: forty short steps, each a tone
// pick returns (0: silence).
func (b *beeper) randomTones(pick func() int) {
	for range 40 {
		if hz := pick(); hz > 0 {
			b.sound(hz)
		} else {
			b.nosound()
		}
		b.sDelay(8, 1)
	}
	b.nosound()
}

// effect plays effect kind with its parameter.
// C: MUS_0065
func (b *beeper) effect(kind, param int) {
	switch kind {
	case EffectSilence:
		b.nosound()
	case EffectNote:
		b.playNote(param, 10)
	case EffectNoise:
		b.playNoise(1, 800, 10000)
	case EffectProjectile:
		base := 2000 - param*16
		b.randomTones(func() int {
			if randRange(0, 3) == 0 {
				if hz := randRange(0, 200) + base - 100; hz > 19 {
					return hz
				}
			}
			return 0
		})
	case EffectMissile:
		base := 1000 - param*8
		b.randomTones(func() int {
			if randRange(0, 1) != 0 && base > 20 {
				return randRange(19, randRange(20, base))
			}
			return 0
		})
	case EffectCrackle:
		base := 1000 - param*8
		b.randomTones(func() int {
			if randRange(0, 7) == 0 {
				if hz := randRange(0, base*2) + base; hz > 19 {
					return hz
				}
			}
			return 0
		})
	case EffectImpact:
		if param < 0 {
			b.playNoise(1, 800, 3000)
			return
		}
		b.randomTones(func() int {
			if param > 19 && randRange(0, 7) == 0 {
				return randRange(0, 40) + param
			}
			return 0
		})
	case EffectRumble:
		b.randomTones(func() int { return randRange(19, 200) })
	case EffectSweepShort:
		b.slide(1200, 2000, 1, 40)
	case EffectHiss:
		b.playNoise(1, 7000, 600)
	case EffectSplatter:
		for range 20 {
			b.playNoise(1, 350, randRange(600, 4800))
		}
	case EffectSweep, EffectFailure:
		b.slide(800, 2000, 1, 50)
	case EffectSuccess:
		b.playNote(2000, 10)
	case EffectWhoosh:
		for hz := 2000; hz < 20000; hz += 1000 {
			b.playNoise(40, 120, hz)
		}
	case EffectSwing:
		b.slide(400, 750, 5, 150)
	case EffectChime:
		b.playNote(param, 6)
		b.playWavedNote(param, 1, 700, 30000, -25)
	case EffectWavedLow:
		b.playWavedNote(param/2, 1, 6000, 30000, -5)
	case EffectWaved:
		b.playWavedNote(param, 1, 4000, 15000, -3)
	case EffectPluck:
		b.playWavedNote(param/2, 1, 2000, 20000, -20)
	case EffectLowNote:
		b.playNote(param/4, 200)
	case EffectFanfare:
		for i := range 50 {
			b.playWavedNote(i*16+800, 1, 1000, 10000-i*64, (50-i)*4)
		}
		b.playWavedNote(1600, 1, 8000, 0, 8)
	case EffectSqueak:
		b.slide(5000, 8000, 1, 100)
	default:
		b.playNote(param, effectDefaultNote)
	}
}

// samplesDuration is how long n output samples last.
func samplesDuration(n int) time.Duration {
	return time.Duration(n) * time.Second / outputRate
}
