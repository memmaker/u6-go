//go:build darwin || linux || freebsd

package sound

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestMT32Songs plays every song on the MT-32 with the ROMs in U6ROMS (default: Ultima
// Underworld's), saving them as WAVs in U6WAVDIR if set.
func TestMT32Songs(t *testing.T) {
	dir := dataFS(t)
	roms := os.Getenv("U6ROMS")
	if roms == "" {
		home, _ := os.UserHomeDir()
		roms = filepath.Join(home, "Projects/UltimaUnderworld/roms")
	}
	if err := StartMT32(roms); err != nil {
		t.Skip(err)
	}
	defer func() { mt32 = false }()
	files = os.DirFS(dir)
	out := os.Getenv("U6WAVDIR")
	for _, name := range Songs[1:] {
		if err := PlaySong(name); err != nil {
			t.Fatal(name, err)
		}
		if midiDriver.instruments.program[0] == noProgram {
			t.Error("MIDI.DAT: no program for ULTIMA.M's first instrument")
		}
		pcm := make([]float32, outChannels*6*outputRate)
		tickDue = 0
		renderMusic(song, pcm)
		if out != "" {
			if err := writeWAV(filepath.Join(out, name+".wav"), pcm); err != nil {
				t.Fatal(err)
			}
		}
		if r := rms(pcm); r < 0.005 {
			t.Errorf("%s: silent (rms %.4f)", name, r)
		}
		stereo := false
		for i := 0; i < len(pcm) && !stereo; i += outChannels {
			stereo = pcm[i] != pcm[i+1]
		}
		if !stereo {
			t.Errorf("%s: mono (the MT-32 pans its parts and reverb)", name)
		}
	}
	StopSong()
}

// TestMT32RestartedNote: CREATE.M ends its G2 bass and starts it again a tick later at
// the start of every bar. munt must hold the restart until the old note is cut off, as
// the real MT-32 does, so the second bar has its bass like the first.
func TestMT32RestartedNote(t *testing.T) {
	const (
		bassHz   = 98.0 // G2
		barTicks = 120
		minRatio = 0.5
	)
	dir := dataFS(t)
	home, _ := os.UserHomeDir()
	if err := StartMT32(filepath.Join(home, "Projects/UltimaUnderworld/roms")); err != nil {
		t.Skip(err)
	}
	defer func() { mt32 = false }()
	files = os.DirFS(dir)
	if err := PlaySong("CREATE.M"); err != nil {
		t.Fatal(err)
	}
	defer StopSong()
	bar := outChannels * outputRate * barTicks / driverTickHz
	pcm := make([]float32, 2*bar)
	tickDue = 0
	renderMusic(song, pcm)
	first, second := tone(pcm[:bar], bassHz), tone(pcm[bar:], bassHz)
	if second < minRatio*first {
		t.Errorf("the second bar's bass is at %.4f, the first's at %.4f", second, first)
	}
}

// tone is the amplitude of frequency hz in the left channel of stereo frames.
func tone(pcm []float32, hz float64) float64 {
	var re, im float64
	n := len(pcm) / outChannels
	for i := range n {
		phase := 2 * math.Pi * hz * float64(i) / outputRate
		re += float64(pcm[i*outChannels]) * math.Cos(phase)
		im += float64(pcm[i*outChannels]) * math.Sin(phase)
	}
	return math.Hypot(re, im) / float64(n)
}
