package game

import (
	"image"
	"strings"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
)

// The port's screen layout: the map view fills the screen and the panels float over it.
// The game still draws its panels on the canvas (the driver's screen, taller than what is
// shown) where the original layout had them, and the port's new panels below the shown
// part; Frame composes the world picture with copies of those panels, and pointAt maps the
// pointer back onto the canvas.

const (
	// the back buffer's pixels left out at each side of the shown view
	viewCropX = (vga.BackBufferWidth - host.ScreenWidth) / 2
	viewCropY = (vga.BackBufferHeight - host.ScreenHeight) / 2
	// the part of the back buffer that is shown
	viewPixelStartX = viewCropX
	viewPixelEndX   = viewCropX + host.ScreenWidth
	viewPixelStartY = viewCropY
	viewPixelEndY   = viewCropY + host.ScreenHeight
	// lightFirst and lightLast bound the area places whose light and sight are computed: the
	// view, a tile around it and the lamps near it
	lightFirst = 4
	lightLast  = areaSize - lightFirst

	// the clip the conversation is printed in while talking, in character cells, right of
	// the party column; its last row only holds the cursor
	speechX0, speechY0 = 3, 26
	speechColumns      = 37
	speechRows         = 12
	// the text flows beside the portrait (and a column of room) for its rows
	portraitColumns = portraitWidth/cellSize + 1
	portraitRows    = portraitHeight / cellSize
	// the clip of the conversation options, in character cells, and how they are laid out:
	// the row after the options is where the player types
	clipOptions    = 2
	optionsX0      = speechX0
	optionsY0      = speechY0 + speechRows + 1
	optionsColumns = 3
	optionWidth    = 12 // characters per option column
	optionsRows    = 6
	// the marks of more options above or below, in the column right of the options
	moreAboveChar, moreBelowChar = 24, 25
	moreColumn                   = optionsX0 + optionsColumns*optionWidth

	// margin is the room left around panels and between them and the screen's edges
	margin = 4
	// scrollPadding is the room between a scroll's edge and what it holds
	scrollPadding = 8
	// the dock slides up this many pixels a frame
	dockSlide = 4
)

// Hot keys of the port's panels (Ctrl+D, Ctrl+L): U6 uses neither.
const (
	keyCtrlD = 4
	keyCtrlL = 12
)

// Canvas rectangles of the panels.
var (
	skyPanel     = image.Rect(176, 6, 312, 22)
	statusPanel  = image.Rect(statusX0, statusY0, statusX1+1, statusY1+1)
	datePanel    = image.Rect(16, 168, 160, 176)
	iconsPanel   = image.Rect(4, 174, 172, 197)
	partyPanel   = image.Rect(partyColumnX, partyColumnY, partyColumnX+tileSize, partyColumnY+partyListRows*partyRowHeight)
	speechPanel  = cellRect(speechX0, speechY0, speechColumns, speechRows)
	optionsPanel = cellRect(optionsX0, optionsY0, speechColumns, optionsRows+1)
	optionsList  = cellRect(optionsX0, optionsY0, speechColumns, optionsRows)
)

// The message scroll in PAPER.BMP that the panels' scrolls are cut from: its borders and
// the inside, which is repeated to fill.
var (
	scrollSource = image.Rect(171, 106, 315, 196)
	scrollInner  = image.Rect(176, 112, 312, 192)
)

func cellRect(x, y, columns, rows int) image.Rectangle {
	return image.Rect(x*cellSize, y*cellSize, (x+columns)*cellSize, (y+rows)*cellSize)
}

// layout is the state of the port's screen layout.
type layout struct {
	// world is the shown part of the map view, copied from the back buffer
	world [host.ScreenWidth * host.ScreenHeight]byte
	// shown is the composed screen Frame returns
	shown [host.ScreenWidth * host.ScreenHeight]byte
	paper *vga.Pixmap
	// showDate and showLog are toggled by the hot keys
	showDate, showLog bool
	// dockOffset is how far the command icons have slid up into the screen
	dockOffset int
	// prompting is set while the game waits for a key, atMainPrompt while it waits for a
	// command: other prompts show the log
	prompting, atMainPrompt bool
	// talkPortrait is the canvas rectangle of the portrait last drawn
	talkPortrait image.Rectangle
	// pageFull is set when the conversation's page is full: the next text waits for a key
	// and starts a new page
	pageFull bool
	// talkOptions are the keywords offered at the conversation's prompt, talkChoice the one
	// highlighted; optionPicked is set by a click on one
	talkOptions  []string
	talkChoice   int
	talkTop      int // the first row of options shown; they scroll
	optionPicked bool
}

// panel is a canvas rectangle shown with its corner at a screen point.
type panel struct {
	from image.Rectangle
	at   image.Point
}

func (p panel) onScreen() image.Rectangle { return p.from.Sub(p.from.Min).Add(p.at) }

