package game

import (
	"testing"

	"github.com/memmaker/u6-go/internal/sound"
)

func TestFreeBalloonMovement(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	removeGargoyles(g)
	leader := int(g.Party[g.Active])
	g.setType(leader, ObjBalloon2)
	if !g.windBorne() {
		t.Error("the balloon is not wind-borne")
	}
	g.WindDir = noWind
	start := g.object(leader).Pos
	defer func() { sound.Settings.FreeBalloonMovement = false }()
	for _, free := range []bool{false, true} {
		sound.Settings.FreeBalloonMovement = free
		g.advance(dirEast)
		if moved := g.object(leader).Pos != start; moved != free {
			t.Errorf("free %v: moved %v without wind", free, moved)
		}
	}
}
