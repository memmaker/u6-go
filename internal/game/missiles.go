package game

// Flying missiles and what they hit (Missiles in seg_2FC1, parts of seg_2337).

import "github.com/memmaker/u6-go/internal/sound"

// missileAngles are the angles (128 per turn) of a sprite flying from the centre to each
// place 7 tiles around, by y then x (D_2A5C).
var missileAngles = [15 + 1][15 + 1]byte{
	{48, 46, 45, 43, 40, 38, 35, 32, 29, 26, 24, 21, 19, 18, 16, 15},
	{50, 48, 46, 44, 41, 38, 35, 32, 29, 26, 23, 20, 18, 16, 14, 13},
	{52, 50, 48, 46, 43, 39, 36, 32, 28, 25, 21, 18, 16, 14, 13, 11},
	{53, 52, 50, 48, 45, 41, 37, 32, 27, 23, 19, 16, 14, 12, 11, 9},
	{56, 55, 53, 51, 48, 44, 38, 32, 26, 20, 16, 13, 11, 9, 8, 7},
	{58, 58, 57, 55, 52, 48, 41, 32, 23, 16, 12, 9, 7, 6, 6, 5},
	{61, 61, 60, 59, 58, 55, 48, 32, 16, 9, 6, 5, 4, 3, 3, 3},
	{64, 64, 64, 64, 64, 64, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{67, 67, 68, 69, 70, 73, 80, 96, 112, 119, 122, 123, 124, 125, 125, 125},
	{70, 70, 71, 73, 76, 80, 87, 96, 105, 112, 116, 119, 121, 122, 122, 123},
	{72, 73, 75, 77, 80, 84, 90, 96, 102, 108, 112, 115, 117, 119, 120, 121},
	{75, 76, 78, 80, 83, 87, 91, 96, 101, 105, 109, 112, 114, 116, 117, 119},
	{77, 78, 80, 82, 85, 89, 92, 96, 100, 103, 107, 110, 112, 114, 115, 117},
	{78, 80, 82, 84, 87, 90, 93, 96, 99, 102, 105, 108, 110, 112, 114, 115},
	{80, 82, 83, 85, 88, 90, 93, 96, 99, 102, 104, 107, 109, 110, 112, 113},
	{81, 83, 85, 87, 89, 91, 93, 96, 99, 101, 103, 105, 107, 109, 111, 112},
}

// missileKinds describe the missiles by tile (D_2B5C): the angle the tile is drawn at, how
// fast it spins, its pivot and how often it is redrawn (D_2B80, D_2B92, D_2BA4, D_2BB6,
// D_2BC8). The first kind is any other tile.
var missileKinds = [...]struct {
	tile, angle, spin, pivotX, pivotY, refresh int
}{
	{0, 64, 0, 127, 127, 6},
	{547, 16, 0, 117, 127, 3},  // TIL_223 dagger
	{548, 0, 10, 127, 127, 2},  // TIL_224
	{549, 16, 10, 134, 139, 2}, // TIL_225
	{560, 0, 10, 122, 127, 4},  // TIL_230
	{566, 0, 0, 113, 70, 4},    // TIL_236
	{567, 0, 0, 127, 61, 4},    // TIL_237
	{392, 32, 0, 127, 127, 2},  // TIL_188
	{393, 32, 0, 127, 127, 2},  // TIL_189
	{565, 0, 0, 127, 127, 3},   // TIL_235
	{394, 32, 0, 127, 127, 2},  // TIL_18A
	{395, 32, 0, 127, 127, 2},  // TIL_18B
	{396, 32, 0, 127, 127, 2},  // TIL_18C
	{397, 32, 0, 127, 127, 2},  // TIL_18D
	{380, 0, 10, 127, 127, 2},  // TIL_17C
	{381, 0, 10, 127, 127, 2},  // TIL_17D
	{382, 0, 10, 127, 127, 2},  // TIL_17E
	{383, 0, 10, 127, 127, 2},  // TIL_17F
}

// Missile options (the attr of Missiles).
const (
	missileReverse  = 1 // fly from the destination to the source
	missileTrail    = 2 // leave the sprites drawn (explosions)
	missileAltSound = 4
	missileNoSpin   = 8
)

const (
	maxMissiles      = 16
	saveSlots        = 8 // sprite slots 0-7 keep what is under the sprites
	fixedSpriteSlot  = 15
	missileReach     = 7 // the tiles of rangeTable
	viewPixelEnd     = 168
	viewPixelStart   = 8
	maxHits          = 62 // the size of hitList less its end mark
	halfTile         = 8
	trailRefresh     = 6
	paletteEveryStep = 3
)

// missileState is what the last flight left: where the first missile stopped (D_EBFE),
// what it hit (D_EC9E), every character in its way (D_EC1E) and who threw it (D_196A).
type missileState struct {
	missileX, missileY int
	missileHit         int
	hitList            []int
	thrower            int
}

// clipRect is the part of the view to copy to the screen (D_ECBC).
type clipRect struct{ x0, x1, y0, y1 int }

// grow adds the 16x16 square around (x, y), kept inside the view.
// C: seg_2FC1:017F (C_2FC1_017F)
func (r *clipRect) grow(x, y int) {
	const last = viewPixelEnd - 1
	if x < tileSize {
		r.x0 = viewPixelStart
	} else if x-halfTile < r.x0 {
		r.x0 = x - halfTile
	}
	if y < tileSize {
		r.y0 = viewPixelStart
	} else if y-halfTile < r.y0 {
		r.y0 = y - halfTile
	}
	if x > last-halfTile {
		r.x1 = last
	} else if x+halfTile > r.x1 {
		r.x1 = x + halfTile
	}
	if y > last-halfTile {
		r.y1 = last
	} else if y+halfTile > r.y1 {
		r.y1 = y + halfTile
	}
}

// redrawFrame draws the frame of the view on the back buffer where the rectangle touches
// it.
// C: seg_2FC1:0212 (C_2FC1_0212)
func (g *Game) redrawFrame(x0, y0, x1, y1 int) {
	const far = (viewTiles - 1) * tileSize
	if x0 < tileSize && y0 < tileSize {
		g.gr.DrawTileOffscreen(tileFrameTopLeft, 0, 0)
	}
	if x1 >= far && y0 < tileSize {
		g.gr.DrawTileOffscreen(tileFrameTopRight, far, 0)
	}
	if x0 < tileSize && y1 >= far {
		g.gr.DrawTileOffscreen(tileFrameBottomLeft, 0, far)
	}
	if x1 >= far && y1 >= far {
		g.gr.DrawTileOffscreen(tileFrameBottomRight, far, far)
	}
	for i := 1; i < viewTiles-1; i++ {
		if y0 < tileSize {
			g.gr.DrawTileOffscreen(tileFrameTop, i*tileSize, 0)
		}
		if y1 >= far {
			g.gr.DrawTileOffscreen(tileFrameBottom, i*tileSize, far)
		}
		if x0 < tileSize {
			g.gr.DrawTileOffscreen(tileFrameLeft, 0, i*tileSize)
		}
		if x1 >= far {
			g.gr.DrawTileOffscreen(tileFrameRight, far, i*tileSize)
		}
	}
}

// redrawForeground draws again the foreground tiles and the darkness over the sprites in
// the rectangle.
// C: seg_2FC1:032B (C_2FC1_032B)
func (g *Game) redrawForeground(x0, y0, x1, y1 int) {
	for i := x0 / tileSize; i <= x1/tileSize && i < viewTiles; i++ {
		for j := y0 / tileSize; j <= y1/tileSize && j < viewTiles; j++ {
			for k := g.objectView[j][i]; k != 0; k = g.objChain[k] {
				if tile := g.objChainTile[k]; tile != 0 && g.isForeground(tile&tileNumberMask) {
					g.gr.DrawTileOffscreen(tile&tileNumberMask, i*tileSize, j*tileSize)
				}
			}
			if t := g.tileView[j][i]; t == tileHidden || t == tileDarkness {
				g.gr.DrawTileOffscreen(tileHidden, i*tileSize, j*tileSize)
			}
		}
	}
	g.redrawFrame(x0, y0, x1, y1)
}

func (t *tables) isNoShoot(tile int) bool { return t.TileFlag[tile]&tileNoShoot != 0 }

// missiles flies count missiles of tile from (srcX, srcY) to each destination, stopped by
// what blocks them or by maxRange. Where a missile stopped short its destination is
// changed; it returns false then. Every character crossed goes to hitList.
// C: seg_2FC1:042D (Missiles)
func (g *Game) missiles(srcX, srcY int, dstX, dstY []int, maxRange, tile, attr int) bool {
	count := len(dstX)
	g.drawMap(false)
	var hit [npcCount]bool
	ok := true
	var (
		curX, curY, deltaX, deltaY, tileX, tileY, rot, sprX, sprY, berr [maxMissiles]int
		incX, incY, berrInc                                             [maxMissiles][2]int
	)
	g.missileX, g.missileY = srcX, srcY
	tileX[0], tileY[0] = srcX, srcY
	if attr&missileReverse != 0 {
		tileX[0], dstX[0] = dstX[0], tileX[0]
		tileY[0], dstY[0] = dstY[0], tileY[0]
	}
	curX[0], curY[0] = tileX[0]*tileSize+halfTile, tileY[0]*tileSize+halfTile
	kind := 0
	for i, k := range missileKinds {
		if k.tile == tile {
			kind = i
			break
		}
	}
	mk := missileKinds[kind]
	refresh := mk.refresh
	if attr&missileTrail != 0 {
		refresh = trailRefresh
	}
	if tile != 0 {
		if attr&missileNoSpin != 0 {
			g.gr.MakeRotatedSprite(fixedSpriteSlot, tile, 0, missileKinds[0].pivotX, missileKinds[0].pivotY)
		} else {
			for d := range count {
				rot[d] = int(missileAngles[dstY[d]-tileY[0]+missileReach][dstX[d]-tileX[0]+missileReach]) - mk.angle
				slot := d + saveSlots
				if attr&missileTrail != 0 {
					slot = d
				}
				g.gr.MakeRotatedSprite(slot, tile, rot[d], mk.pivotX, mk.pivotY)
			}
		}
	}
	deltaMax := 0
	for d := range count {
		curX[d], curY[d] = curX[0], curY[0]
		tileX[d], tileY[d] = tileX[0], tileY[0]
		deltaX[d], deltaY[d] = (dstX[d]-tileX[0])*tileSize, (dstY[d]-tileY[0])*tileSize
		incY[d][0] = 0
		incX[d] = [2]int{1, 1}
		if deltaX[d] < 0 {
			deltaX[d] = -deltaX[d]
			incX[d] = [2]int{-1, -1}
		}
		incY[d][1] = 1
		if deltaY[d] < 0 {
			deltaY[d] = -deltaY[d]
			incY[d][1] = -1
		}
		if deltaX[d] < deltaY[d] {
			deltaX[d], deltaY[d] = deltaY[d], deltaX[d]
			incY[d][0] = incY[d][1]
			incX[d][1] = incX[d][0]
			incX[d][0] = 0
		}
		deltaMax = max(deltaMax, deltaX[d])
		berr[d] = 2*deltaY[d] - deltaX[d]
		berrInc[d] = [2]int{2 * deltaY[d], (deltaY[d] - deltaX[d]) << 1}
	}
	for i := 0; i < deltaMax; i++ {
		shown := false
		r := clipRect{viewPixelEnd, viewPixelStart - 1, viewPixelEnd, viewPixelStart - 1}
		for d := 0; d < count; d++ {
			if deltaX[d] <= i {
				if i != 0 {
					r.grow(sprX[d], sprY[d])
				}
				continue
			}
			dir := 0
			if berr[d] >= 0 {
				dir = 1
			}
			berr[d] += berrInc[d][dir]
			curX[d] += incX[d][dir]
			curY[d] += incY[d][dir]
			if tile != 0 && refresh != 0 && i%refresh == 0 {
				nx := curX[d] - (int(g.MapX)-viewCenter)*tileSize
				ny := curY[d] - (int(g.MapY)-viewCenter)*tileSize
				if nx >= viewPixelStart && nx < viewPixelEnd && ny >= viewPixelStart && ny < viewPixelEnd {
					if !shown {
						if d == 0 {
							switch {
							case attr&missileAltSound != 0:
								g.sfx(sound.EffectMissile, i)
							case attr&missileReverse != 0:
								g.sfx(sound.EffectProjectile, deltaMax-i)
							default:
								g.sfx(sound.EffectProjectile, i)
							}
						}
						shown = true
					}
					r.grow(nx, ny)
					if attr&missileTrail != 0 {
						g.gr.DrawSprite(d, nx, ny)
					} else {
						if i != 0 {
							r.grow(sprX[d], sprY[d])
						}
						sprX[d], sprY[d] = nx, ny
					}
				}
				if mk.spin != 0 {
					rot[d] += mk.spin
					if attr&(missileTrail|missileNoSpin) == 0 {
						g.gr.MakeRotatedSprite(d+saveSlots, tile, rot[d], mk.pivotX, mk.pivotY)
					}
				}
			}
			tx, ty := curX[d]/tileSize, curY[d]/tileSize
			if tx == tileX[d] && ty == tileY[d] {
				continue
			}
			offsetX, offsetY := abs(tx-srcX), abs(ty-srcY)
			blocked := true
			if offsetX <= missileReach && offsetY <= missileReach && rangeTable[offsetX][offsetY] <= maxRange {
				tileX[d], tileY[d] = tx, ty
				t := g.GetTileAtXYZ(tx, ty, int(g.MapZ))
				if offsetX <= 1 && offsetY <= 1 && g.isWindow(t) || !g.isNoShoot(t) {
					n := g.FindLoc(tx, ty, int(g.MapZ))
					for ; n >= 0; n = g.NextLoc() {
						o := g.object(n)
						if n < npcCount && (o.Type() != ObjCorpser || o.Frame() != 0) && !g.draggedUnder(n) {
							hit[n] = true
						}
						if g.isNoShoot(g.tileFrame(n) - g.locPart) {
							break
						}
					}
					blocked = n >= 0
				}
			}
			if blocked {
				deltaX[d] = i
				break
			}
			if d == 0 {
				g.missileX, g.missileY = tileX[0], tileY[0]
			}
		}
		if !shown {
			continue
		}
		if i&paletteEveryStep == 0 {
			g.paletteAnimation()
		}
		if attr&missileTrail == 0 {
			for d := range count {
				if deltaX[d] > i || attr&missileNoSpin != 0 {
					g.gr.SaveUnderSprite(d, sprX[d], sprY[d])
				}
			}
			for d := range count {
				if attr&missileNoSpin != 0 {
					g.gr.DrawSprite(fixedSpriteSlot, sprX[d], sprY[d])
				} else if deltaX[d] > i {
					g.gr.DrawSprite(d+saveSlots, sprX[d], sprY[d])
				}
			}
		}
		if r.x0 <= r.x1 && r.y0 <= r.y1 {
			g.redrawForeground(r.x0, r.y0, r.x1, r.y1)
			g.gr.CopyBackBufferToScreen(r.x0, r.y0, r.x1, r.y1, r.x0, r.y0)
		}
		if attr&missileTrail == 0 {
			for d := range count {
				if deltaX[d] > i || attr&missileNoSpin != 0 {
					g.gr.DrawSprite(d, sprX[d], sprY[d])
				}
			}
		}
	}
	g.sfx(sound.EffectSilence, 0)
	if tile != 0 && attr&(missileTrail|missileNoSpin) == 0 {
		g.OtherAnimations()
	}
	for d := range count {
		if tileX[d] == dstX[d] && (tileY[d] == dstY[d] || attr&missileReverse != 0) {
			continue
		}
		ok = false
		dstX[d], dstY[d] = tileX[d], tileY[d]
		if d == 0 && g.thrower >= 0 {
			g.missileHit = g.objectAtForSelection(g.thrower, tileX[0], tileY[0])
		}
	}
	g.hitList = g.hitList[:0]
	for n := range hit {
		if hit[n] && len(g.hitList) < maxHits {
			g.hitList = append(g.hitList, n)
		}
	}
	return ok
}

// throw flies a missile of tile from character n towards (*x, *y).
// C: seg_2337:0B3F (COMBAT_Missile)
func (g *Game) throw(n int, x, y *int, maxRange, tile int) bool {
	g.thrower = n
	deltaX, deltaY := []int{*x}, []int{*y}
	p := g.object(n).Pos
	ok := g.missiles(p.X, p.Y, deltaX, deltaY, maxRange, tile, 0)
	*x, *y = deltaX[0], deltaY[0]
	g.thrower = noObject
	return ok
}

// canSee says whether character a sees object b.
// C: seg_2337:0837 (COMBAT_canSee)
func (g *Game) canSee(a, b int) bool {
	ob := g.object(b)
	visible := !ob.Has(StatusInvisible)
	if g.isPlayerControlled(a) && !visible && b < npcCount && g.isPlayerControlled(b) {
		visible = true
	}
	if b < npcCount && g.draggedUnder(b) {
		visible = false
	}
	if ob.Shape == shapeOf(ObjCorpser, 0) {
		visible = false
	}
	if g.isIgnored(g.tileFrame(b)) {
		visible = false
	}
	if g.object(a).Pos.Z != ob.Pos.Z {
		visible = false
	}
	return visible
}

// objectAtForSelection returns the object at (x, y) a command of actor acts on: the
// first living character it sees, else the first object it sees, else the first object.
// C: seg_2337:08F1 (C_2337_08F1)
func (g *Game) objectAtForSelection(actor, x, y int) int {
	first := noObject
	n := g.FindLoc(x, y, int(g.MapZ))
	for ; n >= 0; n = g.NextLoc() {
		if g.canSee(actor, n) {
			break
		}
		if first < 0 {
			first = n
		}
	}
	found := n
	living := n
	for ; living >= 0; living = g.NextLoc() {
		if g.canSee(actor, living) {
			if living < npcCount && !g.NPCs[living].Has(Dead) {
				break
			}
			continue
		}
		if first < 0 {
			first = living
		}
	}
	if living >= 0 {
		return living
	}
	if first >= 0 && found < 0 {
		return first
	}
	return found
}
