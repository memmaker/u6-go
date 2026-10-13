package main

import (
	"github.com/memmaker/u6-go/internal/game"
	"github.com/memmaker/u6-go/internal/host"
)

// run plays the game from the files in dir.
func run(dir string) error {
	// TODO(port): the intro (U.EXE) makes the character; until it runs here, a default one
	if err := game.WriteCharacter(dir, game.DefaultCharacter()); err != nil {
		return err
	}
	g, err := game.New(dir)
	if err != nil {
		return err
	}
	host.ShowFrame = g.Frame
	g.Play()
	return nil
}

// writeCover writes dir/COVER.PNG, the picture goplay shows on the game's floppy label.
func writeCover(dir string) error { return nil }
