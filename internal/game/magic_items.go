package game

import (
	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// useStaff casts the first charge of staff n.
// C: seg_27A1:3537 (C_27A1_3537)
func (g *Game) useStaff(n int) {
	charge := g.FindInv(n)
	if charge == noObject {
		g.printf("\n%s\n", noEffectMsg)
		return
	}
	spell := g.object(charge).Quality()
	g.printf("With ")
	g.printf("%s", incantation(spell))
	if g.SelectMode != 0 {
		g.SelectRange = widerRange
	}
	g.castSpell(spell, false)
	g.takeObj(n, ObjCharge, qualQuan(spell, 1))
}

// Where the Codex lies between the lenses.
const (
	codexX, codexY  = 923, 851
	codexLensLeftX  = 921
	codexLensRightX = 925
	moonstonePhases = 8
	vortexReach     = 5
)

// openVortex ends the game when the vortex cube n holds the eight moonstones and is used
// by the Codex between the two lenses; it returns false otherwise.
// C: seg_27A1:5FAC (C_27A1_5FAC)
func (g *Game) openVortex(n int) bool {
	var held [moonstonePhases]bool
	for i := g.FindInv(n); i >= 0; i = g.NextInv() {
		held[g.object(i).Frame()] = true
	}
	for _, h := range held {
		if !h {
			return false
		}
	}
	if g.typeAt(codexLensLeftX, codexY, 0, ObjBritanniaLens) == noObject ||
		g.typeAt(codexLensRightX, codexY, 0, ObjGargoyleLens) == noObject {
		return false
	}
	codex := g.typeAt(codexX, codexY, 0, ObjCodex)
	if codex == noObject {
		return false
	}
	mx, my := int(g.MapX), int(g.MapY)
	if mx-vortexReach > codexX || mx+vortexReach < codexX || my-vortexReach > codexY || my+vortexReach < codexY {
		return false
	}
	g.DeleteObj(codex)
	g.vortex()
	g.printf("\nThe Codex has vanished!\n")
	g.getch()
	g.goToEnding()
	return true
}

// vortex swirls the view.
// C: seg_2FC1:1BCE (C_2FC1_1BCE)
func (g *Game) vortex() {
	g.drawStatusCursor(0)
	g.SelectMode = 0
	g.drawMap(false)
	g.gr.Vortex(g.world[:], host.ScreenWidth, host.ScreenHeight, viewCropX, viewCropY)
	g.redrawWorld()
	g.screenFade = true
	g.drawMap(true)
}

// gazeAt asks for a latitude and longitude and shows the place (the crystal ball).
// C: seg_27A1:5935 (C_27A1_5935), after the intelligence test
func (g *Game) gazeAt() {
	if g.SelectMode != 0 {
		g.drawStatusCursor(0)
		g.SelectMode = 0
	}
	g.printf("\nEnter degrees followed by N, S, E or W.\n")
	g.printf("\nAt latitude=")
	row := g.current.curY
	const latitude, longitude = 1, 2
	lat := g.parseNumber(g.gets(4), latitude)
	const northmost, southmost, westmost, eastmost = -44, 80, -37, 88
	if lat > southmost || lat < northmost {
		g.gotoxy(12, row)
		g.printf("0   \n")
		return
	}
	g.printf("\n  longitude=")
	row = g.current.curY
	long := g.parseNumber(g.gets(4), longitude)
	if long > eastmost || long < westmost {
		g.gotoxy(12, row)
		g.printf("0   \n")
		return
	}
	g.putch('\n')
	saved := g.SoundOn
	g.SoundOn = 0
	x := meridianX + long<<degreeShift
	y := equatorY + lat<<degreeShift
	if g.MapZ != 0 {
		x >>= 2
		y >>= 2
	}
	g.wizardEye(x, y, false)
	g.SoundOn = saved
}

// goTo asks for a place in hexadecimal and takes the party there (Alt-214).
// C: seg_0A33:0E8A (C_0A33_0E8A), case 214
func (g *Game) goTo() {
	if g.inVehicle() {
		g.printf("<nat uail abord wip!>\n")
		return
	}
	g.printf("<gotu eks>: ")
	x := g.getHex(3)
	g.printf("\n<uai>: ")
	y := g.getHex(3)
	g.printf("\n<zi>: ")
	z := g.getHex(1)
	if z != 0 {
		x &= byteMask
		y &= byteMask
	} else {
		x &= coordMask
		y &= coordMask
	}
	z = min(z, gargoyleLevel)
	g.partyTeleport(x, y, z, 0)
	g.playAreaSong()
}

// getHex reads a hexadecimal number of up to n digits.
// C: seg_0A33:1C4C (__GetHex)
func (g *Game) getHex(n int) int {
	n = min(n, 4)
	v := 0
	for _, c := range g.gets(n) {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v += int(c - '0')
		case c >= 'a' && c <= 'z':
			v += int(c - 'a' + 10)
		default:
			v += int(c - 'A' + 10)
		}
	}
	return v
}