// copyView copies the back buffer's rectangle from (x0, y0) to (x1, y1) (inclusive) to the
// world picture, its corner at back buffer place (toX, toY).
func (g *Game) copyView(x0, y0, x1, y1, toX, toY int) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			sx, sy := toX+x-x0-viewCropX, toY+y-y0-viewCropY
			if x >= 0 && y >= 0 && x < vga.BackBufferWidth && y < vga.BackBufferHeight &&
				sx >= 0 && sy >= 0 && sx < host.ScreenWidth && sy < host.ScreenHeight {
				g.world[sy*host.ScreenWidth+sx] = g.gr.BackBuffer[y*vga.BackBufferWidth+x]
			}
		}
	}
}

// panels returns the scrolls and then the panels shown over the world, the topmost last.
func (g *Game) panels() (scrolls []image.Rectangle, panels []panel) {
	add := func(scroll bool, from image.Rectangle, at image.Point) {
		p := panel{from, at}
		if scroll {
			scrolls = append(scrolls, p.onScreen().Inset(-scrollPadding))
		}
		panels = append(panels, p)
	}
	right := host.ScreenWidth - margin
	if g.InConversation {
		at := image.Pt(margin+scrollPadding, margin+scrollPadding)
		add(true, speechPanel, at)
		if !g.talkPortrait.Empty() {
			add(false, g.talkPortrait, at)
		}
		if len(g.talkOptions) > 0 {
			add(true, optionsPanel, image.Pt(at.X, host.ScreenHeight-margin-scrollPadding-optionsPanel.Dy()))
		}
		return scrolls, panels
	}
	if g.StatusDisplay == cmdStatusParty {
		add(false, skyPanel, image.Pt((host.ScreenWidth-skyPanel.Dx())/2, 0))
		add(false, partyPanel, image.Pt(right-partyPanel.Dx(), margin))
	} else {
		add(true, statusPanel, image.Pt(right-scrollPadding-statusPanel.Dx(), margin+scrollPadding))
	}
	if g.showDate {
		add(true, datePanel, image.Pt((host.ScreenWidth-datePanel.Dx())/2, skyPanel.Dy()+scrollPadding+margin))
	}
	if log := g.logPanel(); !log.Empty() && (g.showLog || g.prompting && !g.atMainPrompt) {
		at := image.Pt(right-tileSize-margin-scrollPadding-log.Dx(), host.ScreenHeight-margin-scrollPadding-log.Dy())
		add(true, log, at)
	}
	if g.dockOffset > 0 {
		add(false, iconsPanel, image.Pt((host.ScreenWidth-iconsPanel.Dx())/2, host.ScreenHeight-g.dockOffset))
	}
	return scrolls, panels
}

// logPanel is the part of the message clip with text: at the command prompt, not its
// last line, which only holds the prompt.
func (g *Game) logPanel() image.Rectangle {
	c := &g.clips[clipMessage]
	rows := c.curY
	if !g.atMainPrompt {
		rows++
	}
	return cellRect(c.x0, c.y0, c.width, rows)
}

// Frame composes the screen for the host: the world with the panels over it.
func (g *Game) Frame() *host.Frame {
	g.shown = g.world
	scrolls, panels := g.panels()
	for _, s := range scrolls {
		g.drawScroll(s)
	}
	for _, p := range panels {
		for y := p.from.Min.Y; y < p.from.Max.Y; y++ {
			sy := y - p.from.Min.Y + p.at.Y
			if sy < 0 || sy >= host.ScreenHeight {
				continue
			}
			row := g.gr.Screen[y*vga.ScreenWidth+p.from.Min.X : y*vga.ScreenWidth+p.from.Max.X]
			copy(g.shown[sy*host.ScreenWidth+p.at.X:(sy+1)*host.ScreenWidth], row)
		}
	}
	frame := g.gr.Frame()
	frame.Pixels = g.shown[:]
	return frame
}

// drawScroll draws a scroll filling r, cut from PAPER.BMP's message scroll.
func (g *Game) drawScroll(r image.Rectangle) {
	if g.paper == nil {
		return
	}
	left, top := scrollInner.Min.X-scrollSource.Min.X, scrollInner.Min.Y-scrollSource.Min.Y
	right, bottom := scrollSource.Max.X-scrollInner.Max.X, scrollSource.Max.Y-scrollInner.Max.Y
	r = r.Intersect(image.Rect(0, 0, host.ScreenWidth, host.ScreenHeight))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		var from int
		switch {
		case y-r.Min.Y < top:
			from = scrollSource.Min.Y + y - r.Min.Y
		case r.Max.Y-1-y < bottom:
			from = scrollSource.Max.Y - (r.Max.Y - y)
		default:
			from = scrollInner.Min.Y + (y-r.Min.Y-top)%scrollInner.Dy()
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			var fromX int
			switch {
			case x-r.Min.X < left:
				fromX = scrollSource.Min.X + x - r.Min.X
			case r.Max.X-1-x < right:
				fromX = scrollInner.Max.X + right - (r.Max.X - x)
			default:
				fromX = scrollInner.Min.X + (x-r.Min.X-left)%scrollInner.Dx()
			}
			g.shown[y*host.ScreenWidth+x] = g.paper.Pixels[from*g.paper.Width+fromX]
		}
	}
}

