package data

import (
	"os"
	"path/filepath"
	"testing"
)

// dataDir is the original game files (set U6DATA, default the u6-re checkout's install copy).
func dataDir(t *testing.T) string {
	dir := os.Getenv("U6DATA")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Projects/u6-re/gog/inst")
	}
	if _, err := os.Stat(filepath.Join(dir, "LZMAP")); err != nil {
		t.Skip("no game data")
	}
	return dir
}

func TestDecompressMap(t *testing.T) {
	packed, err := os.ReadFile(filepath.Join(dataDir(t), "LZMAP"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decompress(packed)
	if err != nil {
		t.Fatal(err)
	}
	// the world map: 64 superchunks of 16x16 chunk numbers (12 bits each) plus 5 dungeon
	// levels of 32x32
	if want := 64*16*16*3/2 + 5*32*32*3/2; len(out) != want {
		t.Fatalf("unpacked %d bytes, want %d", len(out), want)
	}
}
