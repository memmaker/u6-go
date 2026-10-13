package intro

import (
	"encoding/binary"
	"strings"
)

// The text output of UL.LIB: a window (a surface and a cursor) printing the glyphs of the
// U6.SET font, word wrapped and justified.

// font is U6.SET: a height, the colours its glyphs are drawn in, and per character a width
// and the offset of its glyph (rows of width pixels).
type font struct {
	raw []byte
}

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

// Justification of txt_putsj.
const (
	justifyLeft = iota
	justifyRight
	justifyCenter
	justifyFull
)

// textWindow is a text window (tTextWin): the surface it prints on, the cursor, colours,
// the string buffer TXT_sprintf fills, and the justification.
type textWindow struct {
	surf    *surface
	x, y    int
	fg, bg  int
	buf     []byte
	justify int
}

// glyph draws character c at the cursor and moves the cursor right (at most to the right
// edge). The font's own colours become the window's; the glyph is not clipped.
// C: MCGADRV GR(0x12)
func (u *intro) glyph(c byte) {
	w := u.win
	f := u.font
	width, at := f.width(c), f.glyphAt(c)
	fore, back := f.raw[fontForeAt], f.raw[fontBackAt]
	s := w.surf
	for row := range f.height() {
		for col := range width {
			p := f.raw[at+row*width+col]
			switch p {
			case fore:
				p = byte(w.fg)
			case back:
				p = byte(w.bg)
			}
			if x, y := w.x+col, w.y+row; s.inPicture(x, y) {
				s.pixels[s.at(x, y)] = p
			}
		}
	}
	w.x = min(w.x+width, s.right)
}

const (
	charLineFeed       = '\n'
	charCarriageReturn = '\r'
)

// textChar prints a character at the cursor; CR and LF move the cursor, and a character
// that does not fit on the line starts a new one.
// C: seg_135A:0040 (text_char)
func (u *intro) textChar(c byte) {
	switch c {
	case charCarriageReturn:
		u.textCR()
		return
	case charLineFeed:
		u.textLF()
		return
	}
	for u.win.x+u.font.width(c)-1 > u.win.surf.right {
		u.textLF()
	}
	u.glyph(c)
}

// textPrint prints s with textChar.
// C: seg_12EE:0229 (TXT_printf)
func (u *intro) textPrint(s string) {
	for i := range len(s) {
		u.textChar(s[i])
	}
}

// textCR moves the cursor to the left edge.
// C: seg_135A:00B2 (text_cr)
func (u *intro) textCR() { u.win.x = u.win.surf.left }

// textLF starts a new line, scrolling the window when the cursor is on its last line.
// C: seg_135A:00BF (text_lf)
func (u *intro) textLF() {
	u.textCR()
	if h := u.font.height(); u.win.y+h < u.win.surf.bottom {
		u.win.y += h
	} else {
		u.scrollUp()
	}
}

// scrollUp scrolls the window up one text line and clears the last one.
// C: seg_1317:003C (txt_scrollup)
func (u *intro) scrollUp() {
	s := u.win.surf
	h := u.font.height()
	for y := s.top + h; y <= s.bottom; y++ {
		from, to := s.at(s.left, y), s.at(s.left, y-h)
		copy(s.pixels[to:to+s.right-s.left+1], s.pixels[from:from+s.right-s.left+1])
	}
	u.clearLine(s.bottom-h+1, s.left)
}

// clearLine fills a text line (one row more than the font is high) from x to the right edge
// with the background colour.
// C: seg_1317:00B1 (txt_clrline)
func (u *intro) clearLine(y, x int) {
	r := *u.win.surf
	r.left, r.top, r.bottom = x, y, y+u.font.height()
	fill(r, u.win.bg)
}

// clearScreen homes the cursor and clears the window.
// C: seg_1288:0002 (txt_clrscr)
func (u *intro) clearScreen() {
	u.win.x, u.win.y = u.win.surf.left, u.win.surf.top
	u.clearDown()
}

// clearDown clears from the cursor's line to the bottom (the first line from the cursor).
// C: seg_1288:0034 (txt_clrdown), seg_1288:0024 (txt_clreos)
func (u *intro) clearDown() {
	x := u.win.x
	for y := u.win.y; y < u.win.surf.bottom; y += u.font.height() {
		u.clearLine(y, x)
		x = u.win.surf.left
	}
}

