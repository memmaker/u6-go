package main

import (
	"os"
	"testing"
)

func TestWriteCover(t *testing.T) {
	dir := os.Getenv("U6COVERDIR")
	if dir == "" {
		t.Skip("set U6COVERDIR to a folder with TITLES.SHP and PALETTES.INT")
	}
	if err := writeCover(dir); err != nil {
		t.Fatal(err)
	}
}
