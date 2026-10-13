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

	// the clip of the conversation options, in character cells, and how they are laid out
	clipOptions     = 2
	optionsX0       = 4
	optionsY0       = 39
	optionsColumns  = 3
	optionWidth     = 12 // characters per option column
	optionsRows     = 8
	optionsShownMax = optionsColumns * optionsRows
	// the clip the conversation is printed in while talking
	speechX0, speechY0 = 4, 26
	speechColumns      = 28
	speechRows         = 12

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
	optionsPanel = cellRect(optionsX0, optionsY0, optionsColumns*optionWidth, optionsRows)
)

// The scroll in PAPER.BMP (the message scroll's top left part) that the panels' scrolls are
// cut from: its top border is drawn mirrored for the bottom one.
var (
	scrollSource = image.Rect(171, 106, 320, 200)
	scrollInner  = image.Rect(176, 112, 312, 200)
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
	// talkOptions are the keywords offered at the conversation's prompt, talkChoice the one
	// highlighted; optionPicked is set by a click on one
	talkOptions  []string
	talkChoice   int
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
		const top = margin + scrollPadding
		talk := image.Rect(margin, margin, right, top+speechPanel.Dy()+scrollPadding)
		scrolls = append(scrolls, talk)
		textX := talk.Min.X + scrollPadding
		if !g.talkPortrait.Empty() {
			panels = append(panels, panel{g.talkPortrait, image.Pt(textX, top)})
			textX += g.talkPortrait.Dx() + scrollPadding
		}
		panels = append(panels, panel{speechPanel, image.Pt(textX, top)})
		if len(g.talkOptions) > 0 {
			add(true, optionsPanel, image.Pt(talk.Min.X+scrollPadding, talk.Max.Y+margin+scrollPadding))
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
	right := scrollSource.Max.X - scrollInner.Max.X
	r = r.Intersect(image.Rect(0, 0, host.ScreenWidth, host.ScreenHeight))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		var from int
		switch {
		case y-r.Min.Y < top:
			from = scrollSource.Min.Y + y - r.Min.Y
		case r.Max.Y-1-y < top:
			from = scrollSource.Min.Y + r.Max.Y - 1 - y
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
				g.pointOption(at, host.MouseLeft && !g.mouse.left)
			}
			return at.X, at.Y, true
		}
	}
	return x + viewCropX, y + viewCropY, false
}

// pointOption highlights the option at canvas point at, and picks it on a click.
func (g *Game) pointOption(at image.Point, clicked bool) {
	column := (at.X - optionsPanel.Min.X) / (optionWidth * cellSize)
	i := column*optionsRows + (at.Y-optionsPanel.Min.Y)/cellSize
	if i >= len(g.talkOptions) || i >= optionsShownMax {
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

// startTalkLayout prints the conversation in its own clip next to the portrait.
func (g *Game) startTalkLayout() clip {
	g.putch0(endOfLine)
	saved := g.clips[clipMessage]
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(speechPanel.Min.X, speechPanel.Min.Y, speechPanel.Max.X-1, speechPanel.Max.Y-1)
	c := &g.clips[clipMessage]
	c.x0, c.y0, c.width, c.height, c.curX, c.curY = speechX0, speechY0, speechColumns, speechRows, 0, 0
	g.talkPortrait = image.Rectangle{}
	return saved
}

// endTalkLayout goes back to printing in the log.
func (g *Game) endTalkLayout(saved clip) {
	g.putch0(endOfLine)
	g.clips[clipMessage] = saved
	g.talkOptions = nil
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
	g.talkChoice = 0
	for i, w := range g.talkOptions {
		if w == "bye" {
			g.talkChoice = i
		}
	}
	g.drawTalkOptions()
}

// drawTalkOptions writes the options on the canvas in columns, the chosen one highlighted.
func (g *Game) drawTalkOptions() {
	saved := g.current
	g.current = &g.clips[clipOptions]
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(optionsPanel.Min.X, optionsPanel.Min.Y, optionsPanel.Max.X-1, optionsPanel.Max.Y-1)
	for i, w := range g.talkOptions[:min(len(g.talkOptions), optionsShownMax)] {
		g.current.fg = g.colours.text
		if i == g.talkChoice {
			g.current.fg = g.colours.highlight
		}
		x, y := optionsX0+i/optionsRows*optionWidth, optionsY0+i%optionsRows
		for j := range min(len(w), optionWidth-1) {
			g.putCharAt(w[j], x+j, y)
		}
	}
	g.current = saved
}
