package game

import "github.com/memmaker/u6-go/internal/sound"

// The passing of time: spells running out, torches burning, the date, the moons and the
// moongates, the wind and the light (seg_0A33), and the sky over the party list
// (seg_2FC1).

// clock is the state of the time functions.
type clock struct {
	// lastHour is the hour last shown (D_0369); noHour before the first.
	lastHour int
	// promptFlag asks the main loop to show the prompt again (PromptFlag).
	promptFlag bool
	// StatusDirty asks for the status area to be drawn again.
	StatusDirty bool
	// musicClock is the hour the clock tower music strikes (MUS_Clock)
	musicClock int
}

const noHour = 255

const (
	hoursPerDay    = 24
	minutesPerHour = 60
	daysPerMonth   = 28
	monthsPerYear  = 12
	daysPerWeek    = 7
	// the sun is up from dawnHour to duskHour
	dawnHour = 5
	duskHour = 19
	// daylight is AmbientLight at day
	daylight = 7
)

// Messages (seg_356A).
const (
	burnOutMsg      = "A torch burned out.\n"
	ringVanishedMsg = "A %s has vanished!\n"
	dateMsg         = "%2d-%02d-%04d Wind:%2s\n"
)

// moonPhases gives the phases of the two moons on each day of the month (D_036A).
var moonPhases = [daysPerMonth][2]int{
	{0, 0}, {7, 0}, {7, 7}, {6, 6}, {6, 5}, {5, 4}, {5, 3}, {4, 2}, {3, 1}, {3, 0},
	{2, 0}, {2, 7}, {1, 6}, {1, 5}, {0, 4}, {7, 3}, {7, 2}, {6, 1}, {6, 0}, {5, 0},
	{5, 7}, {4, 6}, {3, 5}, {3, 4}, {2, 3}, {2, 2}, {1, 1}, {1, 0},
}

// windNames are the directions the wind blows from, calm first (D_0357).
var windNames = [9]string{"C ", "N ", "NE", "E ", "SE", "S ", "SW", "W ", "NW"}

// ring and cloak chances: one in a thousand minutes, on these numbers
const (
	vanishRange          = 1000
	invisibilityVanishes = 429 // 0x1ad
	regenerationVanishes = 734 // 0x2de
	stormCloakVanishes   = 588 // 0x24c
)

// maxHitPoints is the most hit points a character can have.
// C: seg_2337:0529 (MaxHP)
func (g *Game) maxHitPoints(n int) int {
	const perLevel, most = 30, 255
	hp := int(g.NPCs[n].Level) * perLevel
	if hp == 0 {
		hp = 1
	}
	return min(hp, most)
}

// maxMagic is the most magic points a character can have.
// C: seg_2337:0559 (MaxMagic)
func (g *Game) maxMagic(n int) int {
	intel := int(g.NPCs[n].Intelligence)
	magic := 0
	t := g.object(n).Type()
	if t == ObjAvatar {
		magic = intel << 1
	}
	if t == ObjMage {
		magic = intel
	} else if t == ObjSwashbuckler || t == ObjMusician {
		magic = intel >> 1
	}
	return magic
}

// isEclipse says whether the sun is eclipsed: the first of every third month, or by the
// spell.
func (g *Game) isEclipse() bool {
	const eclipseMonths = 3
	return g.Day == 1 && g.Month%eclipseMonths == 0 || g.SpellFx[spellEclipse] != 0
}

