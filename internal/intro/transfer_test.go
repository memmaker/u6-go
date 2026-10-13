package intro

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// TestTransfer transfers an Ultima IV bard from a PARTY.SAV in the game's ULTIMA directory.
func TestTransfer(t *testing.T) {
	t.Setenv("U6GO_INPUT", "test")
	dir := dataCopy(t)
	party := make([]byte, ultima4CharacterAt+u4Virtues+2*u4VirtueCount)
	r := party[ultima4CharacterAt:]
	for at, v := range map[int]int{u4HitPoints: 300, u4MaxHitPoints: 400, u4Experience: 4000,
		u4Strength: 25, u4Dexterity: 30, u4Intelligence: 18, u4Magic: 12, u4Virtues: 50} {
		binary.LittleEndian.PutUint16(r[at:], uint16(v))
	}
	copy(r[u4Name:], "Iolo")
	r[u4Sex], r[u4Class] = maleSex, 1
	if err := os.MkdirAll(filepath.Join(dir, transferSubdir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, transferSubdir, ultima4Save), party, 0o644); err != nil {
		t.Fatal(err)
	}
	pngs := os.Getenv("U6PNG")
	if pngs == "" {
		pngs = t.TempDir()
	}
	u := &intro{fsys: os.DirFS(dir), saveDir: dir}
	if err := u.load(); err != nil {
		t.Fatal(err)
	}
	var result Result
	var runErr error
	host.ShowFrame = u.frame
	host.StartHeadless(func() { result, runErr = u.main() })
	shots := 0
	shot := func(name string) { shots++; savePNG(t, pngs, fmt.Sprintf("t%02d_%s", shots, name), u) }
	steps := []struct {
		name string
		key  byte
	}{{"menu", 't'}, {"found", 'c'}, {"compare", 'c'}, {"portrait", 't'}}
	step, since := 0, 0
	inMenu := func() bool { return u.menuDrawn && u.curMusic == sound.SongUltima }
	const maxTicks = 20000
	tick := 0
	for ; tick < maxTicks && host.Step(); tick++ {
		since++
		if step == 0 && !inMenu() {
			if since%60 == 0 {
				host.PushChar(charEscape)
			}
			continue
		}
		if step < len(steps) && since >= 90 {
			shot(steps[step].name)
			host.PushChar(steps[step].key)
			step, since = step+1, 0
		} else if step == len(steps) && since%60 == 0 {
			host.PushChar(' ')
		}
	}
	if runErr != nil || result != StartGame {
		shot("end")
		t.Fatalf("Run = %v, %v after %d ticks; want StartGame", result, runErr, tick)
	}
	b, err := os.ReadFile(filepath.Join(dir, saveGameDir, objlistName))
	if err != nil {
		t.Fatal(err)
	}
	if got := cString(b[nameAt : nameAt+nameBytes]); got != "Iolo" {
		t.Errorf("name %q", got)
	}
	// strength 25 → 18, dexterity 30 → 20, intelligence 18 → 14; 4000 experience → 400, level 4
	if b[strengthAt] != 18 || b[dexterityAt] != 20 || b[intelligenceAt] != 14 || b[levelAt] != 4 ||
		binary.LittleEndian.Uint16(b[experienceAt:]) != 400 || b[femaleAt] != 0 {
		t.Errorf("str %d dex %d int %d level %d exp %d female %d", b[strengthAt], b[dexterityAt],
			b[intelligenceAt], b[levelAt], binary.LittleEndian.Uint16(b[experienceAt:]), b[femaleAt])
	}
}