// clearToEOL clears the cursor's line from the cursor.
// C: seg_10E1:000A (txt_clreol)
func (u *intro) clearToEOL() { u.clearLine(u.win.y, u.win.x) }

// sprintf starts the window's string buffer with s (the formatting is done by the callers).
// C: seg_12EE:0249 (TXT_sprintf)
func (u *intro) sprintf(s string) { u.win.buf = append(u.win.buf[:0], s...) }

// sprintfcat appends s to the window's string buffer.
// C: seg_12EE:0277 (TXT_sprintfcat)
func (u *intro) sprintfcat(s string) { u.win.buf = append(u.win.buf, s...) }

// printBuffer prints the string buffer justified (the %P of TXT_vprintf).
func (u *intro) printBuffer() { u.putsJustified(string(u.win.buf)) }

// putsJustified prints s word wrapped to the window, each line justified by the window's
// mode: left, right, centred, or (but for a paragraph's last line) stretched to both edges.
// C: seg_128D:000C (txt_putsj), seg_128D:001F (just_text)
func (u *intro) putsJustified(s string) {
	str := append([]byte(s), 0) // a C string
	w := u.win
	mode, right, left := w.justify, w.surf.right, w.surf.left
	isEnd := func(c byte) bool { return c == 0 || c == charLineFeed }
	pos := 0
	for {
		// the line [start, end): up to the last space before the right edge, or a whole first
		// word, or through the end of the paragraph
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
			if x += u.font.width(c); x >= right {
				break
			}
		}
		available := right - w.x + 1
		width := 0
		for i := start; i < end; i++ {
			width += u.font.width(str[i])
		}
		line := justifiedLine{u: u, str: str, start: start, end: end, width: width, available: available}
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
			u.textLF()
			for str[pos] == ' ' {
				pos++
			}
		}
		if last == 0 {
			return
		}
	}
}

// justifiedLine is one line of putsJustified.
type justifiedLine struct {
	u                *intro
	str              []byte
	start, end       int
	width, available int
}

// center indents the line by half the space it leaves.
// C: seg_128D:0149 (just_center)
func (l *justifiedLine) center() {
	if l.str[l.start] != 0 && l.width <= l.available {
		l.u.win.x += (l.available - l.width + 1) / 2
	}
	l.left()
}

// right puts the line against the right edge (at the left edge when it does not fit).
// C: seg_128D:016C (just_right)
func (l *justifiedLine) right(left, right int) {
	if l.str[l.start] != 0 && l.width <= l.available {
		l.u.win.x = right - l.width + 1
	} else {
		l.u.win.x = left
	}
	l.left()
}

// left prints the line from the cursor.
// C: seg_128D:0190 (just_left)
func (l *justifiedLine) left() {
	for i := l.start; i < l.end && l.str[i] != 0; i++ {
		l.u.textChar(l.str[i])
	}
}

// full prints leading spaces and the first word, then spreads the room left over the
// line's spaces.
// C: seg_128D:01B0 (just_full)
func (l *justifiedLine) full(right int) {
	if l.width >= l.available || l.str[l.end-1] == 0 || l.str[l.end-1] == charLineFeed {
		l.left()
		return
	}
	u := l.u
	for l.str[l.start] == ' ' {
		u.textChar(' ')
		l.width -= u.font.width(' ')
		l.start++
	}
	for l.str[l.start] != ' ' {
		c := l.str[l.start]
		u.textChar(c)
		l.width -= u.font.width(c)
		l.start++
	}
	room := right - u.win.x + 1 - l.width
	spaces := strings.Count(string(l.str[l.start:l.end]), " ")
	if spaces == 0 {
		l.left()
		return
	}
	perSpace, extra := room/spaces, room%spaces
	for i := l.start; i < l.end && l.str[i] != 0; i++ {
		u.textChar(l.str[i])
		if l.str[i] == ' ' {
			u.win.x += perSpace
			if extra != 0 {
				extra--
				u.win.x++
			}
		}
	}
}