// passTime lets minutes pass.
// C: seg_0A33:1355 (C_0A33_1355)
func (g *Game) passTime(minutes int) {
	if g.SpellFx[spellTimeStop] != 0 {
		// the original also told the driver whether time stood still (tGr._0e), which
		// U6MCGA.DRV ignores
		if int(g.SpellFx[spellTimeStop]) > minutes {
			g.SpellFx[spellTimeStop] -= uint8(minutes)
			return
		}
		g.SpellFx[spellTimeStop] = 0
	}
	for i := range g.SpellFx {
		if g.SpellFx[i] != 0 {
			if int(g.SpellFx[i]) <= minutes {
				g.SpellFx[i] = 0
			} else {
				g.SpellFx[i] -= uint8(minutes)
			}
		}
	}
	if int(g.KegMinutes) > minutes {
		g.KegMinutes -= int16(minutes)
	} else if g.KegMinutes != 0 {
		g.KegMinutes = 0
		g.detonate(int(g.KegObject))
	}
	if g.rand(0, 7) == 0 {
		g.erupting = 1
	}
	torchLit, cloakWorn := false, false
	// ringWorn is only reset for party members: the others see the last member's
	ringWorn := false
	for i := range npcCount {
		o := g.object(i)
		if o.Shape == 0 {
			continue
		}
		npc := &g.NPCs[i]
		if g.isPlayerControlled(i) {
			ringWorn = false
			vanished, what := false, "ring"
			for di := g.FindInv(i); di >= 0; di = g.NextInv() {
				item := g.object(di)
				if item.CoordUse() == InInventory {
					continue
				}
				switch item.Type() {
				case ObjTorch:
					if item.Frame() != 1 {
						break
					}
					if g.rand(0, 1) != 0 {
						if item.Quantity() <= minutes {
							g.printf(burnOutMsg)
							g.DeleteObj(di)
							g.promptFlag = true
						} else {
							torchLit = true
							item.Amount -= uint16(minutes)
						}
					} else {
						torchLit = true
					}
				case ObjInvisibilityRing:
					if g.rand(0, vanishRange) == invisibilityVanishes {
						vanished = true
						o.Status &^= StatusInvisible
						g.screenFade = true
						g.drawMap(true)
					} else {
						ringWorn = true
					}
				case ObjRegenerationRing:
					if g.StatusDisplay == cmdStatusParty || g.StatusDisplay == cmdStatusPortrait && int(g.Party[g.statusMember]) == i {
						if int(npc.HitPoints) < g.maxHitPoints(i) {
							g.StatusDirty = true
						}
					}
					npc.HitPoints = byte(min(int(npc.HitPoints)+minutes, g.maxHitPoints(i)))
					if g.rand(0, vanishRange) == regenerationVanishes {
						vanished = true
					}
				case ObjStormCloak:
					if g.rand(0, vanishRange) == stormCloakVanishes {
						what = "cloak"
						vanished = true
						g.SpellFx[spellStormCloak] = 0
					} else {
						cloakWorn = true
					}
				}
				if vanished {
					g.printf(ringVanishedMsg, what)
					g.sfx(sound.EffectSweep, 0)
					g.DeleteObj(di)
					g.promptFlag = true
					vanished = false
				}
			}
		}
		for range minutes {
			if o.Has(StatusInvisible) && !ringWorn && g.rand(0, 63) == 0 && int(npc.Intelligence) <= g.rand(1, 30) {
				o.Status &^= StatusInvisible
			}
			if npc.Has(Protected) && g.rand(0, 63) == 0 && int(npc.Intelligence) <= g.rand(1, 30) {
				npc.Status &^= Protected
			}
			if o.Has(StatusCursed) && g.rand(0, 15) == 0 && int(npc.Intelligence) >= g.rand(1, 30) {
				o.Status &^= StatusCursed
			}
			if o.Has(StatusCharmed) && g.rand(0, 7) == 0 && int(npc.Intelligence) >= g.rand(1, 30) {
				g.stopCharm(i)
			}
			if npc.Has(Paralyzed) && g.rand(0, 3) == 0 && int(npc.Strength) >= g.rand(1, 30) {
				npc.Status &^= Paralyzed
			}
			if npc.Has(Asleep) && npc.Mode != AISleep && g.rand(0, 15) == 0 {
				npc.Status &^= Asleep
				npc.Flags |= flagSkipTest
				o.Shape = npc.OrigShape
			}
			if npc.Has(Poisoned) && !npc.Has(Protected) && g.rand(0, 7) == 0 {
				g.loseHitPoints(i, 1)
			}
		}
		g.checkStepOn(i)
		npc.Flags &^= flagSkipTest
	}
	if cloakWorn {
		g.SpellFx[spellStormCloak] = 1
	}

	minute := int(g.Minute) + minutes
	for ; minute >= minutesPerHour; minute -= minutesPerHour {
		if g.Hour++; g.Hour >= hoursPerDay {
			g.Hour = 0
			if g.Day++; g.Day > daysPerMonth {
				g.Day = 1
				if g.Month++; g.Month > monthsPerYear {
					g.Month = 1
					g.Year++
				}
			}
		}
		if g.NextSleep != 0 {
			g.NextSleep--
		}
		if g.DrunkCounter != 0 {
			g.DrunkCounter--
		}
		g.followSchedules()
		for i := range int(g.PartySize) {
			p := int(g.Party[i])
			if t := g.object(p).Type(); t == ObjAvatar || t == ObjMage || t == ObjSwashbuckler || t == ObjMusician {
				npc := &g.NPCs[p]
				npc.Magic += npc.Level
				if g.maxMagic(p) < int(npc.Magic) {
					npc.Magic = byte(g.maxMagic(p))
				}
				g.StatusDirty = true
			}
		}
	}
	if g.lastHour != int(g.Hour) {
		if g.lastHour != noHour {
			g.Unused2C6E = 0
			g.CaughtFish = 0
			g.MustRingBell = 1
			const clockHours = 12
			g.musicClock = int(g.Hour) % clockHours
			if g.musicClock == 0 {
				g.musicClock = clockHours
			}
		}
		g.lastHour = int(g.Hour)
		const nightStart, sundialStep = 20, 1
		sundial := tileSundialOff
		if g.Hour <= nightStart && g.Hour >= dawnHour {
			sundial = tileSundial + int(g.Hour-dawnHour)>>sundialStep
		}
		g.gr.SetTileFrame(int(g.BaseTile[ObjSundial]), sundial)
		const hoursPerPhase, trammelRise, feluccaRise = 3, 18, 20
		g.MoonPhases[0] = uint8(moonPhases[g.Day-1][0])
		g.MoonPhases[1] = uint8((int(g.MoonPhases[0])*hoursPerPhase + trammelRise - int(g.Hour)) % hoursPerDay)
		g.MoonPhases[2] = uint8(moonPhases[g.Day-1][1])
		g.MoonPhases[3] = uint8((int(g.MoonPhases[2])*hoursPerPhase + feluccaRise - int(g.Hour)) % hoursPerDay)
		g.drawSky()
		g.updateMoongates()
	}
	g.changeWind(false)
	g.Minute = uint8(minute)
	before := g.AmbientLight
	switch {
	case g.isEclipse() || g.Hour < dawnHour || g.Hour > duskHour || g.MapZ > 0 && g.MapZ < dungeonLevels:
		g.AmbientLight = 0
	case g.Hour == dawnHour:
		g.AmbientLight = int8(g.Minute/10 + 1)
	case g.Hour == duskHour:
		g.AmbientLight = int8((minutesPerHour-1-int(g.Minute))/10 + 1)
	default:
		g.AmbientLight = daylight
	}
	const lampLight = 4
	if g.AmbientLight < lampLight && (torchLit || g.SpellFx[spellLight] != 0) {
		g.AmbientLight = lampLight
	}
	if g.AmbientLight != before {
		g.computeView()
	}
	g.setClip(clipScreen)
	const dateX, dateY = 2, 21
	g.gotoxy(dateX, dateY)
	g.printf(dateMsg, int(g.Month), int(g.Day), int(g.Year), windNames[int(g.WindDir)+1])
	g.setClip(clipMessage)
}

