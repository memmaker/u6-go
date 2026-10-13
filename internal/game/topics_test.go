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
		{"Sin'Vraal", "mari", "mariah"},
		{"Selganor", "kine", "kinetic force"},
		{"Horance", "tubl", "traubel"},
		{"Iolo", "spam", "spam"},
	} {
		g := &Game{}
		g.script = append([]byte{0xff, 1}, c.npc+"\xf1 bye"...)
		if got := g.topicWord(c.keyword); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.npc, c.keyword, got, c.want)
		}
	}
}

// Every spelled-out topic fits its option column whole.
func TestTopicWordsFit(t *testing.T) {
	check := func(w string) {
		if len(w) >= optionWidth {
			t.Errorf("topic %q is wider than its column (%d)", w, optionWidth-1)
		}
	}
	for _, w := range topicWords {
		check(w)
	}
	for _, words := range npcTopicWords {
		for _, w := range words {
			check(w)
		}
	}
}
