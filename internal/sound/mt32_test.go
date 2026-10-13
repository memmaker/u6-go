//go:build darwin || linux || freebsd

package sound

import (
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
		pcm := make([]float32, 6*outputRate)
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
	}
	StopSong()
}
