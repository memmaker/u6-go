package game

// The map view: the 11x11 tiles around the party (Tile_11x11), the objects drawn on each
// (Obj_11x11, chains through D_E5E0), and drawing them on the back buffer.

// Tiles the view uses.
const (
	tileNumberMask           = 2047 // 0x7ff
	tileFlow                 = 8    // TIL_008, the first of the eight river tiles
	tileWallPatch            = 144  // TIL_090
	tileHidden               = 255  // TIL_0FF
	tileWallCorner           = 256  // TIL_100
	tileSundialOff           = 316  // TIL_13C
	tileSundial              = 328  // TIL_148
	tileMountains            = 352  // TIL_160
	tileSunLow               = 361  // TIL_169
	tileSun                  = 362  // TIL_16A
	tileSunEclipse           = 363  // TIL_16B
	tileAimDirection         = 364  // TIL_16C
	tileAimTarget            = 365  // TIL_16D
	tileCaveLeft             = 372  // TIL_174
	tileCave                 = 373  // TIL_175
	tileCaveRight            = 374  // TIL_176
	tileStatusPortraitButton = 384  // TIL_180
	tileStatusEquipButton    = 386  // TIL_182
	tileStatusPrevious       = 387  // TIL_183
	tileStatusNext           = 388  // TIL_184
	tileCommandIcons         = 400  // TIL_190
	tileStatusBackground     = 411  // TIL_19B
	tileCombatIcon           = 414  // TIL_19E
	tileFrameTopLeft         = 432  // TIL_1B0
	tileFrameTop             = 433
	tileFrameTopRight        = 434
	tileFrameBottomLeft      = 435
	tileFrameBottom          = 436
	tileFrameBottomRight     = 437
	tileFrameLeft            = 438
	tileFrameRight           = 439
	tileGargoyleLensRight    = 440 // TIL_1B8
	tileGargoyleLensLeft     = 441
	tileBritanniaLensRight   = 442
	tileBritanniaLensLeft    = 443
	tileDarkness             = 444 // TIL_1BC, the next three are dimmer shades
	firstObjectTile          = 512
)

// silhouettes: a tile value with these bits draws the tile's black pixels in a colour
const (
	silhouetteInvisible = 2 << typeBits
	silhouetteProtected = 4 << typeBits
	silhouetteCursed    = 6 << typeBits
)

// Area flags (AreaFlags[]), computed for each tile around the view.
const (
	areaLightMask  = 3 // the light an object there gives
	areaBackground = 4 // an object there lets one see through the map tile
	areaWall       = 8
	areaWindow     = 16
	areaOpaque     = 32
	areaQueued     = 64
	areaVisible    = 128
)

// Spells in SpellFx.
const (
	spellLight       = 0
	spellInfravision = 1
	spellStormCloak  = 13
	spellTimeStop    = 14
	spellEclipse     = 15
	spellHaste       = 10
	spellSlow        = 7
)

// view is the state of the view (seg_1100, the drawing parts of seg_1184 and seg_0A33).
type view struct {
	tileView   [viewTiles][viewTiles]int
	objectView [viewTiles][viewTiles]byte
	// objChain links the tiles drawn on a place (D_E5E0; slot 0 is the free list), the
	// tile of each slot is objChainTile (D_D5DC)
	objChain     [npcCount]byte
	objChainTile [npcCount]int
	areaFlags    [areaSize * areaSize]byte
	areaLight    [areaSize * areaSize]byte
	// the queue of places whose light is computed (D_BBCC, D_BCCC, visited_w, visited_r)
	queueX, queueY [npcCount]byte
	queueIn        byte
	queueOut       byte
	// xray shows everything (D_05E8)
	xray bool
	// frameCounter drives the tile animations (D_0338)
	frameCounter int
	// animationTick lets characters fidget once per prompt frame (D_033E)
	animationTick bool
	// worldShown is D_0340: the view is drawn while the world is on screen.
	worldShown bool
	// screenFade makes the next drawMap fade the view in (ScreenFade)
	screenFade bool
	// cyclopsShake is how hard a walking cyclops shakes the ground (CyclopsFlag)
	cyclopsShake int
	// erupting is 1 while a volcano may erupt, then the volcano (Erupting)
	erupting int
	// paletteCounter animates the palette (D_01D2)
	paletteCounter byte
	// basePalette is U6PAL (D_8C2C)
	basePalette *[768]byte
}