// Peering draws each tile of the map as a dot of 4x4 pixels.
const (
	peerCell        = 4
	peerSize        = 40 // the cells across the view
	peerPixels      = peerSize * peerCell
	peerOrigin      = 8 // where the map is drawn on the back buffer
	white           = 15
	black           = 0
	peerColour      = 7 // impassable dry land (D_2A5A)
	unseenTile      = 255
	lastTerrainTile = 255
)

// peer shows the map around the party, a dot per tile, until a key.
// C: seg_2FC1:1FD6 (C_2FC1_1FD6)
func (g *Game) peer() {
	underground := g.MapZ > 0 && g.MapZ < gargoyleLevel
	g.areaFlags = [areaSize * areaSize]byte{}
	g.areaLight = [areaSize * areaSize]byte{}
	top, left, bottom, right := 0, 0, peerSize, peerSize
	if underground {
		g.computeLight(g.ViewX+viewCenterX, g.ViewY+viewCenterY, peerSize, lightPeer)
	} else if g.MapZ == 0 {
		const (
			edge        = 1024      // the width of the surface
			lastArea    = edge - 16 // the area that wraps round the edge
			beforeLast  = edge - 24
			edgeInView  = edge - 32 // the first area that reaches the edge
			wrappedLeft = 8
		)
		switch {
		case g.AreaX == lastArea:
			left = wrappedLeft
		case g.AreaX == edgeInView:
			right = edge - edgeInView
		case g.AreaY >= lastArea:
			top = edge - g.AreaY
		case g.AreaY == edgeInView || g.AreaY == beforeLast:
			bottom = edge - g.AreaY
		}
	}
	g.gr.Offscreen = true
	g.gr.SetColor(black)
	g.gr.FillRect(peerOrigin, peerOrigin, peerOrigin+peerPixels-1, peerOrigin+peerPixels-1)
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			t := int(g.AreaTiles[y*areaSize+x])
			if underground && g.areaFlags[y*areaSize+x]&areaVisible == 0 {
				t = unseenTile
			}
			g.peerTile(t, x*peerCell+peerOrigin, y*peerCell+peerOrigin)
		}
	}
	for n := g.SearchArea(g.AreaX, g.AreaY, g.AreaX+areaSize-1, g.AreaY+areaSize-1); n >= 0; n = g.NextArea() {
		if n == 0 && !g.isPlayerControlled(n) {
			continue
		}
		p := g.object(n).Pos
		x, y := p.X-g.AreaX, p.Y-g.AreaY
		if underground && g.areaFlags[y*areaSize+x]&areaVisible == 0 {
			continue
		}
		px, py := x*peerCell+peerOrigin, y*peerCell+peerOrigin
		if n < npcCount {
			g.peerTile(-1, px, py)
			continue
		}
		t := g.tileFrame(n)
		g.peerTile(t, px, py)
		wide, tall := g.TileFlag[t]&tileDoubleWidth != 0, g.TileFlag[t]&tileDoubleHeight != 0
		if wide && x != 0 {
			if tall && y != 0 {
				g.peerTile(t-2, px, py-peerCell)
				g.peerTile(t-3, px-peerCell, py-peerCell)
			}
			g.peerTile(t-1, px-peerCell, py)
		} else if tall && y != 0 {
			g.peerTile(t-1, px, py-peerCell)
		}
	}
	g.gr.SetColor(white)
	partyX := (g.ViewX+viewCenterX)*peerCell + peerOrigin
	partyY := (g.ViewY+viewCenterY)*peerCell + peerOrigin
	g.gr.FillRect(partyX, partyY, partyX+peerCell-1, partyY+peerCell-1)
	g.gr.SetColor(black)
	g.gr.Offscreen = false
	// centred on the screen
	g.copyView(peerOrigin, peerOrigin, peerOrigin+peerPixels-1, peerOrigin+peerPixels-1,
		viewCropX+(host.ScreenWidth-peerPixels)/2, viewCropY+(host.ScreenHeight-peerPixels)/2)
	g.animatePrompt = false
	g.getch()
	g.animatePrompt = true
	g.computeView()
}

// peerTile draws the dot of a tile; -1 is a character.
// C: seg_2FC1:1F14 (peer_tile)
func (g *Game) peerTile(tile, x, y int) {
	pattern := 1
	colour := -1
	switch {
	case tile < 0:
		colour, pattern = white, 0
	case tile == unseenTile:
		colour = black
	case g.TerrainType[tile]&terrainDamaging != 0:
		colour = int(g.colours.highlight)
	case g.TerrainType[tile]&terrainImpassable != 0:
		if g.TerrainType[tile]&terrainWet != 0 {
			colour = int(g.colours.defaultCommand)
		} else {
			colour = peerColour
		}
	}
	if colour < 0 && tile > lastTerrainTile {
		return
	}
	if colour < 0 {
		colour = int(g.colours.poisoned)
	}
	g.gr.SetColor(byte(colour))
	for i := range peerCell {
		for j := range peerCell {
			if (i+j)&1 == pattern {
				g.gr.PutPixel(i+x, j+y)
			}
		}
	}
}