// The moons rise and set every day; a moongate stands at each moonstone while one of
// them is up (its place in the sky below this).
const moonHigh = 15

// updateMoongates puts up the moongates at the moonstones in the area while a moon is up
// and takes them down when both are down.
// C: seg_0A33:121A (C_0A33_121A)
func (g *Game) updateMoongates() {
	changed := false
	for _, stone := range g.Moonstones {
		x, y, z := int(stone[0]), int(stone[1]), int(stone[2])
		if z != int(g.MapZ) || x < g.AreaX || x >= g.AreaX+areaSize || y < g.AreaY || y >= g.AreaY+areaSize {
			continue
		}
		gate := g.findTypeAt(ObjMoongate, x, y)
		if g.MoonPhases[1] < moonHigh || g.MoonPhases[3] < moonHigh {
			if gate < 0 {
				g.AddMapObj(shapeOf(ObjMoongate, 1), 0, 0, x, y, z)
				changed = true
			}
		} else if gate >= 0 {
			g.DeleteObj(gate)
			changed = true
		}
	}
	if changed {
		g.computeView()
		g.viewChanged = true
	}
}

// noWind is WindDir when there is none
const noWind = -1

// changeWind sometimes turns the wind; there is none in the dungeons.
// C: seg_0A33:12F6 (C_0A33_12F6)
func (g *Game) changeWind(noChange bool) {
	if g.MapZ > 0 && g.MapZ < dungeonLevels && g.WindDir >= 0 {
		g.WindDir = noWind
		noChange = true
	}
	if noChange || g.rand(0, 63) != 0 {
		return
	}
	if g.MapZ == 0 || g.MapZ == dungeonLevels {
		g.WindDir = int8(g.rand(0, 8) - 1)
	}
}