// isVisible says whether one sees through area place areaOffset; (x, y) is the place.
// C: seg_1100:000B (__IsVisible)
func (g *Game) isVisible(at, x, y int) bool {
	tile := int(g.AreaTiles[at])
	flags := g.areaFlags[at]
	if flags&areaOpaque != 0 {
		return false
	}
	if g.isWindow(tile) || flags&areaWindow != 0 {
		return g.ViewX+viewCenter == x && abs(g.ViewY+viewCenter-y) < 2 ||
			g.ViewY+viewCenter == y && abs(g.ViewX+viewCenter-x) < 2
	}
	if flags&areaBackground == 0 && g.isOpaque(tile) {
		return false
	}
	return true
}

// queue adds a place to the queue of computeLight.
// C: seg_1100:00CF (__SetVisited)
func (g *Game) queue(x, y int) {
	if x >= 0 && x < areaSize && y >= 0 && y < areaSize {
		g.queueX[g.queueIn] = byte(x)
		g.queueY[g.queueIn] = byte(y)
		g.queueIn++
		if g.queueIn == g.queueOut {
			g.queueIn--
		} else {
			g.areaFlags[y*areaSize+x] |= areaQueued
		}
	}
}

// lightFalloff is how far the light of (0, 0) is from (dx, dy) (D_05EA).
var lightFalloff = [8][8]int{
	{0, 1, 2, 3, 4, 5, 6, 7},
	{1, 1, 2, 3, 4, 5, 6, 7},
	{2, 2, 3, 4, 5, 6, 6, 7},
	{3, 3, 4, 4, 5, 6, 7, 8},
	{4, 4, 5, 5, 6, 7, 7, 8},
	{5, 5, 6, 6, 7, 7, 8, 9},
	{6, 6, 6, 7, 7, 8, 8, 9},
	{7, 7, 7, 8, 8, 9, 9, 10},
}

// the walk around a place: north, then clockwise (D_062C, D_0634)
var (
	aroundX = [8]int{0, 1, 0, 0, -1, -1, 0, 0}
	aroundY = [8]int{-1, 0, 1, 1, 0, 0, -1, -1}
)

// light modes of computeLight
const (
	lightSource = 0 // a lamp: spread as far as its light
	lightView   = 1 // the party's eyes: spread over the view and mark what is seen
	lightPeer   = 2 // a gem: spread everywhere
)

const fullLight = 4

// computeLight spreads light (and, in lightView mode, sight) from area place (x, y).
// C: seg_1100:0131 (C_1100_0131)
func (g *Game) computeLight(x, y, dist, mode int) {
	at := y*areaSize + x
	if g.isVisible(at, x, y) {
		g.queue(x, y)
	} else {
		if g.isVisible(at+1, x+1, y) {
			g.queue(x+1, y)
		}
		if g.isVisible(at+areaSize, x, y+1) {
			g.queue(x, y+1)
		}
	}
	for g.queueIn != g.queueOut {
		qx, qy := int(g.queueX[g.queueOut]), int(g.queueY[g.queueOut])
		g.queueOut++
		at := qy*areaSize + qx
		light := fullLight - lightFalloff[abs(qx-x)][abs(qy-y)] + dist
		light = max(0, min(light, fullLight))
		g.areaLight[at] += byte(light)
		if mode != lightSource {
			g.areaFlags[at] |= areaVisible
		}
		if !g.isVisible(at, qx, qy) {
			continue
		}
		for dir := range len(aroundX) {
			qx += aroundX[dir]
			qy += aroundY[dir]
			at = qy*areaSize + qx
			// fix: the original reads outside the area for places on its edge
			if at < 0 || at >= len(g.areaFlags) {
				continue
			}
			if g.areaFlags[at]&areaQueued != 0 {
				continue
			}
			switch {
			case mode == lightView:
				if g.ViewX-1 > qx || g.ViewX+viewTiles < qx || g.ViewY-1 > qy || g.ViewY+viewTiles < qy {
					continue
				}
				g.queue(qx, qy)
			case mode == lightPeer:
				g.queue(qx, qy)
			case abs(qx-x) < len(lightFalloff) && abs(qy-y) < len(lightFalloff) &&
				lightFalloff[abs(qx-x)][abs(qy-y)] <= dist+3:
				g.queue(qx, qy)
			}
		}
	}
}

