package game

// The animation of the world: animated tiles, and the idle movements of the characters
// near the party (seg_0A33).

// frame bits of characters: the low two bits are the step, the rest the direction
const (
	stepMask      = 3
	directionBits = 252 // 0xfc
)

// fidget plays an idle frame from a three-step walk: from the middle step it rarely moves
// a foot, from a foot it goes back to the middle. It returns the new step and whether it
// changed.
func (g *Game) fidget(step int) (int, bool) {
	switch step {
	case 0, 2:
		if g.rand(0, 31) == 0 {
			return 1, true
		}
	case 1:
		if g.rand(0, 63) == 0 {
			return g.rand(0, 2), true
		}
	}
	return step, false
}

// animateTiles advances the animated tiles and lets the characters near the party
// fidget.
// C: seg_0A33:0073 (C_0A33_0073)
func (g *Game) animateTiles() {
	if g.SpellFx[spellTimeStop] != 0 {
		return
	}
	g.frameCounter++
	for i := range g.animCount {
		frame := int(g.animFrame[i])
		switch g.animState[i] {
		case animForward:
			frame += g.frameCounter & int(g.animMask[i]) >> g.animShift[i]
		case animBackward:
			frame += ^g.frameCounter & int(g.animMask[i]) >> g.animShift[i]
		}
		g.gr.SetTileFrame(int(g.animTile[i]), frame)
	}
	if g.frameCounter&1 == 0 {
		g.gr.AnimateCoasts()
	}
	const nearby = 6
	refresh := false
	for i := range npcCount {
		o := g.object(i)
		if o.Shape == 0 || o.Pos.Z != int(g.MapZ) || abs(o.Pos.X-int(g.MapX)) >= nearby || abs(o.Pos.Y-int(g.MapY)) >= nearby {
			continue
		}
		t := o.Type()
		// the original tests this inside the loop
		if !g.animationTick {
			continue
		}
		npc := &g.NPCs[i]
		if npc.Has(Dead | Asleep | Paralyzed) {
			continue
		}
		frame := o.Frame()
		base := frame & directionBits
		changed := false
		switch {
		case t >= ObjFighter && t <= ObjAvatar && t != ObjCat:
			switch t {
			case ObjFarmer2:
				if g.rand(0, 3) != 0 {
					if g.rand(0, 15) == 0 {
						o.Shape = shapeOf(ObjFarmer, 0)
						frame, base, changed = 0, 8, true
					}
				} else {
					frame ^= 1
					changed = true
				}
			case ObjMusician2:
				if g.rand(0, 1) != 0 {
					frame ^= 1
					changed = true
				}
			case ObjJester:
				frame &= stepMask
				if g.rand(0, 1) != 0 && frame < 3 {
					frame++
					if frame > 2 {
						frame = 0
					}
					changed = true
				}
			default:
				frame, changed = g.fidget(frame & stepMask)
				if t == ObjFarmer && npc.Mode == AIFarm && g.rand(0, 15) == 0 {
					o.Shape = shapeOf(ObjFarmer2, 0)
					frame, base, changed = 0, 0, true
				}
			}
		case t == ObjGargoyle:
			const steps = 3
			base = frame - frame%steps
			frame, changed = g.fidget(frame % steps)
		case t == ObjWingedGargoyle:
			const steps = 12
			base = frame - frame%steps
			frame %= steps
			switch frame {
			case 3, 11:
				if g.rand(0, 31) == 0 {
					frame, changed = 7, true
				}
			case 7:
				if g.rand(0, 63) == 0 {
					frame, changed = 11, true
					if g.rand(0, 1) != 0 {
						frame = 3
					}
				}
			}
			base += frame
			frame = 0
		case t >= ObjDaemon && t <= ObjMongbat:
			if g.rand(0, 7) == 0 {
				frame ^= 1
				changed = true
			}
		case t == ObjGiantBat || t == ObjAcidSlug:
			if g.rand(0, 1) != 0 {
				if npc.Flags&flagWalking != 0 {
					if frame++; frame >= 3 {
						frame = 3
						npc.Flags &^= flagWalking
					}
				} else if frame--; frame <= 0 {
					frame = 0
					npc.Flags |= flagWalking
				}
				changed = true
			}
		case t == ObjGazer || t == ObjReaper || t == ObjRotWorms:
			if g.rand(0, 1) != 0 {
				if frame++; frame > 3 {
					frame = 0
				}
				changed = true
			}
		case t == ObjInsects || t == ObjGhost || t == ObjGremlin:
			if t == ObjInsects || g.rand(0, 1) != 0 {
				frame, changed = g.rand(0, 3), true
			}
		case t == ObjWisp || t == ObjGiantSquid:
			if g.rand(0, 1) != 0 {
				if g.rand(0, 15) == 0 {
					frame = 3
				} else {
					frame = g.rand(0, 2)
				}
				changed = true
			}
		case t == ObjCorpser:
			if frame != 0 && g.rand(0, 1) != 0 {
				if npc.Flags&flagWalking != 0 {
					frame--
				} else {
					frame++
				}
				if frame > 3 {
					frame = 2
					npc.Flags |= flagWalking
				}
				if frame < 1 {
					frame = 2
					npc.Flags &^= flagWalking
				}
				changed = true
			}
		case t == ObjBird:
			if frame < 3 {
				if g.rand(0, 31) == 0 {
					frame, changed = 3, true
				} else if g.rand(0, 1) != 0 {
					if frame++; frame > 2 {
						frame = 0
					}
					changed = true
				}
			}
		case t == ObjTangleVine:
			if g.rand(0, 7) == 0 {
				frame ^= 1
				changed = true
			}
		case t == ObjHydra:
			changed = g.animateHydraHeads(o.Pos.X, o.Pos.Y)
			frame = o.Frame()
			if g.rand(0, 1) != 0 {
				if frame == 0 || frame < 3 && g.rand(0, 4) == 0 {
					npc.Flags |= flagWalking
				}
				if frame == 3 || frame > 0 && g.rand(0, 4) == 0 {
					npc.Flags &^= flagWalking
				}
				if npc.Flags&flagWalking != 0 {
					frame++
				} else {
					frame--
				}
				changed = true
			}
		case t == ObjDragon:
			if g.rand(0, 1) != 0 {
				frame ^= 1
				changed = true
			}
			const tailFrames = 24 // 0x18
			for d := 0; d < len(dirStepX); d += 2 {
				part := g.dragonPart(o.Pos.X, o.Pos.Y, int(g.MapZ), d, o.Frame())
				if part < 0 {
					continue
				}
				p := g.object(part)
				if p.Frame() >= tailFrames {
					const partDirection = 56 // 0x38
					p.SetFrame(p.Frame()&partDirection | frame&7)
				} else if g.rand(0, 7) == 0 {
					p.SetFrame(p.Frame() ^ 1)
					changed = true
				}
			}
		case t == ObjSheep || t == ObjDog || t == ObjDeer || t == ObjGiantRat || t == ObjSnake ||
			t == ObjGiantSpider || t >= ObjGiantScorpion || t == ObjWolf || t == ObjSeaSerpent ||
			t >= ObjDaemon && t <= ObjMongbat:
			if g.rand(0, 7) == 0 && (t != ObjCow || g.rand(0, 3) == 0) {
				frame ^= 1
				changed = true
				if head := g.objectHead(i); head >= 0 {
					h := g.object(head)
					h.SetFrame(h.Frame() ^ 1)
				}
			}
		}
		refresh = refresh || changed
		o.SetFrame(frame&stepMask + base)
	}
	g.animationTick = false
	if refresh {
		g.ShowObjects()
	}
}

