package game

import (
	"strings"
	"unicode"
)

// The console prints text in character cells (8x8 pixels) inside a clip window: the
// message scroll, the party list, the date line. Text is word wrapped through a line
// buffer and understands a few control characters (CON_putch).

// Clip attributes (tClipInfo._attr).
const (
	clipWordWrap  = 1
	clipCentre    = 2
	clipUppercase = 4
	// clipHighlight sets the high bit of printed characters (the font's runic/inverse half)
	clipHighlight = 8
	clipPaging    = 16
	clipScroll    = 32
)

// clip is a text window (struct tClipInfo); positions and sizes are in character cells.
type clip struct {
	x0, y0, width, height int
	curX, curY            int
	fg, bg                byte
	attr                  byte
}

// the clips (D_B6B5)
const (
	clipScreen  = 0
	clipParty   = 1
	clipMessage = 3
	clipCount   = 4
)

const (
	cellSize       = 8
	lineBufferSize = 41
	// promptChar is the first of the four frames of the animated cursor (PromptCh)
	promptChar = 5
	// morePrompt is the cursor shown while paging waits for a key
	morePrompt = 1
	// endOfLine flushes the line buffer without a new line (CON_putch0(-1))
	endOfLine = -1
)

// console is the state of the console module (seg_0C9C).
type console struct {
	clips   [clipCount]clip
	current *clip
	// initialAttr is given to new clips (D_04E1)
	initialAttr byte
	// the line being word wrapped: characters and their colours (D_B651, D_B67A), its
	// length (D_057A), the part already shown (D_057B), and whether a word started on it
	// (D_057C)
	line       [lineBufferSize]byte
	lineColour [lineBufferSize]byte
	lineLen    int
	lineShown  int
	lineWord   bool
	// linesLeft counts the lines until paging waits for a key (D_B729)
	linesLeft int
	// TextOn turns all printing off when false.
	TextOn bool
	// capitalizeNext comes from '`' (D_0498)
	capitalizeNext bool
	// hideText is toggled by '&' (D_04A3): the text in between is shown only to those who
	// speak gargish
	hideText bool
	// skipNewline drops new lines at the top of a scrolling clip (D_04A9)
	skipNewline bool
	// control is the pending CON_putch control character (D_057D)
	control byte
	// plural is set when the last number printed was not 1 (D_049D)
	plural bool
	// Help highlights keywords ('@') when on (D_04A1).
	Help bool
	// VarInt and VarStr are the conversation variables printed by '#' and '$'.
	VarInt [36]int
	VarStr [36]string

	// prompt state: the cursor frame (PromptCh), its animation counter (PromptCnt),
	// whether the keyboard keeps only the last waiting key (D_04AB), whether CON_getch flushes
	// the line first (D_04AF), whether keys are made uppercase (D_04B1), whether a string
	// is being typed (D_049B) and whether the prompt animates the world (D_049C)
	promptChar     int
	promptCount    int
	drainKeys      bool
	flushOnGetch   bool
	uppercaseKeys  bool
	typingString   bool
	animatePrompt  bool
	pagingDisabled bool // D_049F
}

func (c *console) initConsole() {
	c.current = &c.clips[clipScreen]
	*c.current = clip{width: 40, height: 25, fg: 15}
	c.initialAttr = clipWordWrap
	c.lineWord = true
	c.TextOn = true
	c.Help = true
	c.promptChar = promptChar
	c.drainKeys = true
	c.flushOnGetch = true
	c.uppercaseKeys = true
	c.animatePrompt = true
}

// putCharAt draws ch at cell (x, y) in the current clip's colours.
// C: seg_0C9C:0286 (C_0C9C_0286)
func (g *Game) putCharAt(ch byte, x, y int) {
	g.gr.TextBackground = g.current.bg
	g.gr.TextForeground = g.current.fg
	g.gr.DrawChar(ch, x*cellSize, y*cellSize)
}

// scrollUp scrolls a clip up by one line and clears its last line.
// C: seg_0C9C:02F1 (C_0C9C_02F1)
func (g *Game) scrollUp(x, y, w, h int) {
	x, y, w, h = x*cellSize, y*cellSize, w*cellSize, h*cellSize
	g.gr.ScrollRectVertically(x, y, x+w-1, y+h-1, -cellSize)
	g.gr.SetColor(g.current.bg)
	g.gr.FillRect(x, y+h-cellSize, x+w-1, y+h-1)
}

// createClip sets up a clip from (x0, y0) to (x1, y1) and makes it current.
// C: seg_0C9C:2E09 (CON_createClip)
func (g *Game) createClip(index, x0, y0, x1, y1 int) {
	c := &g.clips[index]
	g.current = c
	*c = clip{x0: x0, y0: y0, width: x1 - x0 + 1, height: y1 - y0 + 1,
		bg: g.colours.background, fg: g.colours.text, attr: g.initialAttr}
}

