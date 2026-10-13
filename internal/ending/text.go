package ending

import (
	"encoding/binary"
	"errors"
	"strings"
)

var errShortPalette = errors.New("ending: short palette file")

// font is U6.SET: a height, the colours its glyphs are drawn in, and per character a width
// and the offset of its glyph.
type font struct{ raw []byte }

const (
	fontHeightAt     = 0
	fontForeAt       = 2
	fontBackAt       = 3
	fontWidthsAt     = 4
	fontOffsetLowAt  = 260
	fontOffsetHighAt = 516
)

func (f *font) height() int      { return int(binary.LittleEndian.Uint16(f.raw[fontHeightAt:])) }
func (f *font) width(c byte) int { return int(f.raw[fontWidthsAt+int(c)]) }
func (f *font) glyphAt(c byte) int {
	return int(f.raw[fontOffsetLowAt+int(c)]) | int(f.raw[fontOffsetHighAt+int(c)])<<8
}

// Justification (tText's f_10).
const (
	justifyLeft = iota
	justifyRight
	justifyCenter
	justifyFull
)

// textWindow is a tText: the surface it prints on, the cursor, colours and justification.
type textWindow struct {
	surf    *surface
	x, y    int
	fg, bg  int
	justify int
}

const (
	charLineFeed       = '\n'
	charCarriageReturn = '\r'
)

// glyph draws c at the cursor and moves it right; the font's colours become the window's.
// C: driver entry 18
func (e *ending) glyph(c byte) {
	w, f, s := &e.win, e.font, e.win.surf
	width, glyph := f.width(c), f.glyphAt(c)
	fore, back := f.raw[fontForeAt], f.raw[fontBackAt]
	for row := range f.height() {
		for col := range width {
			p := f.raw[glyph+row*width+col]
			switch p {
			case fore:
				p = byte(w.fg)
			case back:
				p = byte(w.bg)
			}
			if x, y := w.x+col, w.y+row; x >= 0 && x < s.Width && y >= 0 && y < screenHeight {
				s.Pixels[at(*s, x, y)] = p
			}
		}
	}
	w.x = min(w.x+width, s.Right)
}

// textChar prints c, starting a new line when it does not fit.
// C: F_0693a (CONPUT.ASM)
func (e *ending) textChar(c byte) {
	switch c {
	case charCarriageReturn:
		e.textCR()
		return
	case charLineFeed:
		e.textLF()
		return
	}
	for e.win.x+e.font.width(c)-1 > e.win.surf.Right {
		e.textLF()
	}
	e.glyph(c)
}

// C: F_069ac (CONPUT.ASM)
func (e *ending) textCR() { e.win.x = e.win.surf.Left }

// textLF starts a new line, scrolling on the last one.
// C: F_069b9 (CONPUT.ASM)
func (e *ending) textLF() {
	e.textCR()
	if h := e.font.height(); e.win.y+h < e.win.surf.Bottom {
		e.win.y += h
	} else {
		e.scrollUp()
	}
}

// scrollUp scrolls the window one text line up and clears the last one.
// C: F_0666e (TEXTFN.ASM)
func (e *ending) scrollUp() {
	s := *e.win.surf
	h := e.font.height()
	for y := s.Top + h; y <= s.Bottom; y++ {
		from, to := at(s, s.Left, y), at(s, s.Left, y-h)
		copy(s.Pixels[to:to+s.Right-s.Left+1], s.Pixels[from:from+s.Right-s.Left+1])
	}
	r := s
	r.Top = s.Bottom - h + 1
	r.Bottom = r.Top + h // one row more than the font is high, as the original
	fill(r, e.win.bg)
}

// printJustified prints s word wrapped, each line justified by the window's mode; the last
// line of a paragraph is not stretched.
// C: textFlush (TEXT.ASM)
func (e *ending) printJustified(s string) {
	str := append([]byte(s), 0)
	w := &e.win
	mode, right, left := w.justify, w.surf.Right, w.surf.Left
	isEnd := func(c byte) bool { return c == 0 || c == charLineFeed }
	pos := 0
	for {
		start, end := pos, pos
		if str[end] != 0 {
			for {
				c := str[end]
				end++
				if isEnd(c) {
					break
				}
				if c == ' ' {
					end--
					break
				}
			}
		}
		x := w.x
		for i := pos; ; {
			c := str[i]
			if isEnd(c) {
				end = i + 1
				break
			}
			if c == ' ' {
				end = i
			}
			i++
			if x += e.font.width(c); x >= right {
				break
			}
		}
		available := right - w.x + 1
		width := 0
		for i := start; i < end; i++ {
			width += e.font.width(str[i])
		}
		line := justifiedLine{e: e, str: str, start: start, end: end, width: width, available: available}
		switch mode {
		case justifyLeft:
			line.left()
		case justifyRight:
			line.right(left, right)
		case justifyCenter:
			line.center()
		default:
			line.full(right)
		}
		pos = end
		last := str[end-1]
		if !isEnd(last) {
			e.textLF()
			for str[pos] == ' ' {
				pos++
			}
		}
		if last == 0 {
			return
		}
	}
}