// wallPatches pick the piece of wall shown for the sides of it one sees (D_0644).
var wallPatches = [16]int{0, 0, 1, 10, 0, 0, 2, 2, 1, 5, 1, 1, 11, 0, 2, 2}

// the sides of a wall (high nibble of TerrainType) and of the places seen around it
const (
	sideWest  = 1
	sideSouth = 2
	sideEast  = 4
	sideNorth = 8
	allSides  = 15
)

// computeView works out what the party sees: the light and sight over the area, the
// tiles of the view (hidden, dark, or a wall cut to the sides seen) and the objects.
// C: seg_1100:0306 (C_1100_0306)
func (g *Game) computeView() {
	if g.xray {
		for y := range viewTiles {
			for x := range viewTiles {
				g.tileView[y][x] = int(g.AreaTiles[(y+g.ViewY)*areaSize+x+g.ViewX])
			}
		}
		for y := chunkTiles; y < areaSize-chunkTiles; y++ {
			for x := chunkTiles; x < areaSize-chunkTiles; x++ {
				g.areaLight[y*areaSize+x] = fullLight
			}
		}
		g.ShowObjects()
		return
	}
	const first, last = chunkTiles, areaSize - chunkTiles // the 24x24 places that are computed
	for y := first; y < last; y++ {
		for x := first; x < last; x++ {
			g.areaFlags[y*areaSize+x] = 0
			g.areaLight[y*areaSize+x] = 0
		}
	}
	cornerX := (int(g.MapX) - areaMargin) &^ (chunkTiles - 1)
	cornerY := (int(g.MapY) - areaMargin) &^ (chunkTiles - 1)
	for n := g.SearchArea(cornerX+first, cornerY+first, cornerX+last-1, cornerY+last-1); n >= 0; n = g.NextArea() {
		o := g.object(n)
		x, y := o.Pos.X-cornerX, o.Pos.Y-cornerY
		at := y*areaSize + x
		// fix: objects across the map's edge fall outside the area
		if x < 0 || y < 0 || x >= areaSize || y >= areaSize {
			continue
		}
		flags := &g.areaFlags[at]
		tile := g.tileFrame(n)
		switch {
		case g.isWindow(tile):
			*flags |= areaWindow
		case g.isOpaque(tile):
			*flags |= areaOpaque
		case g.isBackground(tile):
			*flags |= areaBackground
		}
		double := func(bit byte) {
			if g.TileFlag[tile]&tileDoubleHeight != 0 && y != 0 {
				g.areaFlags[at-areaSize] |= bit
			}
			if g.TileFlag[tile]&tileDoubleWidth != 0 && x != 0 {
				g.areaFlags[at-1] |= bit
			}
		}
		if g.isOpaque(tile - 1) {
			double(areaOpaque)
		}
		if g.isWall(tile) {
			*flags |= areaWall
			double(areaWall)
		}
		if light := g.tileLight(tile); light != 0 && int(*flags&areaLightMask) < light {
			*flags = *flags&^areaLightMask | byte(light)
		}
	}

	g.computeLight(g.ViewX+viewCenter, g.ViewY+viewCenter, int(g.AmbientLight), lightView)

	const daylight = 7
	if g.AmbientLight < daylight {
		for y := first; y < last; y++ {
			for x := first; x < last; x++ {
				at := y*areaSize + x
				flags := g.areaFlags[at]
				if flags&areaLightMask == 0 || flags&areaVisible == 0 {
					continue
				}
				// the original tests the place below twice
				if g.isWall(int(g.AreaTiles[at])) && g.areaFlags[at+areaSize]&areaVisible == 0 {
					continue
				}
				const lampReach = 3
				for lx := x - lampReach; lx <= x+lampReach; lx++ {
					for ly := y - lampReach; ly <= y+lampReach; ly++ {
						g.areaFlags[ly*areaSize+lx] &^= areaQueued
					}
				}
				g.computeLight(x, y, int(flags&areaLightMask)-fullLight, lightSource)
			}
		}
	}

	for vy := range viewTiles {
		for vx := range viewTiles {
			at := (vy+g.ViewY)*areaSize + vx + g.ViewX
			var tile int
			switch {
			case g.areaFlags[at]&areaVisible == 0:
				tile = tileHidden
			case g.areaLight[at] == 0:
				tile = tileDarkness
			default:
				tile = g.wallSeen(at)
			}
			g.tileView[vy][vx] = tile
		}
	}
	g.ShowObjects()
}

