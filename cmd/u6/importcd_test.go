package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestImportCD imports the FM Towns CD (a zip of its .cue and MODE1/2352 .img), from
// $U6_TOWNS_IMAGE.
func TestImportCD(t *testing.T) {
	image := os.Getenv("U6_TOWNS_IMAGE")
	if image == "" {
		t.Skip("no FM Towns CD image (U6_TOWNS_IMAGE)")
	}
	dir := t.TempDir()
	if err := importCD(image, dir); err != nil {
		t.Fatal(err)
	}
	voices, _ := filepath.Glob(filepath.Join(dir, "CD", "SPEECH", "CHAR*.SAM"))
	if len(voices) < 100 {
		t.Errorf("%d voices", len(voices))
	}
	if err := importCD(t.TempDir(), dir); err == nil {
		t.Error("an empty folder imported")
	}
}
