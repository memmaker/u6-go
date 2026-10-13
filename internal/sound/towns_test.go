package sound

import (
	"os"
	"path/filepath"
	"testing"
)

// townsDir is a data folder with the FM Towns CD's files under CD/ (u6 -import-cd), from
// $U6_TOWNS_DATA, else the test is skipped.
func townsDir(t *testing.T) string {
	dir := os.Getenv("U6_TOWNS_DATA")
	if _, err := os.Stat(filepath.Join(dir, "CD", "SOUNDS2.DAT")); dir == "" || err != nil {
		t.Skip("no FM Towns data (U6_TOWNS_DATA)")
	}
	return dir
}

// TestTownsVoices decodes a few of the CD's voices: Lord British's and Tholden's first lines.
func TestTownsVoices(t *testing.T) {
	fsys := os.DirFS(townsDir(t))
	for _, c := range []struct {
		name  string
		index int
	}{{"CHAR5.SAM", 0}, {"CHAR8.SAM", 0}, {"CHAR8.SAM", 2}, {"CHAR228.SAM", 0}} {
		pcm, err := loadSpeech(fsys, c.name, c.index)
		if err != nil {
			t.Fatalf("%s %d: %v", c.name, c.index, err)
		}
		peak := float32(0)
		for _, v := range pcm {
			peak = max(peak, v, -v)
		}
		seconds := float64(len(pcm)) / outputRate
		t.Logf("%s %d: %.2f s, peak %.2f", c.name, c.index, seconds, peak)
		if seconds < 0.3 || seconds > 30 || peak < 0.1 || peak > 1 {
			t.Errorf("%s %d: %.2f s, peak %.2f: not a voice", c.name, c.index, seconds, peak)
		}
	}
}

// TestTownsSamples plays each mapped FM Towns sound effect.
func TestTownsSamples(t *testing.T) {
	files = os.DirFS(townsDir(t))
	defer func() { files, townsLoaded, townsLibraries, samples = nil, false, [2][]byte{}, nil }()
	for _, n := range []int{TownsBlocked, TownsHit, TownsSwing, TownsFire, TownsMissile, TownsBrokenGlass, TownsExplosion, TownsFountain, TownsWaterWheel} {
		samples = nil
		if !PlaySample(n) || len(samples) == 0 {
			t.Fatalf("sample %d did not play", n)
		}
		t.Logf("sample %d: %.2f s", n, float64(len(samples))/outputRate)
	}
}
