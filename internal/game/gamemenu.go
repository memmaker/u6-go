package game

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// The port's game menu, quick save and quick load (after Nuvie's game menu dialog and its
// QUICK_SAVE, QUICK_LOAD and LOAD_LATEST_SAVE actions). U6 keeps one saved game in
// SAVEGAME, which is also where the game in play keeps its objects; the quick save is a copy
// of it in QUICKSAV.

// The port's commands of the game menu and its hot keys: they are not keys anyone types.
const (
	cmdQuickSave  = 171
	cmdQuickLoad  = 172
	cmdLoadLatest = 173
	cmdSettings   = 174
	// keyCtrlO opens the game menu
	keyCtrlO = 15
)

// quickSaveDir holds the quick save, beside SAVEGAME.
const quickSaveDir = "QUICKSAV"

// setting is a toggle of the settings page.
type setting struct {
	name   string
	on     func(g *Game) bool
	toggle func(g *Game)
}

// settings are the toggles of the settings page: the sound, then the options of settings.json.
var settings = []setting{
	{"Sound", func(g *Game) bool { return g.SoundOn != 0 }, (*Game).toggleSound},
	option("Show stealing", &sound.Settings.ShowStealing),
	option("Free balloon movement", &sound.Settings.FreeBalloonMovement),
	option("Displayed wind direction", &sound.Settings.DisplayedWindDir),
	option("Skip intro", &sound.Settings.SkipIntro),
	option("Smooth lighting", &sound.Settings.SmoothLighting),
	option("Map tile lighting", &sound.Settings.MapTileLighting),
	option("Square pixels", &sound.Settings.SquarePixels),
}

// talkMusicChoices are the TalkMusic settings in the order the settings page steps through them.
var talkMusicChoices = []string{"", sound.TalkMusicStop, sound.TalkMusicKeep}

// SaveSettings, when set, writes the options after the settings page changed one.
var SaveSettings func()

// option is the setting toggling flag.
func option(name string, flag *bool) setting {
	return setting{name, func(*Game) bool { return *flag }, func(*Game) { *flag = !*flag }}
}

// gameMenu is the game menu (Ctrl+O): it returns the command picked, or 0.
func (g *Game) gameMenu() int {
	items := []menuItem{
		{"Save", cmdSave}, {"Load", keyCtrlR}, {"Quick save", cmdQuickSave}, {"Quick load", cmdQuickLoad},
		{"Load latest", cmdLoadLatest}, {"Settings", cmdSettings}, {"Quit", keyCtrlQ},
	}
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	return g.runMenu(items, g.centredPopup(names).at)
}

// settingsPage shows the settings and the talk music, and changes the one picked until
// cancelled.
func (g *Game) settingsPage() {
	choice := 0
	for {
		lines := make([]string, len(settings), len(settings)+1)
		for i, s := range settings {
			state := "off"
			if s.on(g) {
				state = "on"
			}
			lines[i] = s.name + ": " + state
		}
		talkMusic := sound.Settings.TalkMusic
		if talkMusic == "" {
			talkMusic = "lower"
		}
		lines = append(lines, "Talk music: "+talkMusic)
		p := g.centredPopup(lines)
		p.choice = choice
		if choice = g.runPopup(p); choice == noChoice {
			return
		}
		if choice < len(settings) {
			settings[choice].toggle(g)
		} else {
			next := (slices.Index(talkMusicChoices, sound.Settings.TalkMusic) + 1) % len(talkMusicChoices)
			sound.Settings.TalkMusic = talkMusicChoices[next]
		}
		host.SquarePixels = sound.Settings.SquarePixels
		if SaveSettings != nil {
			SaveSettings()
		}
	}
}

// saveFiles are the files of a saved game.
func saveFiles() []string {
	names := []string{objlistFile}
	for b := range objectBlocks {
		names = append(names, objblkName(b))
	}
	return names
}

// copySave copies the saved game in directory from to directory to.
func copySave(from, to string) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	for _, name := range saveFiles() {
		raw, err := os.ReadFile(filepath.Join(from, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(to, name), raw, 0o644)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// quickSave saves the game without asking and keeps a copy as the quick save.
func (g *Game) quickSave() {
	err := g.saveGame()
	if err == nil {
		err = copySave(filepath.Join(g.dir, saveDir), filepath.Join(g.dir, quickSaveDir))
	}
	if err != nil {
		g.printf("%s\n", err.Error())
		return
	}
	g.printf("Quick saved.\n")
}

// quickLoad makes the quick save the saved game and restores it, without asking.
func (g *Game) quickLoad() {
	quick := filepath.Join(g.dir, quickSaveDir)
	if _, err := os.Stat(filepath.Join(quick, objlistFile)); err != nil {
		g.printf("No quick save.\n")
		return
	}
	if err := copySave(quick, filepath.Join(g.dir, saveDir)); err != nil {
		g.printf("%s\n", err.Error())
		return
	}
	g.restoreGame()
}

// gameMenuKey turns the port's hot keys of the game menu into commands at the command
// prompt: Ctrl+O the menu, F11 quick save, F12 quick load, Shift+F12 load the latest save.
func (g *Game) gameMenuKey(ch int) int {
	switch {
	case ch == keyCtrlO:
		return g.gameMenu()
	case ch == keyF11:
		return cmdQuickSave
	case ch == keyF12 && host.ShiftDown:
		return cmdLoadLatest
	case ch == keyF12:
		return cmdQuickLoad
	}
	return ch
}