// wallSeen returns the tile at area place at, for a wall the piece with only the sides
// one sees ("hide inside wall").
// C: seg_1100:0306 (C_1100_0306, the wall patch)
func (g *Game) wallSeen(at int) int {
	tile := int(g.AreaTiles[at])
	if int(g.TerrainType[tile])&terrainLowMask != terrainWall|terrainImpassable {
		return tile
	}
	const rowMask = 240 // 0xf0
	row := max(tile&rowMask, tileWallPatch)
	seen := 0
	if g.areaFlags[at-areaSize]&areaVisible != 0 {
		seen |= sideNorth
	}
	if g.areaFlags[at+1]&areaVisible != 0 {
		seen |= sideEast
	}
	if g.areaFlags[at+areaSize]&areaVisible != 0 {
		seen |= sideSouth
	}
	if g.areaFlags[at-1]&areaVisible != 0 {
		seen |= sideWest
	}
	if seen == allSides {
		return tile
	}
	sides := int(g.TerrainType[tile]) >> terrainCostShift
	if sides&sideEast != 0 {
		above := int(g.AreaTiles[at-areaSize])
		if g.areaFlags[at-areaSize]&areaWall == 0 &&
			(g.TerrainType[above]&terrainWall == 0 || g.TerrainType[above]&terrainSouth == 0) {
			sides &^= sideNorth
		}
	}
	if sides&sideSouth != 0 {
		left := int(g.AreaTiles[at-1])
		if g.areaFlags[at-1]&areaWall == 0 &&
			(g.TerrainType[left]&terrainWall == 0 || g.TerrainType[left]&terrainEast == 0) {
			sides &^= sideWest
		}
	}
	if sides == sideEast|sideSouth|sideWest || sides == sideNorth|sideSouth|sideWest ||
		sides > sideNorth|sideEast || sides == seen {
		sides &= seen
		const rowShift = 3
		if sides == sideSouth|sideWest || sides == sideNorth|sideEast {
			return tileWallCorner + row>>rowShift - tileWallPatch>>rowShift + wallPatches[sides]
		}
		return row + wallPatches[sides]
	}
	return tile
}

// showObject puts a tile on view place (x, y): a background tile replaces the map tile,
// others are chained, foreground tiles last. mode 1 puts it after all others (the parts
// of big objects), 2 treats it as foreground, 3 is for the shades of darkness.
// C: seg_1184:347C (ShowObject)
func (g *Game) showObject(tileFrame, x, y, mode int) {
	// fix: the lenses of the temple of singularity put tiles outside the view
	if x < 0 || y < 0 || x >= viewTiles || y >= viewTiles {
		return
	}
	tile := tileFrame & tileNumberMask
	if int(g.BaseTile[ObjEgg]) == tile {
		return
	}
	under := g.tileView[y][x]
	if under == tileHidden {
		return
	}
	if under == tileDarkness && (g.SpellFx[spellInfravision] == 0 || !g.isWarm(tile)) {
		return
	}
	if g.isBackground(tile) {
		g.tileView[y][x] = tileFrame
		return
	}
	slot := g.objChain[0]
	if slot == 0 {
		return
	}
	g.objChain[0] = g.objChain[slot]
	g.objChainTile[slot] = tileFrame
	link := &g.objectView[y][x]
	if g.isForeground(tile) || mode == 2 {
		switch {
		case mode&1 != 0 && g.SpellFx[spellInfravision] != 0 && mode == 3:
			at := link
			for ; *link != 0; link = &g.objChain[*link] {
				if !g.isWarm(g.objChainTile[*link] & tileNumberMask) {
					at = &g.objChain[*link]
					break
				}
			}
			link = at
		case mode&1 != 0:
			for *link != 0 {
				link = &g.objChain[*link]
			}
		default:
			for *link != 0 && !g.isForeground(g.objChainTile[*link]&tileNumberMask) {
				link = &g.objChain[*link]
			}
		}
	}
	g.objChain[slot] = *link
	*link = slot
}