// justifiedLine is one line of printJustified.
type justifiedLine struct {
	e                *ending
	str              []byte
	start, end       int
	width, available int
}

// C: F_05f13 (TEXT.ASM)
func (l *justifiedLine) center() {
	if l.str[l.start] != 0 && l.width <= l.available {
		l.e.win.x += (l.available - l.width + 1) / 2
	}
	l.left()
}

// C: F_05f36 (TEXT.ASM)
func (l *justifiedLine) right(left, right int) {
	if l.str[l.start] != 0 && l.width <= l.available {
		l.e.win.x = right - l.width + 1
	} else {
		l.e.win.x = left
	}
	l.left()
}

// C: F_05f5a (TEXT.ASM)
func (l *justifiedLine) left() {
	for i := l.start; i < l.end && l.str[i] != 0; i++ {
		l.e.textChar(l.str[i])
	}
}

// full prints leading spaces and the first word, then spreads the room left over the
// line's spaces.
// C: F_05f7a (TEXT.ASM)
func (l *justifiedLine) full(right int) {
	if l.width >= l.available || l.str[l.end-1] == 0 || l.str[l.end-1] == charLineFeed {
		l.left()
		return
	}
	e := l.e
	for l.str[l.start] == ' ' {
		e.textChar(' ')
		l.width -= e.font.width(' ')
		l.start++
	}
	for l.str[l.start] != ' ' {
		c := l.str[l.start]
		e.textChar(c)
		l.width -= e.font.width(c)
		l.start++
	}
	room := right - e.win.x + 1 - l.width
	spaces := strings.Count(string(l.str[l.start:l.end]), " ")
	if spaces == 0 {
		l.left()
		return
	}
	perSpace, extra := room/spaces, room%spaces
	for i := l.start; i < l.end && l.str[i] != 0; i++ {
		e.textChar(l.str[i])
		if l.str[i] == ' ' {
			e.win.x += perSpace
			if extra != 0 {
				extra--
				e.win.x++
			}
		}
	}
}

// The scroll the story is printed on.

// scroll is where showWindow puts the scroll shape and its text (g_1ebc, g_1ebe, g_1e9c and
// g_1ede's corner).
type scroll struct {
	x, y         int
	shape        int
	textX, textY int
}

// Scroll shapes of END.SHP.
const (
	shapeScrollLarge  = 10
	shapeScrollSmall  = 11
	shapeScrollMedium = 12
)

// page is one text of showText: the scroll, the text picture's size, the text.
type page struct {
	scroll
	right, bottom int
	text          string
	centred       bool
}

const (
	textColour   = 62
	scrollColour = 49
	textLeft     = 8
	textRight    = 303
	// closeQuoteChar is U6.SET's closing quotation mark.
	closeQuoteChar = 127
	oneLine        = 15 // text picture heights
	threeLines     = 23
	fiveLines      = 39
	sixLines       = 47
)

var closeQuote = string(rune(closeQuoteChar))

// Scroll placements of the pages.
var (
	smallScroll     = scroll{x: 0, y: 160, shape: shapeScrollSmall, textX: textLeft, textY: 174}
	smallScrollHigh = scroll{x: 0, y: 152, shape: shapeScrollSmall, textX: textLeft, textY: 162}
	mediumScroll    = scroll{x: 1, y: 151, shape: shapeScrollMedium, textX: textLeft, textY: 163}
	largeScroll     = scroll{x: 1, y: 133, shape: shapeScrollLarge, textX: textLeft, textY: 146}
	largeScrollFull = scroll{x: 1, y: 133, shape: shapeScrollLarge, textX: textLeft, textY: 142}
	congratsScroll  = scroll{x: 1, y: 68, shape: shapeScrollLarge, textX: textLeft, textY: 77}
)