// pointAt maps a screen point to the canvas when a panel shows it there (overUI), else to
// the back buffer. It also slides the dock, and highlights and picks conversation options.
func (g *Game) pointAt(x, y int) (int, int, bool) {
	dockHeight := iconsPanel.Dy()
	if y >= host.ScreenHeight-1 || g.dockOffset > 0 && y >= host.ScreenHeight-dockHeight {
		g.dockOffset = min(g.dockOffset+dockSlide, dockHeight)
	} else {
		g.dockOffset = max(g.dockOffset-dockSlide, 0)
	}
	_, panels := g.panels()
	for i := len(panels) - 1; i >= 0; i-- {
		p := panels[i]
		if pt := image.Pt(x, y); pt.In(p.onScreen()) {
			at := pt.Sub(p.at).Add(p.from.Min)
			if p.from == optionsPanel {
				g.scrollOptions(-host.TakeWheel())
				g.pointOption(at, host.MouseLeft && !g.mouse.left)
			}
			return at.X, at.Y, true
		}
	}
	return x + viewCropX, y + viewCropY, false
}

// pointOption highlights the option at canvas point at, and picks it on a click.
func (g *Game) pointOption(at image.Point, clicked bool) {
	column, row := (at.X-optionsList.Min.X)/(optionWidth*cellSize), (at.Y-optionsList.Min.Y)/cellSize
	i := (g.talkTop+row)*optionsColumns + column
	if row >= optionsRows || column >= optionsColumns || i >= len(g.talkOptions) {
		return
	}
	if i != g.talkChoice {
		g.talkChoice = i
		g.drawTalkOptions()
	}
	g.optionPicked = g.optionPicked || clicked
}

// hotKey toggles the date or the log for the port's hot keys and returns 0 for them.
func (g *Game) hotKey(ch int) int {
	switch ch {
	case keyCtrlD:
		g.showDate = !g.showDate
	case keyCtrlL:
		g.showLog = !g.showLog
	default:
		return ch
	}
	return 0
}

// startTalkLayout prints the conversation in pages on its own scroll, around the portrait.
func (g *Game) startTalkLayout() clip {
	g.putch0(endOfLine)
	saved := g.clips[clipMessage]
	c := &g.clips[clipMessage]
	c.y0, c.height = speechY0, speechRows
	c.attr &^= clipPaging // the pages wait for a key instead
	g.talkPortrait = image.Rectangle{}
	g.newSpeechPage()
	return saved
}

// endTalkLayout goes back to printing in the log.
func (g *Game) endTalkLayout(saved clip) {
	g.putch0(endOfLine)
	g.clips[clipMessage] = saved
	g.talkOptions = nil
	g.pageFull = false
}

// newSpeechPage clears the conversation's scroll and prints from its top.
func (g *Game) newSpeechPage() {
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(speechPanel.Min.X, speechPanel.Min.Y, speechPanel.Max.X-1, speechPanel.Max.Y-1)
	c := &g.clips[clipMessage]
	c.curX, c.curY = 0, 0
	g.lineLen, g.lineShown = 0, 0
	g.flowSpeech()
}

// flowSpeech fits the conversation's clip to its row: beside the portrait or below it. A
// full page leaves the cursor on its last row.
func (g *Game) flowSpeech() {
	c := &g.clips[clipMessage]
	if !g.InConversation || g.current != c {
		return
	}
	c.x0, c.width = speechX0, speechColumns
	if c.curY < portraitRows && !g.talkPortrait.Empty() {
		c.x0 += portraitColumns
		c.width -= portraitColumns
	}
	g.pageFull = c.curY >= c.height-1
}

// speechSkips is putch0's part in the conversation's pages: it drops the blank lines at the
// top of a page and starts a new page (after a key) for text that does not fit.
func (g *Game) speechSkips(ch int) bool {
	c := g.current
	if !g.InConversation || c != &g.clips[clipMessage] || ch < 0 {
		return false
	}
	newLine := ch == '\n' || ch == 0
	if g.pageFull {
		if newLine || ch == ' ' {
			return true
		}
		g.waitForMore()
		g.newSpeechPage()
		return false
	}
	if g.lineLen == 0 {
		g.flowSpeech()
	}
	return newLine && c.curY == 0 && g.lineLen == 0
}

