package intro

// The acknowledgments: the credits on the vellum of the main menu.

const (
	creditsLeft, creditsRight = 28, 292
	creditsTop, creditsBottom = 89, 189
	creditsRuleRow            = 9
	creditsTextRow            = 14
	creditsTextColour         = 61
	creditsHeadingColour      = 12
	creditsBackground         = 49
	vellumShape               = 0
)

// creditPages are the pages of credits after the first: per page its headings, each with
// the text before it ("\n" lines), the heading and the names.
var creditPages = [...][]struct{ before, heading, names string }{
	{
		{"", "Produced by", "Richard Garriott and Warren Spector\n\n"},
		{"", "Executive Producer", "Dallas Snell\n\n"},
		{"", "Programming", "Cheryl Chen    John Miles\nHerman Miller    Gary Scott Smith"},
	},
	{
		{"\n\n", "Writing", "Stephen Beeman    Dr. Cat    'Manda Dee\nRichard Garriott    Greg Malone\nJohn Miles    Herman Miller\nTodd Porter    Warren Spector\n"},
	},
	{
		{"", "Art", "Keith Berdak    Daniel Bourbonnais\nJeff Dee    'Manda Dee\nGlen Johnson    Denis Loubet\n\n"},
		{"", "Music", "Ken Arnold    Iolo Fitzowen\nHerman Miller    Todd Porter\n"},
	},
	{
		{"", "Quality Assurance", "Paul Malone    Mike Romero\nMarc Schaefgen    John Watson\n\n"},
		{"", "Additional Support", "Michelle Caddel    Melanie Fleming\nAlan Gardner    Jeff Hillhouse\nSherry Hunter    Steve Muchow\nCheryl Neeld"},
	},
}

// showCredits shows the credits on the vellum, a page per key or click.
// C: seg_0676:0006 (ShowCredits)
func (u *intro) showCredits() {
	back, front := u.back, u.screen
	for _, s := range []*surface{&back, &front} {
		s.left, s.top, s.right, s.bottom = creditsLeft, creditsTop, creditsRight, creditsBottom
	}
	u.vellumBack = u.vellum
	samePicture(u.back, &u.vellumBack)
	fill(u.vellumBack, 0)
	saved := surface{right: u.vellum.right - u.vellum.left, bottom: u.vellum.bottom - u.vellum.top}
	newPicture(&saved)
	vellum := u.mustShapes("VELLUM1.SHP")
	u.copyRect(u.vellum, saved)
	u.fadeOut(u.vellum, 0)
	fill(u.vellum, 0)
	u.setPalette(&u.menuPalette)
	vellum.draw(u.vellumBack, u.vellumBack.left, u.vellumBack.top, vellumShape)

	win := textWindow{surf: &back, x: back.left, y: back.top, fg: creditsTextColour,
		bg: creditsBackground, justify: justifyCenter}
	saveWin := u.win
	u.win = &win
	defer func() { u.win = saveWin }()
	u.putsJustified("Ultima VI\nA Lord British Production")
	rule := back
	rule.top, rule.bottom = win.y+creditsRuleRow, win.y+creditsRuleRow
	fill(rule, creditsTextColour)
	back.top = win.y + creditsTextRow
	front.top = back.top
	for i, page := range creditPages {
		u.clearScreen()
		if i == 0 {
			win.y++
		}
		for _, h := range page {
			win.fg = creditsHeadingColour
			u.putsJustified(h.before + h.heading + "\n")
			win.fg = creditsTextColour
			u.putsJustified(h.names)
		}
		if i == 0 {
			u.fadeIn(u.vellumBack, u.vellum)
			continue
		}
		u.waitInput()
		u.copyRect(back, front)
	}
	u.waitInput()
	u.fadeOut(u.vellum, 0)
	fill(u.vellum, 0)
	u.setPalette(&u.menuPalette)
	u.fadeIn(saved, u.vellum)
}
