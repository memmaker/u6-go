package intro

import "strings"

// The text passages of the intro and the character creation: a block of BLOCKS.SHP drawn on
// the back page with the passage's text printed over it.

// passage is one passage: the block and where it goes, the text's rectangle on the back page
// (its top left corner and the size of the text area, TextRight+1 by TextBottom+1), and
// whether its lines are centred (else stretched to both edges).
type passage struct {
	Block, BlockX, BlockY int
	TextX, TextY          int
	TextRight, TextBottom int
	Centered              bool
	Text                  string
}

const (
	// closeQuoteChar is the font's closing quotation mark.
	closeQuoteChar = 127
	// nameMarker in a passage's text is replaced by the avatar's name.
	nameMarker = "%s"

	passageTextColour       = 62
	passageBackgroundColour = 49
)

var closeQuote = string(rune(closeQuoteChar))

// showPassage prints passage n into the passage picture; drawPassage puts it on the back page.
// C: seg_09A0:000B (PASSAGE_show)
func (u *intro) showPassage(n int) {
	saved := u.win
	u.flushKeys()
	u.passageDest = u.back
	p, ok := passages[n]
	if !ok {
		p = undefinedPassage
	}
	u.passageBlockX, u.passageBlockY, u.passageBlock = p.BlockX, p.BlockY, p.Block
	u.passageDest.left, u.passageDest.top = p.TextX, p.TextY
	u.passageText.right, u.passageText.bottom = p.TextRight, p.TextBottom
	fill(u.passageText, passageBackgroundColour)
	win := textWindow{surf: &u.passageText, fg: passageTextColour, bg: passageBackgroundColour,
		justify: justifyFull}
	if p.Centered {
		win.justify = justifyCenter
	}
	u.win = &win
	u.putsJustified(strings.ReplaceAll(p.Text, nameMarker, cString(u.char.name[:])))
	u.win = saved
}

// drawPassage draws the current passage's block and text on the back page.
// C: seg_09A0:21C5 (PASSAGE_21C5)
func (u *intro) drawPassage() {
	u.blocks.draw(u.back, u.passageBlockX, u.passageBlockY, u.passageBlock)
	u.copyRect(u.passageText, u.passageDest)
}