// showBigObject shows an object and the other parts of a double width or height one.
// C: seg_1184:35EA (C_1184_35EA)
func (g *Game) showBigObject(tile, frame, x, y int) {
	if x < viewTiles && y < viewTiles {
		g.showObject(tile+frame, x, y, 0)
	}
	flags := g.TileFlag[tile&tileNumberMask]
	switch {
	case flags&tileDoubleWidth != 0:
		if x != 0 && y < viewTiles {
			g.showObject(tile+frame-1, x-1, y, 1)
		}
		if flags&tileDoubleHeight != 0 {
			if x < viewTiles && y != 0 {
				g.showObject(tile+frame-2, x, y-1, 1)
			}
			if x != 0 && y != 0 {
				g.showObject(tile+frame-3, x-1, y-1, 1)
			}
		}
	case x < viewTiles && y != 0 && flags&tileDoubleHeight != 0:
		g.showObject(tile+frame-1, x, y-1, 1)
	}
}

// the lenses of the temple of singularity
const (
	britanniaLensX = 926 // 0x39e
	gargoyleLensX  = 930 // 0x3a2
	lensY          = 856 // 0x358
)

// ShowObjects chains the objects in view on the view places.
// C: seg_1184:36C2 (ShowObjects)
func (g *Game) ShowObjects() {
	// TODO(port): MUS_051E
	hasMusic := false
	for i := range g.objChain {
		g.objChain[i] = byte(i + 1)
	}
	g.objectView = [viewTiles][viewTiles]byte{}
	mx, my := int(g.MapX), int(g.MapY)
	for n := g.SearchArea(mx-viewCenter, my-viewCenter, mx+viewCenter+1, my+viewCenter+1); n >= 0; n = g.NextArea() {
		o := g.object(n)
		silhouette := 0
		t := o.Type()
		if g.erupting == 1 && (t == ObjVolcano || t == ObjFumarole) {
			g.erupting = n
		}
		if t == ObjMusician2 {
			hasMusic = true
		}
		// TODO(port): MUS_043B (the sounds of what is around)
		if o.Has(StatusInvisible) && (n >= npcCount || !g.isPlayerControlled(n)) {
			continue
		}
		if n < npcCount && g.NPCs[n].Flags&flagDraggedUnder != 0 {
			continue
		}
		if t == ObjMirror && o.Frame() < 2 {
			if g.livingAt(0, o.Pos.X, o.Pos.Y+1) >= 0 {
				o.SetFrame(1)
			} else {
				o.SetFrame(0)
			}
		}
		tile := g.tileFrame(n)
		if o.Has(StatusInvisible) && t < ObjGiantScorpion {
			silhouette = silhouetteInvisible
		}
		if n < npcCount {
			if g.NPCs[n].Has(Protected) {
				silhouette = silhouetteProtected
			}
			if o.Has(StatusCursed) {
				silhouette = silhouetteCursed
			}
		}
		x, y := o.Pos.X-mx+viewCenter, o.Pos.Y-my+viewCenter
		if silhouette != 0 {
			g.showBigObject(tile, silhouette, x, y)
		}
		if !o.Has(StatusInvisible) {
			g.showBigObject(tile, 0, x, y)
		}
		if o.Pos.Z == 0 && y < viewTiles && x+mx == britanniaLensX && y+my == lensY && t == ObjBritanniaLens {
			if x != 0 {
				g.showObject(tileBritanniaLensRight, x+1, y, 1)
			}
			if x < viewTiles-1 {
				g.showObject(tileBritanniaLensLeft, x-1, y, 1)
			}
			continue
		}
		if o.Pos.Z == 0 && y < viewTiles && x+mx == gargoyleLensX && y+my == lensY && t == ObjGargoyleLens {
			if x != 0 {
				g.showObject(tileGargoyleLensRight, x+1, y, 1)
			}
			if x < viewTiles-1 {
				g.showObject(tileGargoyleLensLeft, x-1, y, 1)
			}
		}
	}
	if !g.xray {
		for i := range viewTiles {
			for j := range viewTiles {
				// fix: masked; a background object's silhouette bits made the original
				// read past TerrainType
				if g.isWall(g.tileView[j][i] & tileNumberMask) {
					right := (j+g.ViewY)*areaSize + i + 1 + g.ViewX
					below := (j+1+g.ViewY)*areaSize + i + g.ViewX
					if g.areaFlags[right]&areaVisible == 0 || g.areaFlags[below]&areaVisible == 0 {
						if first := g.objectView[j][i]; first != 0 {
							g.objectView[j][i] = 0
							last := first
							for g.objChain[last] != 0 {
								last = g.objChain[last]
							}
							g.objChain[last] = g.objChain[0]
							g.objChain[0] = first
						}
					}
				}
				light := g.areaLight[(j+g.ViewY)*areaSize+i+g.ViewX]
				if light < fullLight && light > 0 {
					g.showObject(tileDarkness+int(light), i, j, 3)
				}
			}
		}
	}
	if g.erupting == 1 {
		g.erupting = 0
	}
	// TODO(port): ShowObjects starting a bard's song (MUS_SongIndex, MUS_Song)
	g.musicianInView = hasMusic
}