// setClip makes clip index current.
// C: seg_0C9C:2E54 (CON_setClip)
func (g *Game) setClip(index int) {
	g.putch0(endOfLine)
	g.current = &g.clips[index]
	g.gr.TextBackground = g.current.bg
	g.gr.TextForeground = g.current.fg
	g.putch0(endOfLine)
}

// gotoxy moves the cursor of the current clip.
// C: seg_0C9C:2E8F (CON_gotoxy)
func (g *Game) gotoxy(x, y int) {
	x = min(max(x, 0), g.current.width)
	y = min(max(y, 0), g.current.height)
	g.putch0(endOfLine)
	g.current.curX, g.current.curY = x, y
	g.putch0(endOfLine)
}

// putch0 adds a character to the line buffer, showing the line when it is full (word
// wrapping it), at a new line or at endOfLine.
// C: seg_0C9C:2EFA (CON_putch0)
func (g *Game) putch0(ch int) {
	if !g.TextOn {
		return
	}
	c := g.current
	if g.speechSkips(ch) {
		return
	}
	if c == &g.clips[clipMessage] && !g.InConversation {
		g.toastChar(ch)
	}
	if g.skipNewline && c.attr&clipScroll != 0 {
		if ch == '\n' {
			return
		}
		if ch >= 0 {
			g.skipNewline = false
		}
	}
	if c.attr&clipUppercase != 0 && g.capitalizeNext {
		ch = toUpper(ch)
	}
	g.capitalizeNext = false
	if ch == 0 {
		ch = '\n'
	}
	if c.attr&clipHighlight != 0 && ch > ' ' {
		ch |= highBit
	}
	if c.width != g.lineLen && ch != '\n' && ch >= 0 {
		if c.attr&clipWordWrap == 0 || ch != ' ' || g.lineWord {
			g.lineColour[g.lineLen] = c.fg
			g.line[g.lineLen] = byte(ch)
			g.lineLen++
		}
		if ch != ' ' {
			g.lineWord = true
		}
		return
	}
	if g.linesLeft != 0 && ch >= 0 && c.attr&clipPaging != 0 {
		g.linesLeft--
	}
	if c.attr&clipPaging != 0 && g.linesLeft == 0 && ch >= 0 {
		bottom := c.height + c.y0 - 1
		g.waitForMore()
		for column := range c.width {
			g.putCharAt(' ', c.x0+column, bottom)
		}
		g.linesLeft = c.height - 1
	}
	end := g.lineLen
	wrapping := c.attr&clipWordWrap != 0 && c.width == g.lineLen && ch >= 0
	if wrapping {
		if ch == '\n' {
			g.line[g.lineLen] = ' '
		} else {
			g.line[g.lineLen] = byte(ch)
		}
		g.lineColour[g.lineLen] = c.fg
		// break at the last space (or after a hyphen) not yet shown
		for g.line[end] != ' ' && g.line[end-1] != '-' && g.lineShown < end {
			end--
		}
		if end == 0 {
			end = g.lineLen
		}
	}
	y := c.y0 + c.curY
	if c.attr&clipCentre != 0 {
		for column := g.lineShown; column < (c.width-end)>>1+g.lineShown; column++ {
			g.putCharAt(' ', c.x0+column, y)
		}
	}
	for column := g.lineShown; column < end; column++ {
		c.fg = g.lineColour[column]
		g.putCharAt(g.line[column], c.x0+column, y)
	}
	c.fg = g.colours.text
	if g.lineShown < end {
		c.curX = end
	}
	switch {
	case wrapping:
		if g.line[end] == ' ' {
			end++
		}
		from := end
		for ; from <= g.lineLen; from++ {
			g.line[from-end] = g.line[from]
			g.lineColour[from-end] = g.lineColour[from]
		}
		g.lineShown = 0
		g.lineLen = from - end
		if ch == ' ' {
			g.lineWord = false
		}
	case ch >= 0:
		g.lineLen, g.lineShown = 0, 0
		if ch != '\n' {
			g.lineColour[0] = c.fg
			g.line[0] = byte(ch)
			g.lineLen = 1
		}
	}
	if ch >= 0 {
		c.curX = 0
		if c.curY < c.height-1 {
			c.curY++
		} else if c.attr&clipScroll != 0 {
			g.scrollUp(c.x0, c.y0, c.width, c.height)
		}
		g.flowSpeech()
	} else {
		g.lineShown, g.lineLen = c.curX, c.curX
		g.hideText = false
	}
	if ch != ' ' {
		g.lineWord = true
	}
}

