// Command u6 runs Ultima VI: The False Prophet (the DOS VGA version) from the original data
// files: ULTIMA6.EXE's job, the intro and character creation (U.EXE), the game (GAME.EXE)
// and the ending (END.EXE), each ported to Go.
//
//	u6 [-data DIR]            play with the game files in DIR (default $GOPLAY_DATA, else ./game)
//	u6 -import SRC -data DIR  copy the game files out of SRC (folder, zip, ISO/BIN/CUE, GOG
//	                          installer, loose files) into DIR, write DIR/COVER.PNG and exit
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/memmaker/u6-go/internal/gameimport"
	"github.com/memmaker/u6-go/internal/host"
)

// markerFiles prove a folder holds Ultima VI.
var markerFiles = []string{"MAPTILES.VGA", "LZMAP", "CONVERSE.A"}

// Permissions of the directories and files written.
const (
	dirMode  = 0o755
	fileMode = 0o644
)

// gameFile says whether name is one of the game's data files: everything on the original
// disks but the programs and drivers for hardware the port does not emulate.
func gameFile(name string) bool {
	base := strings.ToUpper(filepath.Base(name))
	switch strings.ToUpper(filepath.Ext(base)) {
	case ".EXE", ".BAT", ".BIN", ".IMG", ".IMA", ".ISO", ".ZIP", ".CUE", ".GOG", ".7Z", ".COM", ".DLL", ".INI", ".TXT", ".PDF", ".DOC", ".CGA", ".EGA", ".TGA":
		return false
	}
	return !strings.ContainsAny(base, " ")
}

func main() {
	defaultDir := os.Getenv("GOPLAY_DATA")
	if defaultDir == "" {
		defaultDir = "game"
	}
	dir := flag.String("data", defaultDir, "directory with the original game files")
	source := flag.String("import", "", "import the game files from this folder/zip/ISO/installer into -data and exit")
	scale := flag.Int("scale", 3, "window scale")
	roms := flag.String("roms", os.Getenv("GOPLAY_MT32"), "the MT-32 ROM directory for music (default: roms/ in the working directory or next to the binary; none: AdLib)")
	flag.Parse()

	if *source != "" {
		if err := importGame(*source, *dir); err != nil {
			fmt.Fprintln(os.Stderr, "import:", err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stat(filepath.Join(*dir, markerFiles[0])); err != nil {
		fmt.Fprintf(os.Stderr, "no game files in %s; use -import SRC or -data DIR\n", *dir)
		os.Exit(1)
	}
	var gameErr error
	err := host.Run("Ultima VI", *scale, func() { gameErr = run(*dir, romDir(*roms)) })
	if err == nil {
		err = gameErr
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// importGame copies the game files found in source into dir and writes the cover picture.
func importGame(source, dir string) error {
	files, err := gameimport.Find(source, gameFile)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for name := range files {
		have[strings.ToUpper(filepath.Base(name))] = true
	}
	for _, marker := range markerFiles {
		if !have[marker] {
			return fmt.Errorf("no Ultima VI files (%s) found in %s", marker, source)
		}
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, strings.ToUpper(filepath.Base(name))), contents, fileMode); err != nil {
			return err
		}
	}
	return writeCover(dir)
}

// romDir is -roms, else the first roms/ directory found in the working directory or next
// to the binary ("" if none).
func romDir(flagDir string) string {
	if flagDir != "" {
		return flagDir
	}
	dirs := []string{"roms"}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "roms"))
	}
	for _, d := range dirs {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			return d
		}
	}
	return ""
}
