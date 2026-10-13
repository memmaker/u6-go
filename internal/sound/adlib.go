package sound

// The .M song player of U6ADLIB.DRV (an LZW-packed 1.6 KB driver). The game calls it through
// a jump table at the start of the driver: init (port), play (song), stop, silence. Play
// hooks the timer interrupt at 60 Hz; each tick runs the song's byte code until a delay, then
// steps the per-channel effects (frequency slides, vibrato, volume slides).
//
// A .M song is byte code. The high nibble of a command byte is the command, the low nibble
// the channel (0-8) it acts on:
//
//	0 note   n   key off at note n            8 flow  (low nibble selects, see below)
//	1 note   n   key on note n (retriggered)  9-13    nothing
//	2 note   n   key on note n (tied)         14      mark the song's loop point
//	3 level  v   carrier level                15      return from a subsong, or loop
//	4 level  v   modulator level
//	5 slide  s   signed frequency slide per tick
//	6 vibr   dm  vibrato: depth (high nibble), multiplier (low nibble)
//	7 instr  i   load instrument i
//
//	8.1 count, offset  call the subsong at offset count times
//	8.2 ticks          wait
//	8.3 i, 11 bytes    define instrument i
//	8.5 cs             volume slide up on channel c every s+1 ticks
//	8.6 cs             volume slide down on channel c every s+1 ticks
//
// A note byte holds the octave (block) in its top 3 bits and a pitch index in the low 5.

