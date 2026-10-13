package game

import "testing"

func TestTopicWordTable(t *testing.T) {
	for _, c := range []struct{ npc, keyword, want string }{
		{"Lord British", "monu", "monuments"},
		{"Lord British", "mush", "mushroom"},
		{"Shamino", "garg", "gargoyles"},
		{"Ephemerides", "vort", "vortex"},
		{"Yorl", "lock", "locket"},
		{"Troy", "inte", "interconnected"},
		{"Sin'Vraal", "mari", "mari"},
		{"Iolo", "spam", "spam"},
	} {
		g := &Game{}
		g.script = append([]byte{0xff, 1}, c.npc+"\xf1 bye"...)
		if got := g.topicWord(c.keyword); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.npc, c.keyword, got, c.want)
		}
	}
}