// waitForMore shows the paging cursor until a key turns the page.
func (g *Game) waitForMore() {
	g.flushOnGetch = false
	g.promptChar = morePrompt
	for {
		k := g.getch()
		if k == ' ' || k == '\r' || k == keyEscape || k == cmdPass {
			break
		}
	}
	g.promptChar = promptChar
	g.flushOnGetch = true
}

func toUpper(ch int) int {
	if ch >= 'a' && ch <= 'z' {
		return ch - 'a' + 'A'
	}
	return ch
}

// putchRepeat prints ch n times.
// C: seg_0C9C:32DE (C_0C9C_32DE)
func (g *Game) putchRepeat(n int, ch int) {
	for range n {
		g.putch0(ch)
	}
}

// putsRaw prints s without control characters.
// C: seg_0C9C:32F8 (C_0C9C_32F8)
func (g *Game) putsRaw(s string) {
	for i := 0; i < len(s); i++ {
		g.putch0(int(s[i]))
	}
}

// formatNumber writes number in base (negative: signed) with at most maxLen digits.
// C: seg_0C9C:331A (CON_atoi)
func formatNumber(number uint16, maxLen, base int) string {
	negative := false
	if base < 0 {
		base = -base
		if number > 1<<15-1 {
			number = -number
			negative = true
			maxLen--
		}
	}
	var digits []byte
	for number != 0 && len(digits) < maxLen {
		d := int(number) % base
		if d < 10 {
			digits = append(digits, byte('0'+d))
		} else {
			digits = append(digits, byte('A'+d-10))
		}
		number /= uint16(base)
	}
	if len(digits) == 0 {
		digits = append(digits, '0')
	}
	if negative {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// variableIndex is the index of a conversation variable named by a digit or letter.
func variableIndex(ch int) int {
	ch = toUpper(ch)
	switch {
	case ch >= '0' && ch <= '9':
		return ch - '0'
	case ch >= 'A' && ch <= 'Z':
		return ch - 'A' + 10
	}
	return 0
}

const keywordEnd = " ,.:;!?'-\"\n"

// putch prints a character, interpreting the control characters: '#' and '$' print a
// conversation variable, '/' and '\' select the plural or singular ending, '^' a control
// character, '@' highlights a keyword, '*' waits for a key, '&' hides text, '<' and '>'
// switch to runes and back, '`' capitalizes the next letter.
// C: seg_0C9C:33D3 (CON_putch)
func (g *Game) putch(ch int) {
	if !g.TextOn {
		return
	}
	if ch != '&' && g.hideText && g.SpeakGargish == 0 {
		return
	}
	switch ch {
	case '#', '$', '%', '/', '@', '\\', '^':
		g.control = byte(ch)
		return
	case '*':
		g.putch0(endOfLine)
		g.promptChar = morePrompt
		g.getch()
		g.promptChar = promptChar
		return
	case '&':
		g.hideText = !g.hideText
		return
	case '<':
		g.current.attr |= clipHighlight
		return
	case '>':
		g.current.attr &^= clipHighlight
		return
	case '`':
		g.capitalizeNext = true
		return
	}
	isAlpha := ch < unicode.MaxASCII && unicode.IsLetter(rune(ch))
	switch g.control {
	case '#':
		v := g.VarInt[variableIndex(ch)]
		g.plural = v != 1
		g.putsRaw(formatNumber(uint16(v), 6, -10))
		g.control = 0
	case '$':
		g.putsRaw(g.VarStr[variableIndex(ch)])
		g.control = 0
	case '/':
		if !isAlpha || !g.plural {
			g.putch0(ch)
			if !isAlpha {
				g.control = 0
			}
		}
	case '\\':
		if !isAlpha || g.plural {
			g.putch0(ch)
			if !isAlpha {
				g.control = 0
			}
		}
	case '^':
		const controlMask = 31 // '^' makes the next character a control character
		if g.current.attr&clipHighlight != 0 {
			g.putch0(ch&controlMask | highBit)
		} else {
			g.putch0(ch & controlMask)
		}
		g.control = 0
	case '@':
		if strings.IndexByte(keywordEnd, byte(ch)) >= 0 {
			g.control = 0
			g.current.fg = g.colours.text
		} else if g.Help {
			g.current.fg = g.colours.highlight
		}
		g.putch0(ch)
	default:
		g.putch0(ch)
	}
}

// puts prints s with its control characters.
// C: seg_0C9C:3621 (CON_puts)
func (g *Game) puts(s string) {
	for i := 0; i < len(s); i++ {
		g.putch(int(s[i]))
	}
}

// printf prints format, replacing %d, %o, %x, %u (int arguments), %c (a byte or rune)
// and %s (a string) with optional '0' fill, '-' left alignment and width; other
// characters after '%' are printed as they are.
// C: seg_0C9C:3643 (CON_printf)
func (g *Game) printf(format string, args ...any) {
	if !g.TextOn {
		return
	}
	next := func() any {
		if len(args) == 0 {
			return 0
		}
		a := args[0]
		args = args[1:]
		return a
	}
	number := func() uint16 {
		switch v := next().(type) {
		case int:
			return uint16(v)
		case uint16:
			return v
		case int16:
			return uint16(v)
		case byte:
			return uint16(v)
		case int8:
			return uint16(v)
		}
		return 0
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			g.putch(int(format[i]))
			continue
		}
		i++
		fill, left, width := byte(' '), false, 0
		for ; i < len(format) && (format[i] == '0' || format[i] == '-'); i++ {
			if format[i] == '0' {
				fill = '0'
			} else {
				left = true
			}
		}
		for ; i < len(format) && format[i] >= '0' && format[i] <= '9'; i++ {
			width = width*10 + int(format[i]-'0')
		}
		if i >= len(format) {
			break
		}
		// the padding counts the digits of numbers and the length of strings, not
		// characters (%c)
		text, length := "", 0
		switch unicode.ToLower(rune(format[i])) {
		case 'd':
			n := number()
			text = formatNumber(n, 6, -10)
			length = len(text)
			g.plural = n != 1
		case 'o':
			n := number()
			text = formatNumber(n, 6, 8)
			length = len(text)
			g.plural = n != 1
		case 'x':
			n := number()
			text = formatNumber(n, 4, 16)
			length = len(text)
			g.plural = n != 1
		case 'u':
			n := number()
			text = formatNumber(n, 5, 10)
			length = len(text)
			g.plural = n != 1
		case 'c':
			switch v := next().(type) {
			case int:
				text = string(rune(byte(v)))
			case byte:
				text = string(rune(v))
			case rune:
				text = string(rune(byte(v)))
			}
		case 's':
			text, _ = next().(string)
			length = len(text)
		default:
			g.putch0(int(format[i]))
		}
		if left {
			g.puts(text)
		}
		g.putchRepeat(width-length, int(fill))
		if !left {
			g.puts(text)
		}
	}
}

// echo shows a typed character (or erases one for backspace) at the cursor.
// C: seg_0C9C:0F52 (C_0C9C_0F52)
func (g *Game) echo(ch int) {
	c := g.current
	x, y := c.curX+c.x0, c.curY+c.y0
	if ch == '\b' {
		if c.curX != 0 {
			c.curX--
			x--
		} else if c.curY != 0 {
			c.curY--
			y--
			c.curX = c.width - 1
			x = c.curX + c.x0
		}
		g.putCharAt(' ', x, y)
		return
	}
	g.putCharAt(byte(ch), x, y)
	if c.curX++; c.curX >= c.width {
		c.curX = 0
		if c.curY < c.height-1 {
			c.curY++
		} else if c.attr&clipScroll != 0 {
			g.scrollUp(c.x0, c.y0, c.width, c.height)
		}
	}
}

// gets reads a line of at most maxLen characters typed at the prompt.
// C: seg_0C9C:2D3A (CON_gets)
func (g *Game) gets(maxLen int) string {
	g.typingString = true
	g.uppercaseKeys = false
	g.putch0(endOfLine)
	var typed []byte
	g.drainKeys = false
	for done := false; !done; {
		ch := 0
		for ch == 0 {
			ch = g.prompt()
			if g.optionPicked { // the port: a click on a conversation option types it
				g.optionPicked = false
				for range typed {
					g.echo('\b')
				}
				typed = typed[:0]
				ch = '\r'
			}
		}
		switch {
		case ch == keyUp && len(g.talkOptions) > 0:
			g.chooseOption(g.talkChoice - optionsColumns)
		case ch == keyDown && len(g.talkOptions) > 0:
			g.chooseOption(g.talkChoice + optionsColumns)
		case ch == keyLeft && len(g.talkOptions) > 0:
			g.chooseOption(g.talkChoice - 1)
		case ch == keyRight && len(g.talkOptions) > 0:
			g.chooseOption(g.talkChoice + 1)
		case ch == '\r':
			if len(typed) == 0 && len(g.talkOptions) > 0 {
				typed = []byte(g.talkOptions[g.talkChoice])
				for _, c := range typed {
					g.echo(int(c))
				}
			}
			g.cancelled = false
			done = true
		case ch == keyEscape:
			for range typed {
				g.echo('\b')
			}
			typed = typed[:0]
			g.cancelled = true
			done = true
		case ch == keyBackspace:
			if len(typed) > 0 {
				typed = typed[:len(typed)-1]
				g.echo('\b')
			}
		default:
			if len(typed) < maxLen && ch >= ' ' && ch <= '~' {
				typed = append(typed, byte(ch))
				g.echo(ch)
			}
		}
	}
	g.drainKeys = true
	g.linesLeft = g.current.height
	g.uppercaseKeys = true
	g.typingString = false
	return string(typed)
}
