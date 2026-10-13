package game

import (
	"time"

	"github.com/memmaker/u6-go/internal/host"
)

// OSILIB, the Origin Systems library the game is linked with: random numbers, delays and
// the keyboard.

// randomState is the 32-bit seed of OSI_rand (D_3088, D_308A).
type randomState uint32

// next advances the seed: seed*3 + (low half << 16).
// C: OSILIB RAND.ASM C_33C5_000C
func (r *randomState) next() uint16 {
	seed := uint32(*r)
	*r = randomState(seed*3 + seed<<16)
	return uint16(*r >> 16)
}

// rand returns a random number between a and b (inclusive, in either order).
// C: OSILIB RAND.ASM OSI_rand
func (g *Game) rand(a, b int) int {
	high := g.random.next()
	lo, hi := uint16(a), uint16(b)
	if lo > hi {
		lo, hi = hi, lo
	}
	span := hi - lo + 1
	if span == 0 {
		return int(lo)
	}
	return int(high%span + lo)
}

// randomize seeds the generator.
// C: OSILIB RAND.ASM OSI_randomize
func (r *randomState) randomize(v int) {
	*r = randomState(uint32(*r)&^(1<<16-1) | uint32(v|1)&(1<<16-1))
}

// timeSeed is the seed made from the time of day.
// C: OSILIB RAND.ASM C_33CA_000E
func timeSeed(t time.Time) int {
	const (
		scramble = 37355 // 0x91eb
		seedMask = 4095
	)
	seconds, hundredths := t.Second(), t.Nanosecond()/int(10*time.Millisecond)
	v := seconds*8 + hundredths + t.Hour()*2 + t.Minute()*4
	return (v ^ scramble) & seedMask
}

// The BIOS timer (int 1Ch) ticks 18.2 times a second.
const (
	pitHz           = 1193182
	pitTicksPerTick = 65536
)

// biosTicks is the number of BIOS timer ticks since the host started.
func biosTicks() int {
	return int(host.Ticks * pitHz / (pitTicksPerTick * host.TicksPerSecond))
}

// osiDelay waits until n timer ticks have passed since the last delay ended.
// C: OSILIB DELAY.ASM OSI_delay
func (g *Game) osiDelay(n int) {
	for biosTicks()-g.osiTicks < n {
		host.Yield()
	}
	g.osiTicks = biosTicks()
}

// Keys as the keyboard function returns them: the character, or for keys that type none
// (and backspace and the keypad keys but '-' and '+') extendedKey + the scan code.
const (
	extendedKey  = 256
	keyEscape    = 27
	keyBackspace = extendedKey + host.ScanBackspace
	keyUp        = extendedKey + host.ScanUp
	keyDown      = extendedKey + host.ScanDown
	keyLeft      = extendedKey + host.ScanLeft
	keyRight     = extendedKey + host.ScanRight
	keyHome      = extendedKey + host.ScanHome
	keyPageUp    = extendedKey + host.ScanPageUp
	keyEnd       = extendedKey + host.ScanEnd
	keyPageDown  = extendedKey + host.ScanPageDown
	keyF1        = extendedKey + host.ScanF1
	keyF10       = extendedKey + host.ScanF10
	keyF11       = extendedKey + host.ScanF11
	keyF12       = extendedKey + host.ScanF12

	scanKeypadFirst = host.ScanHome
	scanKeypadLast  = host.ScanPageDown
	scanKeypadMinus = 74
	scanKeypadPlus  = 78
)

// readKey returns the next key press, or 0 when none is waiting. With drain it reads all
// waiting presses and returns the last.
// C: OSILIB KBD.ASM C_31FA_000C
func readKey(drain bool) int {
	if len(host.KeyQueue) == 0 {
		return 0
	}
	var k host.Key
	if drain {
		k = host.KeyQueue[len(host.KeyQueue)-1]
		host.KeyQueue = host.KeyQueue[:0]
	} else {
		k = host.KeyQueue[0]
		host.KeyQueue = host.KeyQueue[1:]
	}
	// alt+letter types no character
	if host.AltDown && (k.Char >= 'a' && k.Char <= 'z' || k.Char >= 'A' && k.Char <= 'Z') {
		k.Char = 0
	}
	switch {
	case k.Char == 0:
		if k.Scan == 0 {
			return 0
		}
		return extendedKey + int(k.Scan)
	case k.Scan >= scanKeypadFirst && k.Scan <= scanKeypadLast && k.Scan != scanKeypadMinus && k.Scan != scanKeypadPlus,
		k.Scan == host.ScanBackspace:
		return extendedKey + int(k.Scan)
	}
	return int(k.Char)
}
