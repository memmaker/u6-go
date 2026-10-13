package sound

import (
	"io/fs"
	"strconv"
	"strings"
)

// The MT-32 side of the .M player (song.go): U6ROLAND.DRV, which plays the same songs as
// MIDI through an MPU-401, on munt's MT-32 emulator with the ROMs the player supplies
// (mt32_dl.go on the desktop, the page's goplayMT32 in the browser, mt32_js.go). The game
// maps each of the song's 16 instruments to a MIDI channel and program from MIDI.DAT before
// playing it. A binding provides mtOpen(romDir), mtMsg, mtSysex and mtRender.

const (
	midiVoices      = 16
	midiNoteOff     = 0x80
	midiNoteOn      = 0x90
	midiProgram     = 0xC0
	midiControl     = 0xB0
	midiAllNotesOff = 0x7B
	midiMaxVelocity = 127
	defaultChannel  = 1    // MIDI channel 2, the MT-32's first melodic part
	noProgram       = -1   // the driver's 0xFF: leave the channel's program alone
	midiDataSize    = 1024 // MUS_0851 reads this much of MIDI.DAT
	semitones       = 12
)

// midiNoteTable is the driver's MIDI note in octave 0 for the pitch index of a note byte.
var midiNoteTable = [...]int{
	0, 12, 14, 16, 17, 19, 21, 23,
	0, 13, 15, 17, 18, 20, 22, 24,
	0, 11, 13, 15, 16, 18, 20, 22,
}

// instrumentMap is what the driver's "set instrument" entry stores for the 16 instruments.
type instrumentMap struct {
	channel [midiVoices]int
	program [midiVoices]int
}

type rolandVoice struct {
	channel  int
	note     int // the note sounding, -1: none
	pending  int // the note to start on the next tick, -1: none
	velocity int
}

type roland struct {
	instruments instrumentMap
	voice       [midiVoices]rolandVoice
}

// C: U6ROLAND.DRV (play) — every voice on channel 2 at full velocity.
func newRoland(m instrumentMap) *roland {
	r := &roland{instruments: m}
	for v := range r.voice {
		r.voice[v] = rolandVoice{channel: defaultChannel, note: -1, pending: -1, velocity: midiMaxVelocity}
	}
	return r
}

func midiNote(note int) int {
	pitch := note % pitchesPerNote
	n := 0
	if pitch < len(midiNoteTable) { // fix: the driver reads past its table for indexes 24-31
		n = midiNoteTable[pitch]
	}
	return n + semitones*(note>>noteOctaveShift)
}

func midiMessage(status, d1, d2 int) {
	mtMsg(uint32(status) | uint32(d1)<<8 | uint32(d2)<<16)
}

func (r *roland) release(v *rolandVoice) {
	if v.note >= 0 {
		midiMessage(midiNoteOff|v.channel, v.note, v.velocity)
		v.note = -1
	}
}

func (r *roland) noteOff(voice, note int) {
	v := &r.voice[voice]
	midiMessage(midiNoteOff|v.channel, midiNote(note), v.velocity)
	v.note, v.pending = -1, -1
}

// noteOn ends the voice's note and starts this one with the next tick.
func (r *roland) noteOn(voice, note int) {
	v := &r.voice[voice]
	r.release(v)
	v.pending = midiNote(note)
	v.note = v.pending
}

func (r *roland) tie(voice, note int) { r.noteOn(voice, note) }

// carrierLevel: the OPL attenuation (0 loudest, 63 silent) as a velocity.
func (r *roland) carrierLevel(voice, level int) {
	inverted := maxLevel - level%(maxLevel+1)
	r.voice[voice].velocity = inverted * 3 / 2 % (midiMaxVelocity + 1)
}

func (r *roland) modulatorLevel(voice, level int)    {}
func (r *roland) freqSlide(voice, slide int)         {}
func (r *roland) vibrato(voice, depthAndMult int)    {}
func (r *roland) volumeSlide(voice, dir, period int) {}

// instrument moves the voice to the instrument's channel and selects its program there
// (the driver's check against the channel's last program never matches, so it always does).
func (r *roland) instrument(voice, number, at int) {
	v := &r.voice[voice]
	i := number % midiVoices
	channel := r.instruments.channel[i]
	if channel != v.channel {
		r.release(v)
	}
	v.channel = channel
	if p := r.instruments.program[i]; p != noProgram {
		midiMessage(midiProgram|channel, p, 0)
	}
}

// startTick sends the notes started on the last tick.
func (r *roland) startTick() {
	for i := range r.voice {
		v := &r.voice[i]
		if v.pending >= 0 {
			midiMessage(midiNoteOn|v.channel, v.pending, v.velocity)
			v.pending = -1
		}
	}
}

func (r *roland) endTick() {}

// stop: C: U6ROLAND.DRV (silence) — end every note.
func (r *roland) stop() {
	for i := range r.voice {
		r.release(&r.voice[i])
		r.voice[i].pending = -1
	}
	for c := range midiVoices {
		midiMessage(midiControl|c, midiAllNotesOff, 0)
	}
}

// loadInstrumentMap reads song number n's (1-based) instrument map from MIDI.DAT: the n-th
// group "(channel/program,...)". Without the file every instrument plays on channel 2 with
// its program unchanged.
// C: MUS_0851, MUS_07F6
func loadInstrumentMap(fsys fs.FS, n int) instrumentMap {
	var m instrumentMap
	for i := range midiVoices {
		m.channel[i], m.program[i] = defaultChannel, noProgram
	}
	text, err := fs.ReadFile(fsys, "MIDI.DAT")
	if err != nil {
		text, err = fs.ReadFile(fsys, "midi.dat")
	}
	if err != nil {
		return m
	}
	text = text[:min(len(text), midiDataSize)]
	groups := strings.Split(string(text), "(")
	if n < 1 || n >= len(groups) {
		return m
	}
	group, _, _ := strings.Cut(groups[n], ")")
	numbers := strings.FieldsFunc(group, func(r rune) bool { return r < '0' || r > '9' })
	for i := 0; i+1 < len(numbers) && i/2 < midiVoices; i += 2 {
		channel, _ := strconv.Atoi(numbers[i])
		program, _ := strconv.Atoi(numbers[i+1])
		m.channel[i/2] = (channel + midiVoices - 1) % midiVoices // the driver keeps the low nibble
		m.program[i/2] = program - 1                             // 0 becomes noProgram
	}
	return m
}