const (
	driverTickHz     = 60 // the timer is reprogrammed to 1193182 / 19886 Hz
	maxSubsongDepth  = 16
	regKeyboardSplit = 8  // composite sine mode and note select
	instrumentSize   = 11 // 5 modulator registers, 5 carrier registers, feedback
	maxLevel         = 63
	keyOnBit         = 1 << 13 // in the channel's frequency word: bit 5 of register B0
	blockShift       = 10      // the block (octave) sits above the 10-bit frequency number
	noteOctaveShift  = 5       // a note byte's octave sits above its 5-bit pitch index
	pitchesPerNote   = 32
	commandShift     = 4
	vibratoMultShift = 4
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

// The operator register offsets of each channel's modulator and carrier.
var (
	modulatorOffset = [channels]int{0, 1, 2, 8, 9, 10, 16, 17, 18}
	carrierOffset   = [channels]int{3, 4, 5, 11, 12, 13, 19, 20, 21}
)

// fnumTable is the driver's frequency numbers for the pitch index of a note byte: rests and
// C D E F G A B, then the same sharp and flat.
var fnumTable = [...]int{
	0, 344, 386, 432, 460, 515, 577, 646,
	0, 362, 406, 455, 484, 542, 607, 680,
	0, 327, 366, 410, 437, 489, 548, 614,
}

type subsong struct {
	repeats   int
	returnPos int
	start     int
}

type adlibChannel struct {
	freq                        int // the frequency word: fnum, block and key on bit
	carrierLevel                int
	freqSlide                   int
	vibratoDepth                int
	vibratoMult                 int
	vibratoPos                  int
	vibratoDown                 bool
	volumeSlide                 int // +1, -1 or 0
	slideCountdown, slidePeriod int
}

// songPlayer is the driver's state for one song.
type songPlayer struct {
	chip        *opl2
	song        []byte
	pos         int
	loopPos     int
	delay       int
	stack       []subsong
	instruments [256]int // position of each instrument's definition
	ch          [channels]adlibChannel
}

// C: U6ADLIB.DRV:0320 (init) — enable waveform select, clear rhythm and the CSM bits.
func newSongPlayer(song []byte) *songPlayer {
	p := &songPlayer{chip: newOPL2(), song: song}
	p.chip.write(regTest, bit5)
	p.chip.write(regRhythm, 0)
	p.chip.write(regKeyboardSplit, 0)
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

// noteFreq turns a note byte into a frequency word without the key on bit.
// C: U6ADLIB.DRV:030B
func noteFreq(note int) int {
	pitch := note % pitchesPerNote
	fnum := 0
	if pitch < len(fnumTable) { // fix: the driver reads its jump table for indexes 24-31
		fnum = fnumTable[pitch]
	}
	return fnum + note>>noteOctaveShift<<blockShift
}

// C: U6ADLIB.DRV:0274
func (p *songPlayer) writeFreq(c, freq int) {
	p.chip.write(regFreqLow+c, byte(freq%256))
	p.chip.write(regFreqHigh+c, byte(freq/256))
}

// C: U6ADLIB.DRV:026C
func (p *songPlayer) writeCarrierLevel(c int) {
	p.chip.write(regOpLevel+carrierOffset[c], byte(p.ch[c].carrierLevel))
}

// C: U6ADLIB.DRV:0287
func (p *songPlayer) loadInstrument(c, at int) {
	regs := [...]int{regOpFlags, regOpLevel, regOpAttackDecay, regOpSustain, regWaveform}
	for _, op := range [...]int{modulatorOffset[c], carrierOffset[c]} {
		for _, r := range regs {
			p.chip.write(r+op, p.song[at])
			at++
		}
	}
	p.chip.write(regFeedback+c, p.song[at])
}

// tick is one 60 Hz timer interrupt.
// C: U6ADLIB.DRV:0367
func (p *songPlayer) tick() {
	if p.delay != 0 {
		p.delay--
	}
	if p.delay == 0 {
		p.run()
	}
	p.effects()
}

// run interprets the song until a delay.
func (p *songPlayer) run() {
	for {
		cmd := p.next()
		c := cmd % 16
		switch cmd >> commandShift {
		case cmdKeyOff:
			if c >= channels {
				p.next()
				continue
			}
			p.ch[c].freq = noteFreq(p.next())
			p.writeFreq(c, p.ch[c].freq)
		case cmdKeyOn:
			if c >= channels {
				p.next()
				continue
			}
			ch := &p.ch[c]
			ch.vibratoDown, ch.vibratoPos = false, 0
			freq := noteFreq(p.next())
			p.writeFreq(c, freq)
			ch.freq = freq | keyOnBit
			p.writeFreq(c, ch.freq)
		case cmdTie:
			if c >= channels {
				p.next()
				continue
			}
			p.ch[c].freq = noteFreq(p.next()) | keyOnBit
			p.writeFreq(c, p.ch[c].freq)
		case cmdCarrierLevel:
			v := p.next()
			if c < channels {
				p.ch[c].volumeSlide = 0
				p.ch[c].carrierLevel = v
				p.writeCarrierLevel(c)
			}
		case cmdModulatorLevel:
			v := p.next()
			if c < channels {
				p.chip.write(regOpLevel+modulatorOffset[c], byte(v))
			}
		case cmdFreqSlide:
			v := int(int8(p.next()))
			if c < channels {
				p.ch[c].freqSlide = v
			}
		case cmdVibrato:
			v := p.next()
			if c < channels {
				p.ch[c].vibratoDepth, p.ch[c].vibratoMult = v>>vibratoMultShift, v%16
			}
		case cmdInstrument:
			i := p.next()
			if c < channels && p.instruments[i]+instrumentSize <= len(p.song) {
				p.loadInstrument(c, p.instruments[i])
			}
		case cmdFlow:
			if p.flow(c) {
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
		c := v >> commandShift
		if c >= channels {
			return false
		}
		ch := &p.ch[c]
		ch.volumeSlide = 1
		if sub == flowVolumeSlideDown {
			ch.volumeSlide = -1
		}
		ch.slidePeriod = v%16 + 1
		ch.slideCountdown = ch.slidePeriod
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

// effects steps each channel's slides and vibrato, channel 8 down to 0.
// C: U6ADLIB.DRV:0568
func (p *songPlayer) effects() {
	for c := channels - 1; c >= 0; c-- {
		ch := &p.ch[c]
		if ch.freqSlide != 0 {
			ch.freq = (ch.freq + ch.freqSlide) & (1<<16 - 1)
			p.writeFreq(c, ch.freq)
		} else if ch.vibratoMult != 0 && ch.freq&keyOnBit != 0 {
			if ch.vibratoPos >= ch.vibratoDepth {
				ch.vibratoDown = true
			} else if ch.vibratoPos <= 0 {
				ch.vibratoDown = false
			}
			if ch.vibratoDown {
				ch.vibratoPos--
			} else {
				ch.vibratoPos++
			}
			bend := (ch.vibratoPos - ch.vibratoDepth/2) * ch.vibratoMult
			p.writeFreq(c, (ch.freq+bend)&(1<<16-1))
		}
		if ch.volumeSlide != 0 {
			ch.slideCountdown--
			if ch.slideCountdown == 0 {
				ch.slideCountdown = ch.slidePeriod
				level := ch.carrierLevel + ch.volumeSlide
				if level > maxLevel {
					level, ch.volumeSlide = maxLevel, 0
				} else if level < 0 {
					level, ch.volumeSlide = 0, 0
				}
				ch.carrierLevel = level
				p.writeCarrierLevel(c)
			}
		}
	}
}