// eruptions: a volcano shakes the ground, a fumarole may blow
// C: seg_0A33:117A (C_0A33_117A)
func (g *Game) erupt() {
	if g.erupting <= 1 {
		return
	}
	o := g.object(g.erupting)
	switch o.Type() {
	case ObjVolcano:
		g.shakeScreen(10, 2)
	case ObjFumarole:
		if g.rand(0, 1) != 0 {
			g.shakeScreen(10, 2)
		} else {
			n := g.AddMapObj(shapeOf(ObjFumarole, 0), 0, 0, o.Pos.X, o.Pos.Y, o.Pos.Z)
			g.detonate(n)
		}
	}
	g.erupting = 0
}

// the heights of the sun and the moons over the sky's 15 places (D_2BFA)
var skyHeights = [15]int{10, 7, 5, 3, 2, 1, 0, 0, 0, 1, 2, 3, 5, 7, 10}

// drawSky draws the sun and the moons (or the cave roof) over the party list.
// C: seg_2FC1:19C5 (C_2FC1_19C5)
func (g *Game) drawSky() {
	if g.StatusDisplay != cmdStatusParty && g.StatusDisplay != cmdStatusContainer {
		return
	}
	const skyTiles, skyY, step = 9, 4, 8
	for i := range skyTiles {
		g.gr.DrawTileOffscreen(tileStatusBackground, i*tileSize, skyY)
	}
	if g.MapZ == 0 || g.MapZ == dungeonLevels {
		if g.Hour > dawnHour-1 && g.Hour < duskHour+1 {
			sun := tileSun
			switch {
			case g.isEclipse():
				sun = tileSunEclipse
			case g.Hour == dawnHour || g.Hour == duskHour:
				sun = tileSunLow
			}
			place := duskHour - int(g.Hour)
			g.gr.DrawTileOffscreen(sun, place*step, skyHeights[place])
		}
		if !g.isEclipse() {
			for moon := 0; moon < 2; moon++ {
				phase, place := int(g.MoonPhases[moon*2]), int(g.MoonPhases[moon*2+1])
				if place >= 0 && place < len(skyHeights) {
					g.gr.DrawTileOffscreen(int(g.BaseTile[ObjMoonstone])+phase, place*step, skyHeights[place])
				}
			}
		}
		for i := range skyTiles {
			g.gr.DrawTileOffscreen(tileMountains+i, i*tileSize, skyY)
		}
	} else {
		g.gr.DrawTileOffscreen(tileCaveLeft, 0, skyY)
		for i := 1; i < skyTiles-1; i++ {
			g.gr.DrawTileOffscreen(tileCave, i*tileSize, skyY)
		}
		g.gr.DrawTileOffscreen(tileCaveRight, (skyTiles-1)*tileSize, skyY)
	}
	const width, height, toX, toY = 135, 15, 176, 6
	g.gr.CopyBackBufferToScreen(0, skyY, width, skyY+height, toX, toY)
	g.changeWind(true)
}

// checkStepOn hurts a character standing on something harmful.
// C: seg_0A33:0FEB (C_0A33_0FEB)
func (g *Game) checkStepOn(n int) {
	o := g.object(n)
	t := o.Type()
	if t == ObjBalloon2 || t == ObjSkiff || t == ObjRaft || t == ObjShip {
		return
	}
	cls := monsterClass(t)
	if cls >= 0 && classHas(cls, flies) {
		return
	}
	if g.NPCs[n].Flags&flagSkipTest != 0 || o.Pos.Z != int(g.MapZ) {
		return
	}
	mask := surfaceMask
	if g.MapZ != 0 {
		mask = dungeonMask
	}
	areaX := (int(g.MapX) - areaMargin) & areaAlign
	areaY := (int(g.MapY) - areaMargin) & areaAlign
	x, y := o.Pos.X, o.Pos.Y
	if (x-areaX)&mask >= areaSize || (y-areaY)&mask >= areaSize {
		return
	}
	if tile := g.GetTileAtXYZ(x, y, int(g.MapZ)); g.isDamaging(tile) {
		g.terrainDamage(n, tile)
		return
	}
	for di := g.FindLoc(x, y, int(g.MapZ)); di >= 0; di = g.NextLoc() {
		if di == n {
			continue
		}
		if tile := g.tileFrame(di); g.isDamaging(tile) {
			g.object(di).Status &^= StatusInvisible
			g.terrainDamage(n, tile)
			break
		}
	}
}
