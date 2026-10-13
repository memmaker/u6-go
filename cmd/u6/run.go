package main

import (
	"fmt"
	"os"

	"github.com/memmaker/u6-go/internal/ending"
	"github.com/memmaker/u6-go/internal/game"
	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/intro"
	"github.com/memmaker/u6-go/internal/sound"
)

// run plays the game from the files in dir, as ULTIMA6.EXE did: the intro and character
// creation (U.EXE), then the game (GAME.EXE), which hands over to the ending (END.EXE).
// With MT-32 ROMs in romDir the music plays on the MT-32, as with U6ROLAND.DRV, else on the
// AdLib.
func run(dir, romDir string) error {
	if err := game.Install(dir); err != nil {
		return err
	}
	files := os.DirFS(dir)
	if romDir != "" {
		if err := sound.StartMT32(romDir); err != nil {
			fmt.Fprintln(os.Stderr, "MT-32:", err, "- playing AdLib music")
		}
	}
	sound.Start(files)
	result, err := intro.Run(files, dir)
	if err != nil || result == intro.Quit {
		return err
	}
	g, err := game.New(dir)
	if err != nil {
		return err
	}
	host.ShowFrame = g.Frame
	g.Run()
	if g.Ending == nil {
		return nil
	}
	return ending.Run(files, *g.Ending)
}
