// Package sound is Ultima VI's music and sound effects: the .M songs played the way
// U6ADLIB.DRV plays them, on an OPL2 emulator, and the PC speaker effects of OSISND.LIB as
// square waves, mixed into one Ebiten audio stream.
//
// The music runs on its own 60 Hz clock in the audio stream (the driver's timer interrupt);
// the effects are queued and their functions return how long they sound, so the game can
// host.Wait where the original busy-waited.
package sound

import (
	"encoding/binary"
	"io/fs"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/memmaker/u6-go/internal/data"
)

const (
	outputRate     = 44100
	mt32Gain       = 1.0 / 65536 // the sum of the MT-32's two 16-bit channels into -1..1
	musicGain      = 1.0 / 16384 // the summed OPL channels (each up to about ±4096) into -1..1
	maxQueuedAudio = 4           // seconds of effects the queue keeps at most
	bytesPerFrame  = 8           // two float32 channels
)

// Songs, numbered as GAME.EXE's MUS_091A and U.EXE's play_music take them (0: stop).
var Songs = [...]string{
	"", "ULTIMA.M", "BOOTUP.M", "INTRO.M", "CREATE.M", "FOREST.M", "HORNPIPE.M",
	"ENGAGE.M", "STONES.M", "DUNGEON.M", "BRIT.M", "GARGOYLE.M", "END.M",
}

// Song numbers.
const (
	SongStop = iota
	SongUltima
	SongBootup
	SongIntro
	SongCreate
	SongForest
	SongHornpipe
	SongEngage
	SongStones
	SongDungeon
	SongBrit
	SongGargoyle
	SongEnd
)

var (
	mu       sync.Mutex
	files    fs.FS
	started  bool
	song     *songPlayer // nil: no music
	mt32     bool        // the music goes to the MT-32 (StartMT32)
	mtFrames []int16
	current  string
	speaker  = newBeeper()
	chipDue  float64 // chip samples owed to the output
	tickDue  int     // output samples until the next driver tick
)

// playerBuffer is how much audio the host player queues ahead.
const playerBuffer = 50 * time.Millisecond

// Start remembers where the game files are and opens the audio stream, once. Scripted runs
// (U6GO_INPUT set) stay silent.
func Start(fsys fs.FS) {
	mu.Lock()
	files = fsys
	if started || os.Getenv("U6GO_INPUT") != "" {
		mu.Unlock()
		return
	}
	started = true
	mu.Unlock()
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(outputRate)
	}
	p, err := ctx.NewPlayerF32(stream{})
	if err != nil {
		return
	}
	p.SetBufferSize(playerBuffer)
	p.Play()
}

// loadSong reads and unpacks a .M file (they are LZW packed, like the drivers).
func loadSong(fsys fs.FS, name string) ([]byte, error) {
	packed, err := fs.ReadFile(fsys, name)
	if err != nil {
		packed, err = fs.ReadFile(fsys, strings.ToLower(name))
	}
	if err != nil {
		return nil, err
	}
	return data.Decompress(packed)
}

// PlaySong starts the song of that file name (e.g. "ULTIMA.M"); the song already playing
// carries on. An empty name stops the music.
// C: MUS_091A
func PlaySong(name string) error {
	name = strings.ToUpper(name)
	mu.Lock()
	defer mu.Unlock()
	if name == current {
		return nil
	}
	if song != nil {
		song.driver.stop()
	}
	song, current = nil, ""
	if name == "" || files == nil {
		return nil
	}
	b, err := loadSong(files, name)
	if err != nil {
		return err
	}
	var driver musicDriver = newAdlib(b)
	if mt32 {
		driver = newRoland(loadInstrumentMap(files, songNumber(name)))
	}
	song, current, tickDue = newPlayer(b, driver), name, 0
	return nil
}

func songNumber(name string) int {
	for n, s := range Songs {
		if s == name {
			return n
		}
	}
	return 0
}

// StartMT32 plays the music on the MT-32 emulator with the ROMs in romDir from now on, as
// the game does with U6ROLAND.DRV. It fails, changing nothing, without the emulator or ROMs.
func StartMT32(romDir string) error {
	if err := mtOpen(romDir); err != nil {
		return err
	}
	mu.Lock()
	mt32 = true
	mu.Unlock()
	return nil
}

// Play starts song number n (Songs; 0 stops).
func Play(n int) error {
	if n < 0 || n >= len(Songs) {
		return nil
	}
	return PlaySong(Songs[n])
}

// StopSong stops the music.
func StopSong() { _ = PlaySong("") }

