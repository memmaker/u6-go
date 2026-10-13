package sound

// The AdLib side of the .M player (song.go): U6ADLIB.DRV (an LZW-packed 1.6 KB driver), which
// plays the song's voices on the OPL2's nine channels and steps their frequency slides,
// vibrato and volume slides every tick.

const (
	regKeyboardSplit = 8  // composite sine mode and note select
	instrumentSize   = 11 // 5 modulator registers, 5 carrier registers, feedback
	maxLevel         = 63
	keyOnBit         = 1 << 13 // in the channel's frequency word: bit 5 of register B0
	blockShift       = 10      // the block (octave) sits above the 10-bit frequency number
	vibratoMultShift = 4
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

// adlib is U6ADLIB.DRV's state: the chip and its channels.
type adlib struct {
	chip *opl2
	song []byte // for the instrument definitions
	ch   [channels]adlibChannel
}

// C: U6ADLIB.DRV:0320 (init) — enable waveform select, clear rhythm and the CSM bits.
func newAdlib(song []byte) *adlib {
	a := &adlib{chip: newOPL2(), song: song}
	a.chip.write(regTest, bit5)
	a.chip.write(regRhythm, 0)
	a.chip.write(regKeyboardSplit, 0)
	return a
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
func (a *adlib) writeFreq(c, freq int) {
	a.chip.write(regFreqLow+c, byte(freq%256))
	a.chip.write(regFreqHigh+c, byte(freq/256))
}

// C: U6ADLIB.DRV:026C
func (a *adlib) writeCarrierLevel(c int) {
	a.chip.write(regOpLevel+carrierOffset[c], byte(a.ch[c].carrierLevel))
}

func (a *adlib) noteOff(c, note int) {
	if c < channels {
		a.ch[c].freq = noteFreq(note)
		a.writeFreq(c, a.ch[c].freq)
	}
}

func (a *adlib) noteOn(c, note int) {
	if c >= channels {
		return
	}
	ch := &a.ch[c]
	ch.vibratoDown, ch.vibratoPos = false, 0
	freq := noteFreq(note)
	a.writeFreq(c, freq)
	ch.freq = freq | keyOnBit
	a.writeFreq(c, ch.freq)
}

func (a *adlib) tie(c, note int) {
	if c < channels {
		a.ch[c].freq = noteFreq(note) | keyOnBit
		a.writeFreq(c, a.ch[c].freq)
	}
}

func (a *adlib) carrierLevel(c, level int) {
	if c < channels {
		a.ch[c].volumeSlide = 0
		a.ch[c].carrierLevel = level
		a.writeCarrierLevel(c)
	}
}

func (a *adlib) modulatorLevel(c, level int) {
	if c < channels {
		a.chip.write(regOpLevel+modulatorOffset[c], byte(level))
	}
}

func (a *adlib) freqSlide(c, slide int) {
	if c < channels {
		a.ch[c].freqSlide = slide
	}
}

func (a *adlib) vibrato(c, v int) {
	if c < channels {
		a.ch[c].vibratoDepth, a.ch[c].vibratoMult = v>>vibratoMultShift, v%16
	}
}

// instrument loads the instrument defined at song position at.
// C: U6ADLIB.DRV:0287
func (a *adlib) instrument(c, number, at int) {
	if c >= channels || at < 0 || at+instrumentSize > len(a.song) {
		return
	}
	regs := [...]int{regOpFlags, regOpLevel, regOpAttackDecay, regOpSustain, regWaveform}
	for _, op := range [...]int{modulatorOffset[c], carrierOffset[c]} {
		for _, r := range regs {
			a.chip.write(r+op, a.song[at])
			at++
		}
	}
	a.chip.write(regFeedback+c, a.song[at])
}

func (a *adlib) volumeSlide(c, direction, period int) {
	if c >= channels {
		return
	}
	ch := &a.ch[c]
	ch.volumeSlide = direction
	ch.slidePeriod = period
	ch.slideCountdown = period
}

func (a *adlib) startTick() {}

// endTick steps each channel's slides and vibrato, channel 8 down to 0.
// C: U6ADLIB.DRV:0568
func (a *adlib) endTick() {
	for c := channels - 1; c >= 0; c-- {
		ch := &a.ch[c]
		if ch.freqSlide != 0 {
			ch.freq = (ch.freq + ch.freqSlide) & (1<<16 - 1)
			a.writeFreq(c, ch.freq)
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
			a.writeFreq(c, (ch.freq+bend)&(1<<16-1))
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
				a.writeCarrierLevel(c)
			}
		}
	}
}

// stop: the AdLib driver leaves the chip as it is; the stream stops sampling it.
func (a *adlib) stop() {}
