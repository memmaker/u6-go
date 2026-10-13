package game

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopyProtectionAnswers finds the copy protection questions in every conversation, and
// only them: Lord British's ten, Mariah's five and Selganor's four.
func TestCopyProtectionAnswers(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Projects/u6-re/gog/inst")
	if _, err := os.Stat(filepath.Join(dir, "CONVERSE.A")); err != nil {
		t.Skip("no game data")
	}
	const lordBritish, mariah, selganor = 5, 33, 64
	want := map[int]int{lordBritish: 10, mariah: 5, selganor: 4}
	g := &Game{files: os.DirFS(dir)}
	for n := 1; n < firstMonster; n++ {
		g.script = g.loadConversation(n, false)
		got := 0
		for i, b := range g.script {
			if b != opAskTop {
				continue
			}
			g.pc = i + 1
			if a := g.copyProtectionAnswer(); a != "" {
				got++
				t.Logf("NPC %d: %q", n, a)
			}
		}
		if got != want[n] {
			t.Errorf("NPC %d: %d questions answered, want %d", n, got, want[n])
		}
	}
}
