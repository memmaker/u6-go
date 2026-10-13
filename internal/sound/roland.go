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
	midiVoices      = 16 // the song's voices and instruments the driver keeps tables for
	tickedVoices    = 9  // the voices whose notes the driver's tick starts (0-8)
	midiChannels    = 16
	midiNoteOff     = 128
	midiNoteOn      = 144
	midiProgram     = 192
	midiControl     = 176
	midiAllNotesOff = 123
	midiDataMask    = 127 // a MIDI data byte's 7 bits
	midiMaxVelocity = 127
	defaultChannel  = 1    // MIDI channel 2, the MT-32's first melodic part
	noProgram       = -1   // the driver's 255: leave the channel's program alone
	noNote          = -1   // the driver's 255: no note sounding
	noPending       = 0    // the driver's 0: no note to start (so it never starts MIDI note 0)
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
	note     int // the note sounding, noNote: none
	pending  int // the note to start on the next tick, noPending: none
	velocity int
}

// roland is U6ROLAND.DRV's state. The game loads the driver once and its tables outlive
// the songs: playing a song resets neither the voices nor the instruments MIDI.DAT does not
// list for it.
type roland struct {
	instruments instrumentMap
	voice       [midiVoices]rolandVoice
}

// newRoland is the driver as loaded: every instrument on channel 2 with program 1, every
// voice on channel 2 at full velocity.
// C: U6ROLAND.DRV (data)
func newRoland() *roland {
	r := &roland{}
	for i := range midiVoices {
		r.instruments.channel[i], r.instruments.program[i] = defaultChannel, 0
		r.voice[i] = rolandVoice{channel: defaultChannel, note: noNote, pending: noPending, velocity: midiMaxVelocity}
	}
	return r
}

func midiNote(note int) int {
	pitch := note % pitchesPerNote
	n := 0
	if pitch < len(midiNoteTable) { // fix: the driver reads past its table for indexes 24-31
		n = midiNoteTable[pitch]
	}
	return (n + semitones*(note>>noteOctaveShift)) & midiDataMask
}

// midiOut sends a MIDI message (tests listen in).
var midiOut = func(msg uint32) { mtMsg(msg) }

func midiMessage(status, d1, d2 int) {
	midiOut(uint32(status) | uint32(d1)<<8 | uint32(d2)<<16)
}

// release ends the note sounding on the voice, if any, and with it the note pending.
// C: U6ROLAND.DRV:02B1 (note off)
func (r *roland) release(v *rolandVoice) {
	if v.note == noNote {
		return
	}
	midiMessage(midiNoteOff|v.channel%midiChannels, v.note, v.velocity)
	v.note, v.pending = noNote, noPending
}

// noteOff ends whatever note the voice sounds; the song's note byte is not looked at. A
// note started in this tick is not sounding yet, so it still starts with the next one.
func (r *roland) noteOff(voice, note int) { r.release(&r.voice[voice]) }

// noteOn ends the voice's note and starts this one with the next tick.
// C: U6ROLAND.DRV:043B, 02B1 (note on)
func (r *roland) noteOn(voice, note int) {
	v := &r.voice[voice]
	r.release(v)
	v.pending = midiNote(note)
}

func (r *roland) tie(voice, note int) { r.noteOn(voice, note) }

// carrierLevel: the OPL attenuation (0 loudest, 63 silent) as a velocity.
// C: U6ROLAND.DRV:029E
func (r *roland) carrierLevel(voice, level int) {
	inverted := maxLevel - level%(maxLevel+1)
	r.voice[voice].velocity = inverted * 3 / 2 % (midiMaxVelocity + 1)
}

func (r *roland) modulatorLevel(voice, level int)    {}
func (r *roland) freqSlide(voice, slide int)         {}
func (r *roland) vibrato(voice, depthAndMult int)    {}
func (r *roland) volumeSlide(voice, dir, period int) {}

// instrument moves the voice to the instrument's channel, ending its note there, and
// selects the program (the driver's check against the channel's last program never
// matches, so it always does).
// C: U6ROLAND.DRV:030B
func (r *roland) instrument(voice, number, at int) {
	v := &r.voice[voice]
	i := number % midiVoices
	channel := r.instruments.channel[i]
	if channel != v.channel {
		r.release(v)
	}
	v.channel = channel
	if p := r.instruments.program[i]; p != noProgram {
		midiMessage(midiProgram|channel%midiChannels, p, 0)
	}
}

// startTick sends the notes started on the last tick.
// C: U6ROLAND.DRV:03B9
func (r *roland) startTick() {
	for i := range tickedVoices {
		v := &r.voice[i]
		if v.pending != noPending {
			midiMessage(midiNoteOn|v.channel%midiChannels, v.pending, v.velocity)
			v.note, v.pending = v.pending, noPending
		}
	}
}

func (r *roland) endTick() {}

// stop: the driver has the MPU-401 clear its play map (all notes off) and drops the
// pending notes; it forgets nothing about the notes sounding.
// C: U6ROLAND.DRV:037F (silence)
func (r *roland) stop() {
	for c := range midiChannels {
		midiMessage(midiControl|c, midiAllNotesOff, 0)
	}
	for i := range tickedVoices {
		r.voice[i].pending = noPending
	}
}

// loadInstruments sets the instruments MIDI.DAT lists for song number n (1-based), its n-th
// group "(channel/program,...)"; the others keep what they had. Without the file every
// instrument plays on channel 2 with its program unchanged.
// C: MUS_0851, MUS_07F6
func (m *instrumentMap) load(fsys fs.FS, n int) {
	text, err := fs.ReadFile(fsys, "MIDI.DAT")
	if err != nil {
		text, err = fs.ReadFile(fsys, "midi.dat")
	}
	if err != nil {
		for i := range midiVoices {
			m.channel[i], m.program[i] = defaultChannel, noProgram
		}
		return
	}
	text = text[:min(len(text), midiDataSize)]
	groups := strings.Split(string(text), "(")
	if n < 1 || n >= len(groups) {
		return
	}
	group, _, _ := strings.Cut(groups[n], ")")
	numbers := strings.FieldsFunc(group, func(r rune) bool { return r < '0' || r > '9' })
	for i := 0; i+1 < len(numbers); i += 2 {
		channel, _ := strconv.Atoi(numbers[i])
		program, _ := strconv.Atoi(numbers[i+1])
		instrument := i / 2 % midiVoices                                    // the driver keeps the low nibble
		m.channel[instrument] = (channel + midiChannels - 1) % midiChannels // MIDI channels 1-16
		m.program[instrument] = program - 1                                 // 0 becomes noProgram
	}
}
