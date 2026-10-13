package game

import (
	"slices"
	"testing"
)

func TestFunctionKeysToggleInventory(t *testing.T) {
	g := &Game{}
	g.PartySize = 3
	g.StatusDisplay = cmdStatusParty
	second := keyF1 + 1
	if got := g.statusKey(second); got != cmdStatusEquipment || g.Selection.obj != 1 {
		t.Fatalf("F2 in the party view: %d, member %d", got, g.Selection.obj)
	}
	g.StatusDisplay, g.statusMember = cmdStatusEquipment, 1
	if got := g.statusKey(keyF1); got != cmdStatusEquipment || g.Selection.obj != 0 {
		t.Errorf("F1 over the second's inventory: %d, member %d", got, g.Selection.obj)
	}
	if got := g.statusKey(second); got != cmdStatusParty {
		t.Errorf("F2 again: %d, want the party view", got)
	}
	if got := g.statusKey(keyEscape); got != cmdStatusParty {
		t.Errorf("Esc: %d, want the party view", got)
	}
}

func TestCommandMenuFilters(t *testing.T) {
	names := func(g *Game) []string {
		var n []string
		for _, it := range g.availableCommands() {
			n = append(n, it.name)
		}
		return n
	}
	g := &Game{}
	g.PartySize, g.SoloMember, g.InCombat = 1, -1, 1
	got := names(g)
	for _, want := range []string{"Attack", "Talk", "Get", "Break off combat", "Save game"} {
		if !slices.Contains(got, want) {
			t.Errorf("in combat the menu lacks %q: %v", want, got)
		}
	}
	for _, not := range []string{"Rest", "Begin combat", "Cast", "Party mode"} {
		if slices.Contains(got, not) {
			t.Errorf("in combat the menu offers %q", not)
		}
	}
	g.InCombat, g.Active = 0, 1 // aboard a vehicle, not a ship
	got = names(g)
	for _, not := range []string{"Attack", "Get", "Drop", "Move", "Begin combat", "Break off combat", "Rest"} {
		if slices.Contains(got, not) {
			t.Errorf("aboard the menu offers %q", not)
		}
	}
}
