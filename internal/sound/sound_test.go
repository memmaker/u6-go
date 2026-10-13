package sound

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func dataFS(t *testing.T) string {
	dir := os.Getenv("U6DATA")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Projects/u6-re/gog/inst")
	}
	if _, err := os.Stat(filepath.Join(dir, "ULTIMA.M")); err != nil {
		t.Skip("no game data")
	}
	return dir
}

// writeWAV saves mono float samples as 16-bit PCM.
func writeWAV(path string, pcm []float32) error {
	const header = 44
	b := make([]byte, header+2*len(pcm))
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // mono
	binary.LittleEndian.PutUint32(b[24:], outputRate)
	binary.LittleEndian.PutUint32(b[28:], 2*outputRate)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(2*len(pcm)))
	for i, v := range pcm {
		s := int16(max(-1, min(1, v)) * math.MaxInt16)
		binary.LittleEndian.PutUint16(b[header+2*i:], uint16(s))
	}
	return os.WriteFile(path, b, 0o644)
}

func rms(pcm []float32) float64 {
	sum := 0.0
	for _, v := range pcm {
		sum += float64(v) * float64(v)
	}
	return math.Sqrt(sum / float64(len(pcm)))
}

func TestRenderSongs(t *testing.T) {
	fsys := os.DirFS(dataFS(t))
	for _, name := range Songs[1:] {
		b, err := loadSong(fsys, name)
		if err != nil {
			t.Fatal(name, err)
		}
		pcm := make([]float32, 4*outputRate)
		tickDue, chipDue = 0, 0
		renderMusic(newPlayer(b, newAdlib(b)), pcm)
		if err := writeWAV(filepath.Join(t.TempDir(), name+".wav"), pcm); err != nil {
			t.Fatal(err)
		}
		if r := rms(pcm); r < 0.005 {
			t.Errorf("%s: silent (rms %.4f)", name, r)
		}
	}
}

func TestEffects(t *testing.T) {
	for kind := EffectSilence; kind <= EffectSqueak; kind++ {
		b := newBeeper()
		param := 3116 // a pitch
		if kind == EffectProjectile || kind == EffectCrackle || kind == EffectMissile {
			param = 3 // a distance
		}
		b.effect(kind, param)
		if kind != EffectSilence && rms(b.pcm) == 0 {
			t.Errorf("effect %d is silent", kind)
		}
	}
}
