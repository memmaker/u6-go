package sound

// The .M song player shared by the music drivers (U6ADLIB.DRV, U6ROLAND.DRV, ...). The game
// calls a driver through a jump table at its start: init (port), play (song), stop, silence,
// and on MIDI devices "set instrument" (MIDI.DAT). Play hooks the timer interrupt at 60 Hz;
// each tick runs the song's byte code until a delay.
//
// A .M song is byte code. The high nibble of a command byte is the command, the low nibble
// the voice (0-8) it acts on:
//
//	0 note   n   note off at note n           8 flow  (low nibble selects, see below)
//	1 note   n   note on n (retriggered)      9-13    nothing
//	2 note   n   note on n (tied)             14      mark the song's loop point
//	3 level  v   carrier level (volume)       15      return from a subsong, or loop
//	4 level  v   modulator level
//	5 slide  s   signed frequency slide per tick
//	6 vibr   dm  vibrato: depth (high nibble), multiplier (low nibble)
//	7 instr  i   select instrument i
//
//	8.1 count, offset  call the subsong at offset count times
//	8.2 ticks          wait
//	8.3 i, 11 bytes    define (AdLib) instrument i
//	8.5 cs             volume slide up on voice c every s+1 ticks
//	8.6 cs             volume slide down on voice c every s+1 ticks
//
// A note byte holds the octave in its top 3 bits and a pitch index in the low 5.

const (
	driverTickHz    = 60 // the timer is reprogrammed to 1193182 / 19886 Hz
	maxSubsongDepth = 16
	noteOctaveShift = 5 // a note byte's octave sits above its 5-bit pitch index
	pitchesPerNote  = 32
	commandShift    = 4
	instrumentCount = 256
)

const (
	cmdKeyOff = iota
	cmdKeyOn
	cmdTie
	cmdCarrierLevel
	cmdModulatorLevel
	cmdFreqSlide
	cmdVibrato
	cmdInstrument
	cmdFlow
	cmdLoopPoint = 14
	cmdReturn    = 15
)

const (
	flowCall             = 1
	flowDelay            = 2
	flowDefineInstrument = 3
	flowVolumeSlideUp    = 5
	flowVolumeSlideDown  = 6
)

// musicDriver is what a music driver does with the song's commands.
type musicDriver interface {
	noteOff(voice, note int)
	noteOn(voice, note int)
	tie(voice, note int)
	carrierLevel(voice, level int)
	modulatorLevel(voice, level int)
	freqSlide(voice, slide int)
	vibrato(voice, depthAndMultiplier int)
	// instrument selects instrument number on the voice; at is where the song defined it
	// (-1: not defined).
	instrument(voice, number, at int)
	// volumeSlide starts a volume slide by direction (+1 or -1) every period ticks.
	volumeSlide(voice, direction, period int)
	startTick() // before the tick's commands
	endTick()   // after them
	stop()
}

type subsong struct {
	repeats   int
	returnPos int
	start     int
}

// songPlayer is the drivers' common state for one song.
type songPlayer struct {
	driver      musicDriver
	song        []byte
	pos         int
	loopPos     int
	delay       int
	stack       []subsong
	instruments [instrumentCount]int // position of each instrument's definition, -1: none
}

func newPlayer(song []byte, driver musicDriver) *songPlayer {
	p := &songPlayer{driver: driver, song: song}
	for i := range p.instruments {
		p.instruments[i] = -1
	}
	return p
}

func (p *songPlayer) next() int {
	if p.pos >= len(p.song) { // fix: a truncated song loops instead of reading past its end
		p.pos = p.loopPos
		p.stack = p.stack[:0]
		return cmdFlow<<commandShift | flowDelay
	}
	b := p.song[p.pos]
	p.pos++
	return int(b)
}

// tick is one 60 Hz timer interrupt.
// C: U6ADLIB.DRV:0367, U6ROLAND.DRV:039D
func (p *songPlayer) tick() {
	p.driver.startTick()
	if p.delay != 0 {
		p.delay--
	}
	if p.delay == 0 {
		p.run()
	}
	p.driver.endTick()
}

// run interprets the song until a delay.
func (p *songPlayer) run() {
	d := p.driver
	for {
		cmd := p.next()
		voice := cmd % 16
		switch cmd >> commandShift {
		case cmdKeyOff:
			d.noteOff(voice, p.next())
		case cmdKeyOn:
			d.noteOn(voice, p.next())
		case cmdTie:
			d.tie(voice, p.next())
		case cmdCarrierLevel:
			d.carrierLevel(voice, p.next())
		case cmdModulatorLevel:
			d.modulatorLevel(voice, p.next())
		case cmdFreqSlide:
			d.freqSlide(voice, int(int8(p.next())))
		case cmdVibrato:
			d.vibrato(voice, p.next())
		case cmdInstrument:
			i := p.next()
			d.instrument(voice, i, p.instruments[i])
		case cmdFlow:
			if p.flow(voice) {
				return
			}
		case cmdLoopPoint:
			p.loopPos = p.pos
		case cmdReturn:
			p.ret()
		}
	}
}

// flow runs a flow command and reports whether it was a delay, which ends the tick.
func (p *songPlayer) flow(sub int) bool {
	switch sub {
	case flowCall:
		if len(p.stack) >= maxSubsongDepth {
			return false
		}
		repeats := p.next()
		offset := p.next() + p.next()*256
		p.stack = append(p.stack, subsong{repeats: repeats, returnPos: p.pos, start: offset})
		p.pos = offset
	case flowDelay:
		p.delay = p.next() // 0 waits one tick, as 1 does
		return true
	case flowDefineInstrument:
		i := p.next()
		p.instruments[i] = p.pos
		p.pos += instrumentSize
	case flowVolumeSlideUp, flowVolumeSlideDown:
		v := p.next()
		direction := 1
		if sub == flowVolumeSlideDown {
			direction = -1
		}
		p.driver.volumeSlide(v>>commandShift, direction, v%16+1)
	}
	return false
}

// C: U6ADLIB.DRV:053C
func (p *songPlayer) ret() {
	if len(p.stack) == 0 {
		p.pos = p.loopPos
		return
	}
	top := &p.stack[len(p.stack)-1]
	top.repeats--
	if top.repeats != 0 { // the driver's count of 0 wraps to 65535 more repeats
		p.pos = top.start
		return
	}
	p.pos = top.returnPos
	p.stack = p.stack[:len(p.stack)-1]
}