// the hands of the clock (D_034B): the direction of each hour
var hourDirections = [12]int{0, 1, 1, 2, 3, 3, 4, 5, 5, 6, 7, 7}

// drawClockFace draws the hands of a clock on the tile at (x, y) of the back buffer.
// C: seg_0A33:0931 (C_0A33_0931)
func (g *Game) drawClockFace(x, y int) {
	const centreX, centreY = 4, 14
	x += centreX
	y += centreY
	g.gr.SetColor(g.colours.clockHands)
	g.gr.Offscreen = true
	g.gr.PutPixel(x, y)
	d := hourDirections[int(g.Hour)%len(hourDirections)]
	g.gr.PutPixel(dirStepX[d]+x, dirStepY[d]+y)
	const minutesPerDirection = 15 // two directions per quarter hour
	d = int(g.Minute) * 2 / minutesPerDirection
	g.gr.PutPixel(dirStepX[d]+x, dirStepY[d]+y)
	g.gr.Offscreen = false
}

// the part of the back buffer shown, inside the frame
const (
	viewFirstPixel = borderSize
	viewLastPixel  = viewTiles*tileSize - borderSize - 1
)

// drawMap draws the view on the back buffer and, when onScreen, the aim cursor and the
// frame, then copies it to the screen.
// C: seg_0A33:09CE (C_0A33_09CE)
func (g *Game) drawMap(onScreen bool) {
	g.gr.SetColor(g.colours.viewBorder)
	for j := range viewTiles {
		for i := range viewTiles {
			x, y := i*tileSize, j*tileSize
			g.gr.DrawTileOffscreen(g.tileView[j][i], x, y)
			for k := g.objectView[j][i]; k != 0; k = g.objChain[k] {
				tile := g.objChainTile[k]
				g.gr.DrawTileOffscreen(tile, x, y)
				if int(g.BaseTile[ObjClock]) == tile&tileNumberMask {
					g.drawClockFace(x, y)
				}
			}
		}
	}
	if g.SpellFx[spellStormCloak] != 0 {
		const sparkles, sparkleColour = 100, 8
		g.gr.XorMode, g.gr.Offscreen = true, true
		g.gr.SetColor(sparkleColour)
		for range sparkles {
			x := g.rand(viewFirstPixel, viewLastPixel)
			g.gr.PutPixel(x, g.rand(viewFirstPixel, viewLastPixel))
		}
		g.gr.XorMode, g.gr.Offscreen = false, false
	}
	if !onScreen {
		return
	}
	if g.MouseMode == 0 && !g.selectByTab {
		g.drawStatusCursor(0)
		g.SelectMode = 0
	}
	if g.SelectMode == 1 {
		tile := tileAimTarget
		if g.SelectRange < 0 {
			tile = tileAimDirection
		}
		g.gr.DrawTileOffscreen(tile, g.AimX*tileSize, g.AimY*tileSize)
	}
	const far = (viewTiles - 1) * tileSize
	g.gr.DrawTileOffscreen(tileFrameTopLeft, 0, 0)
	g.gr.DrawTileOffscreen(tileFrameTopRight, far, 0)
	g.gr.DrawTileOffscreen(tileFrameBottomLeft, 0, far)
	g.gr.DrawTileOffscreen(tileFrameBottomRight, far, far)
	for i := 1; i < viewTiles-1; i++ {
		g.gr.DrawTileOffscreen(tileFrameTop, i*tileSize, 0)
		g.gr.DrawTileOffscreen(tileFrameBottom, i*tileSize, far)
		g.gr.DrawTileOffscreen(tileFrameLeft, 0, i*tileSize)
		g.gr.DrawTileOffscreen(tileFrameRight, far, i*tileSize)
	}
	// the pointer is drawn by the host over the screen; the original drew it on the back
	// buffer here
	if g.screenFade {
		g.gr.FadeIn(viewFirstPixel, viewFirstPixel, viewLastPixel, viewLastPixel)
		g.screenFade = false
	} else {
		g.gr.CopyBackBufferToScreen(viewFirstPixel, viewFirstPixel, viewLastPixel, viewLastPixel, viewFirstPixel, viewFirstPixel)
	}
}