// askTopic offers the topics and reads what the player says, typed on the topic scroll;
// the answer starts a new page.
func (g *Game) askTopic() string {
	g.offerOptions()
	speech := g.current
	g.current = &g.clips[clipOptions]
	g.current.curX, g.current.curY = 0, optionsRows
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(optionsList.Min.X, optionsList.Max.Y, optionsPanel.Max.X-1, optionsPanel.Max.Y-1)
	g.printf("you say:")
	typed := g.gets(typedLength)
	g.current = speech
	g.talkOptions = nil
	g.newSpeechPage()
	return typed
}

// topicWord is keyword spelled out as the first longer word of the conversation that starts
// with it (the script matches only the keyword's letters), or keyword itself.
func (g *Game) topicWord(keyword string) string {
	isLetter := func(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }
	for _, w := range strings.FieldsFunc(string(g.script), func(r rune) bool { return !isLetter(r) }) {
		w = strings.ToLower(w)
		if len(w) > len(keyword) && len(w) < optionWidth && strings.HasPrefix(w, keyword) {
			return w
		}
	}
	return keyword
}

// conversationKeywords returns the keywords the script answers at the prompt at pc, as
// matchKeywords walks them, and "look", which converse answers; a '*' answers anything,
// so the keys after it are never reached.
func (g *Game) conversationKeywords() []string {
	var words []string
	seen := map[string]bool{}
	add := func(w string) {
		if w != "" && !seen[strings.ToLower(w)] {
			seen[strings.ToLower(w)] = true
			words = append(words, strings.ToLower(w))
		}
	}
	for at := g.pc; g.scriptByte(at) == opKey; {
		end := at + 1
		for end < len(g.script) && g.script[end] != opRes {
			end++
		}
		catchAll := false
		for _, w := range strings.Split(string(g.script[at+1:end]), ",") {
			catchAll = catchAll || strings.HasPrefix(w, "*")
			if !strings.HasPrefix(w, "*") {
				add(w)
			}
		}
		if catchAll {
			break
		}
		for at = end + 1; at < len(g.script); at++ {
			op := g.script[at]
			if op == opKey || op == opEndRes || op == opFF {
				break
			}
			switch op {
			case opGoto:
				at += 4
			case opByte:
				at++
			case opWord:
				at += 2
			}
		}
	}
	add("look")
	return words
}

// offerOptions shows the keywords the script answers now, "bye" highlighted.
func (g *Game) offerOptions() {
	g.talkOptions = g.conversationKeywords()
	for i, w := range g.talkOptions {
		g.talkOptions[i] = g.topicWord(w)
	}
	g.talkChoice = 0
	for i, w := range g.talkOptions {
		if w == "bye" {
			g.talkChoice = i
		}
	}
	g.talkTop = 0
	g.chooseOption(g.talkChoice)
}

// optionRows is how many rows the options take.
func (g *Game) optionRows() int {
	return (len(g.talkOptions) + optionsColumns - 1) / optionsColumns
}

// chooseOption highlights option i (kept within the options), scrolling it into view.
func (g *Game) chooseOption(i int) {
	g.talkChoice = min(max(i, 0), len(g.talkOptions)-1)
	row := g.talkChoice / optionsColumns
	g.talkTop = min(g.talkTop, row)
	g.talkTop = max(g.talkTop, row-optionsRows+1)
	g.drawTalkOptions()
}

// scrollOptions scrolls the options by rows (negative: up).
func (g *Game) scrollOptions(rows int) {
	if rows == 0 {
		return
	}
	g.talkTop = min(max(g.talkTop+rows, 0), max(g.optionRows()-optionsRows, 0))
	g.drawTalkOptions()
}

// drawTalkOptions writes the shown rows of options on the canvas, the chosen one
// highlighted, and marks more rows above or below.
func (g *Game) drawTalkOptions() {
	saved := g.current
	g.current = &g.clips[clipOptions]
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(optionsList.Min.X, optionsList.Min.Y, optionsList.Max.X-1, optionsList.Max.Y-1)
	first := g.talkTop * optionsColumns
	for i, w := range g.talkOptions[first:min(len(g.talkOptions), first+optionsRows*optionsColumns)] {
		g.current.fg = g.colours.text
		if first+i == g.talkChoice {
			g.current.fg = g.colours.highlight
		}
		x, y := optionsX0+i%optionsColumns*optionWidth, optionsY0+i/optionsColumns
		for j := range min(len(w), optionWidth-1) {
			g.putCharAt(w[j], x+j, y)
		}
	}
	g.current.fg = g.colours.text
	if g.talkTop > 0 {
		g.putCharAt(moreAboveChar, moreColumn, optionsY0)
	}
	if g.talkTop+optionsRows < g.optionRows() {
		g.putCharAt(moreBelowChar, moreColumn, optionsY0+optionsRows-1)
	}
	g.current = saved
}