// animateHydraHeads moves the heads around the hydra at (x, y).
func (g *Game) animateHydraHeads(x, y int) bool {
	changed := false
	for d := range len(dirStepX) {
		for k := g.FindLoc(x+dirStepX[d], y+dirStepY[d], int(g.MapZ)); k >= 0; k = g.NextLoc() {
			head := g.object(k)
			const headDirectionShift, headFrames = 2, 28 // 0x1c
			if head.Type() == ObjHydra2 && head.Frame()>>headDirectionShift == d {
				if g.rand(0, 1) != 0 {
					f := head.Frame()
					head.SetFrame(f&headFrames + (f+1)&stepMask)
					changed = true
				}
				break
			}
		}
	}
	return changed
}

// objectHead returns the other half of a two-tile creature (a horse, a ship, a dragon),
// or n itself.
// C: seg_1184:0153 (OBJ_getHead)
func (g *Game) objectHead(n int) int {
	if n < 0 {
		return noObject
	}
	o := g.object(n)
	t := o.Type()
	switch {
	case t >= ObjGiantScorpion || t == ObjDragon || t == ObjShip:
		x, y := o.Pos.X, o.Pos.Y
		var other byte
		switch g.TerrainType[g.tileFrame(n)] & terrainSides {
		case terrainNorth:
			y--
			other = terrainSouth
		case terrainEast:
			x++
			other = terrainWest
		case terrainSouth:
			y++
			other = terrainNorth
		case terrainWest:
			x--
			other = terrainEast
		default:
			return n
		}
		for i := g.FindLoc(x, y, int(g.MapZ)); i >= 0; i = g.NextLoc() {
			if g.object(i).Type() == t && g.TerrainType[g.tileFrame(i)]&other != 0 {
				return i
			}
		}
	case t == ObjSilverSerpent && n >= npcCount && g.object(o.Quantity()).Type() == ObjSilverSerpent:
		return o.Quantity()
	}
	return n
}

// directionFrame returns the frame of a creature type facing dir.
// C: seg_1184:0073 (OBJ_MakeDirFrame)
func directionFrame(t ObjectType, dir int) int {
	switch {
	case t == ObjChair || t == ObjCannon || t == ObjMouse || t == ObjRabbit || t == ObjCat:
		return dir >> 1
	case t == ObjSheep || t == ObjDog || t == ObjDeer || t == ObjWolf || t == ObjSnake ||
		t == ObjGiantSpider || t >= ObjGiantScorpion || t == ObjSilverSerpent ||
		t >= ObjDaemon && t <= ObjMongbat || t == ObjGiantRat || t == ObjSeaSerpent || t == ObjMusician2:
		return dir
	case t == ObjWingedGargoyle:
		return dir<<1 + dir<<2 + 7
	case t == ObjCyclops:
		return dir<<2 + 3
	case t == ObjGargoyle:
		return dir>>1 + dir + 1
	case t >= ObjFighter && t <= ObjWoman || t >= ObjLordBritish && t <= ObjAvatar:
		return dir<<1 + 1
	}
	return 0
}