// OtherAnimations brings the view up to date and, while the world is shown, animates and
// draws it.
// C: seg_0A33:0CBC (OtherAnimations)
func (g *Game) OtherAnimations() {
	if g.viewChanged {
		g.computeView()
		g.viewChanged = false
	}
	if g.worldShown {
		g.animateTiles()
		g.drawMap(true)
		// TODO(port): MUS_0525
	}
	if g.cyclopsShake != 0 {
		strength := g.cyclopsShake
		g.cyclopsShake = 0
		g.shakeScreen(strength, 1)
	}
}

// updateViewNow computes and draws the view when it changed.
// C: seg_0A33:0D06 (C_0A33_0D06)
func (g *Game) updateViewNow() {
	if g.viewChanged {
		g.computeView()
		g.viewChanged = false
		g.OtherAnimations()
	}
}

// shakeScreen shakes the view.
// TODO(port): ShakeScreen
func (g *Game) shakeScreen(duration, amplitude int) {}

// Palette animation: the fire and water colours cycle, colour 255 pulses.
const (
	fireFirst   = 224 // 0xe0, two runs of eight colours
	cycleEight  = 8
	waterFirst  = 240 // 0xf0, three runs of four colours
	cycleFour   = 4
	pulseColour = 255
	pulseBase   = 12
	pulseSteps  = 16
)

// paletteAnimation cycles the animated colours of the DAC.
// C: seg_0903:0820 (PaletteAnimation)
func (g *Game) paletteAnimation() {
	if g.SpellFx[spellTimeStop] != 0 || g.basePalette == nil {
		return
	}
	level := int(g.paletteCounter) & (pulseSteps - 1)
	g.paletteCounter++
	if level > pulseSteps/2-1 {
		level = pulseSteps - 1 - level
	}
	level += pulseBase
	dac := &g.gr.Palette
	for c := range 3 {
		dac[pulseColour*3+c] = byte(level)
	}
	set := func(to, from int) { copy(dac[to*3:to*3+3], g.basePalette[from*3:from*3+3]) }
	counter := int(g.paletteCounter)
	for k := range cycleEight {
		set((counter-k)&(cycleEight-1)+fireFirst, fireFirst+cycleEight-1-k)
		set((counter-k)&(cycleEight-1)+fireFirst+cycleEight, fireFirst+2*cycleEight-1-k)
	}
	for k := range cycleFour {
		for run := range 3 {
			first := waterFirst + run*cycleFour
			set((counter>>1-k)&(cycleFour-1)+first, first+cycleFour-1-k)
		}
	}
}