// CurrentSong is the file name of the song playing, "" for none.
func CurrentSong() string {
	mu.Lock()
	defer mu.Unlock()
	return current
}

// queue runs f on the speaker and queues what it rendered, returning its length.
func queue(f func(b *beeper)) time.Duration {
	mu.Lock()
	defer mu.Unlock()
	before := len(speaker.pcm)
	f(speaker)
	if over := len(speaker.pcm) - maxQueuedAudio*outputRate; over > 0 {
		speaker.pcm = speaker.pcm[over:]
		before = max(before-over, 0)
	}
	return samplesDuration(len(speaker.pcm) - before)
}

// Effect plays speaker effect kind (the Effect constants) with its parameter (a pitch or
// a distance, 0 for most) and returns how long it lasts.
// C: MUS_0065
func Effect(kind, param int) time.Duration {
	return queue(func(b *beeper) { b.effect(kind, param) })
}

// The speaker primitives, for the game's own sequences (ambient object sounds, spells).

// Tone turns the speaker on at hz. C: OSI_sound
func Tone(hz int) { queue(func(b *beeper) { b.sound(hz) }) }

// NoTone turns the speaker off. C: OSI_nosound
func NoTone() { queue(func(b *beeper) { b.nosound() }) }

// Delay holds the speaker's state for count units of a scale (1: about 0.9 ms, 8: a
// quarter of that). C: OSI_sDelay
func Delay(scale, count int) time.Duration {
	return queue(func(b *beeper) { b.sDelay(scale, count) })
}

// PlayNote plays hz for duration units. C: OSI_playNote
func PlayNote(hz, duration int) time.Duration {
	return queue(func(b *beeper) { b.playNote(hz, duration) })
}

// PlayNoise plays random tones up to maxHz. C: OSI_playNoise
func PlayNoise(step, length, maxHz int) time.Duration {
	return queue(func(b *beeper) { b.playNoise(step, length, maxHz) })
}

// PlayWavedNote plays a pulse width modulated note. C: OSI_playWavedNote
func PlayWavedNote(step, stepUnits, steps, width, widthDelta int) time.Duration {
	return queue(func(b *beeper) { b.playWavedNote(step, stepUnits, steps, width, widthDelta) })
}

// renderMusic adds the song's next samples into out.
func renderMusic(p *songPlayer, out []float32) {
	if mt32 {
		renderMIDI(p, out)
		return
	}
	chip := p.driver.(*adlib).chip
	const samplesPerTick = outputRate / driverTickHz
	for i := range out {
		if tickDue == 0 {
			p.tick()
			tickDue = samplesPerTick
		}
		tickDue--
		chipDue += float64(oplRate) / outputRate
		v := 0
		for ; chipDue >= 1; chipDue-- {
			v = chip.sample()
		}
		out[i] += float32(v) * musicGain
	}
}

// renderMIDI ticks the song between runs of the MT-32's output, mixed down to mono.
func renderMIDI(p *songPlayer, out []float32) {
	const samplesPerTick = outputRate / driverTickHz
	for len(out) > 0 {
		if tickDue == 0 {
			p.tick()
			tickDue = samplesPerTick
		}
		n := min(len(out), tickDue)
		if cap(mtFrames) < 2*n {
			mtFrames = make([]int16, 2*n)
		}
		frames := mtFrames[:2*n]
		mtRender(frames)
		for i := range n {
			out[i] += (float32(frames[2*i]) + float32(frames[2*i+1])) * mt32Gain
		}
		out, tickDue = out[n:], tickDue-n
	}
}

type stream struct{}

var mixBuffer []float32

func (stream) Read(b []byte) (int, error) {
	frames := len(b) / bytesPerFrame
	mu.Lock()
	defer mu.Unlock()
	if cap(mixBuffer) < frames {
		mixBuffer = make([]float32, frames)
	}
	mix := mixBuffer[:frames]
	clear(mix)
	if song != nil {
		renderMusic(song, mix)
		g := float32(fader("music"))
		for i := range mix {
			mix[i] *= g
		}
	}
	n := min(frames, len(speaker.pcm))
	g := float32(fader("sfx"))
	for i := range n {
		mix[i] += speaker.pcm[i] * g
	}
	speaker.pcm = speaker.pcm[n:]
	for i, v := range mix {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(b[i*bytesPerFrame:], bits)
		binary.LittleEndian.PutUint32(b[i*bytesPerFrame+bytesPerFrame/2:], bits)
	}
	return frames * bytesPerFrame, nil
}