// pages are showText's texts 0..14 (15, the congratulation, is made by congratulations).
var pages = [...]page{
	{smallScroll, textRight, oneLine, "A glowing portal springs from the floor!", true},
	{largeScroll, textRight, fiveLines, "From its crimson depths Lord British emerges, trailed by " +
		"the mage Nystul. Anguish and disbelief prevail on the " +
		"royal seer's face, but Lord British directs his stony " +
		"gaze at you and speaks as if to a wayward child.", false},
	{mediumScroll, textRight, threeLines, "\"Thou didst have just cause to burgle our " +
		"Codex, I trust," + closeQuote + " His Majesty says. \"But for Virtue's sake...", false},
	{smallScroll, textRight, oneLine, "\"WHAT HAST THOU DONE WITH IT?" + closeQuote, true},
	{largeScrollFull, textRight, sixLines, "You pick up the concave lens and pass it to the King. " +
		"\"Was the book ever truly ours, Your Majesty? Was it " +
		"written for Britannia alone? Thou dost no longer " +
		"hold the Codex, but is its wisdom indeed lost? Look " +
		"into the Vortex, and let the Codex answer for itself!" + closeQuote, false},
	{mediumScroll, textRight, threeLines, "As Lord British holds the glass before the " +
		"wall, the Codex of Ultimate Wisdom wavers into " +
		"view against a myriad of swimming stars!", false},
	{smallScroll, textRight, oneLine, "Yet the book remains closed.", true},
	{smallScrollHigh, textRight, oneLine, "And waves of heat shimmer in the air, heralding the " +
		"birth of another red gate!", false},
	{largeScrollFull, textRight, sixLines, "King Draxinusom of the Gargoyles strides forward, " +
		"flanked by a small army of wingless attendants. Like " +
		"Lord British, he seems to suppress his rage only through " +
		"a heroic effort of will. His scaly " +
		"hand grasps your shoulder, and your Amulet of Submission grows " +
		"very warm.", false},
	{smallScroll, textRight, oneLine, "\"Thy time hath come, Thief," + closeQuote + " he says.", true},
	{smallScrollHigh, textRight, oneLine, "Quickly you reach down to seize the convex lens...", true},
	{mediumScroll, textRight, threeLines, "...and you press it into the hand of the towering " +
		"Gargoyle king, meeting his sunken eyes. " +
		"\"Join my Lord in his search for peace, I beg thee." + closeQuote, false},
	{largeScroll, textRight, fiveLines, "At your urging, King Draxinusom reluctantly raises his " +
		"lens to catch the light. As Lord British holds up his own lens, " +
		"every eye in the room, human and Gargoyle alike, fixes " +
		"upon the image of the Codex which shines upon the wall.", false},
	{largeScroll, textRight, fiveLines, "The ancient book opens. Both kings gaze upon its pages in " +
		"spellbound silence, as the eloquence of Ultimate Wisdom is revealed " +
		"in the tongues of each lord's domain. You, too, can read the answers " +
		"the Codex gives...", false},
	{largeScroll, textRight, fiveLines, "...and when its wisdom is gleaned, when Lord British " +
		"and King Draxinusom turn to each other as friends, " +
		"hating no longer, fearing no more, you know that your " +
		"mission in Britannia has ended at last.", false},
}

// congratulations is page 15, with the time the quest took.
func (e *ending) congratulations() page {
	var b strings.Builder
	b.WriteString("CONGRATULATIONS\n")
	b.WriteString("Thou hast completed Ultima VI: The False Prophet, ")
	b.WriteString("in ")
	if e.years != "" {
		b.WriteString(e.years)
		if e.months != "" || e.days != "" {
			b.WriteString(", ")
		}
	}
	if e.months != "" {
		b.WriteString(e.months)
		if e.days != "" {
			b.WriteString(",\n")
		}
	}
	if (e.years != "" || e.months != "") && e.days != "" {
		b.WriteString("and ")
	}
	if e.days != "" || e.years == "" && e.months == "" {
		b.WriteString(e.days)
	}
	b.WriteString(".\nReport thy feat unto Lord British at Origin Systems!")
	return page{congratsScroll, textRight, sixLines, b.String(), true}
}

// showText prints page n on the text picture and makes it the scroll's text.
// C: seg_00A9:016A (showText)
func (e *ending) showText(n int) {
	e.flushKeys()
	p := e.congratulations()
	if n < len(pages) {
		p = pages[n]
	}
	e.textDest = e.back
	e.textDest.Left, e.textDest.Top = p.textX, p.textY
	e.scroll = p.scroll
	e.textPicture.Left, e.textPicture.Top = 0, 0
	e.textPicture.Right, e.textPicture.Bottom = p.right, p.bottom
	fill(e.textPicture, scrollColour)
	e.win = textWindow{surf: &e.textPicture, fg: textColour, bg: scrollColour, justify: justifyFull}
	if p.centred {
		e.win.justify = justifyCenter
	}
	e.printJustified(p.text)
}

// showWindow draws the scroll and its text on the back picture.
// C: seg_00A9:099A (showWindow)
func (e *ending) showWindow() {
	e.draw(e.back, e.scroll.x, e.scroll.y, e.scroll.shape)
	copyRect(e.textPicture, e.textDest)
}