// Hail Storm drops up to six stones at a time.
const (
	hailSlots  = 6
	hailSprite = 7
	hitSprite  = 6
	tileHail   = 398 // TIL_18E
	hailFall   = 50  // how far a stone falls, in pixels
	hailFrames = 4   // the stones are drawn every fourth step
	hailPivot  = 127
	hailOdds   = 7
	hailSpread = 30
	hailCentre = 23
	hailMargin = 15 // the stones are drawn this far inside the view
)

// hailStorm drops hail around (x, y); it returns the experience of the hits.
// C: seg_2FC1:13B0 (C_2FC1_13B0)
func (g *Game) hailStorm(x, y int) int {
	falling := false
	step := 0
	xp := 0
	var fromX, fromY, toX, toY [hailSlots]int
	for i := range hailSlots {
		fromX[i], fromY[i], toX[i], toY[i] = -1, -1, -1, -1
	}
	const fewestStones, mostStones = 10, 30
	stones := g.rand(fewestStones, mostStones)
	g.gr.MakeRotatedSprite(hailSprite, tileHail, 0, hailPivot, hailPivot)
	empty := emptyClip
	r := empty
	viewX := (int(g.MapX) - viewCenterX) * tileSize
	viewY := (int(g.MapY) - viewCenterY) * tileSize
	inView := func(i int) (int, int, bool) {
		sx, sy := fromX[i]-viewX, fromY[i]-viewY
		return sx, sy, sx > hailMargin && sx < viewPixelEndX-hailMargin-1 && sy > hailMargin && sy < viewPixelEndY-hailMargin-1
	}
	for stones != 0 || falling {
		noise := 0
		free := 0
		for free < hailSlots && fromX[free] >= 0 {
			free++
		}
		if stones != 0 && free < hailSlots && g.rand(0, hailOdds) == 0 {
			toX[free] = g.rand(1, hailSpread) + x*tileSize + g.rand(1, hailSpread) - hailCentre
			toY[free] = g.rand(1, hailSpread) + y*tileSize + g.rand(1, hailSpread) - hailCentre
			fromX[free], fromY[free] = toX[free]-hailFall, toY[free]-hailFall
			stones--
		}
		falling = false
		if step++; step >= hailFrames {
			for i := range hailSlots {
				if sx, sy, ok := inView(i); fromX[i] >= 0 && ok {
					g.gr.SaveUnderSprite(i, sx, sy)
					r.grow(sx, sy)
				}
			}
			for i := range hailSlots {
				if sx, sy, ok := inView(i); fromX[i] >= 0 && ok {
					g.gr.DrawSprite(hailSprite, sx, sy)
					r.grow(sx, sy)
				}
			}
			if r.x0 <= r.x1 && r.y0 <= r.y1 {
				g.copyView(r.x0, r.y0, r.x1, r.y1, r.x0, r.y0)
			}
			r = empty
			for i := range hailSlots {
				if sx, sy, ok := inView(i); fromX[i] >= 0 && ok {
					g.gr.DrawSprite(i, sx, sy)
					r.grow(sx, sy)
				}
			}
			step = 0
		}
		for i := range hailSlots {
			if fromX[i] < 0 {
				continue
			}
			fromX[i]++
			fromY[i]++
			if toX[i] != fromX[i] || toY[i] != fromY[i] {
				falling = true
				continue
			}
			fromX[i] = -1
			const lowNoise, highNoise = 400, 500
			noise = g.rand(lowNoise, highNoise)
			tx, ty := toX[i]/tileSize, toY[i]/tileSize
			for n := g.FindLoc(tx, ty, int(g.MapZ)); n >= 0; n = g.NextLoc() {
				if n >= npcCount {
					continue
				}
				g.hideNextHit = true
				xp += g.loseHitPoints(n, 1)
				noise = -1
				sx := (tx-(int(g.MapX)-viewCenterX))*tileSize + halfTile
				sy := (ty-(int(g.MapY)-viewCenterY))*tileSize + halfTile
				if sx >= viewPixelStartX && sx < viewPixelEndX && sy >= viewPixelStartY && sy < viewPixelEndY {
					g.gr.SaveUnderSprite(hitSprite, sx, sy)
					g.gr.DrawTileOffscreen(tileHit, sx-halfTile, sy-halfTile)
					g.copyView(sx-halfTile, sy-halfTile, sx+halfTile-1, sy+halfTile-1, sx-halfTile, sy-halfTile)
					g.gr.DrawSprite(hitSprite, sx, sy)
					r.grow(sx, sy)
				}
			}
		}
		g.sfx(sound.EffectImpact, noise)
	}
	g.OtherAnimations()
	return xp
}
