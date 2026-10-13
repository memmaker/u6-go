package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/memmaker/u6-go/internal/sound"
)

// settingsFile holds the optional enhancements (sound.Options) in the game folder; it is
// written with everything off when missing, for the player to edit.
const settingsFile = "settings.json"

// loadSettings reads dir's settings into sound.Settings.
func loadSettings(dir string) {
	path := filepath.Join(dir, settingsFile)
	b, err := os.ReadFile(path)
	if err != nil {
		saveSettings(dir)
		return
	}
	if err := json.Unmarshal(b, &sound.Settings); err != nil {
		fmt.Fprintln(os.Stderr, settingsFile+":", err)
	}
}

// saveSettings writes sound.Settings to dir's settings.
func saveSettings(dir string) {
	if b, err := json.MarshalIndent(sound.Settings, "", "\t"); err == nil {
		_ = os.WriteFile(filepath.Join(dir, settingsFile), b, fileMode)
	}
}
