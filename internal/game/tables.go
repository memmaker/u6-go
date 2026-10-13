package game

import (
	"encoding/binary"
	"fmt"
	"io/fs"
)

// The tables loaded at start: tile properties (TILEFLAG), the first tile of each object
// type (BASETILE), the animated tiles (ANIMDATA) and the NPC schedules (SCHEDULE).

const (
	tileCount = 2048
	typeCount = 1024
	// animSlots is MAX_ANIM.
	animSlots = 32
	// scheduleSlots is the size of Schedule[] (600 entries).
	scheduleSlots = 600
)

// Terrain bits (TerrainType[], TERRAIN_FLAG_xx). The high nibble of a walkable tile is
// its movement cost; of a wall, the sides it blocks.
const (
	terrainWet        = 1
	terrainImpassable = 2
	terrainWall       = 4
	terrainDamaging   = 8
	terrainWest       = 16
	terrainSouth      = 32
	terrainEast       = 64
	terrainNorth      = 128
	terrainCostShift  = 4
	terrainLowMask    = 15
	terrainSides      = 240
)

// Tile bits (TileFlag[], TILE_FLAG1_xx).
const (
	tileLightMask    = 3
	tileOpaque       = 4
	tileWindow       = 8
	tileForeground   = 16
	tileNoShoot      = 32
	tileDoubleHeight = 64
	tileDoubleWidth  = 128
)

// More tile bits (D_B3EF[], TILE_FLAG2_xx).
const (
	tileWarm         = 1
	tileSupporting   = 2
	tileBreakthrough = 4
	tileGeneric      = 8
	tileIgnore       = 16
	tileBackground   = 32
	tileArticle      = 192
)

// scheduleEntry is one step of an NPC's day (struct tSchedule).
type scheduleEntry struct {
	// Time is the hour in the low 5 bits and the day of the week (1..7, 0 for every day)
	// above.
	Time   byte
	Action byte
	Pos    [coordBytes]byte
}

// tables are the tables read from the game files.
type tables struct {
	TerrainType [tileCount]byte
	TileFlag    [tileCount]byte
	TypeWeight  [typeCount]byte
	TileFlag2   [tileCount]byte // D_B3EF
	// BaseTile is the first tile of each object type.
	BaseTile [typeCount]uint16
	// animated tiles: count (NumAnimData), tile (D_67E4), first frame (D_9DF5), counter
	// mask (D_9495) and shift (D_B3F7), and whether each runs forward, backward or not at
	// all (StateAnimData)
	animCount int
	animTile  [animSlots]uint16
	animFrame [animSlots]int16
	animMask  [animSlots]byte
	animShift [animSlots]byte
	animState [animSlots]byte
	// schedulePointer is where each NPC's entries start in schedule (SchedPointer).
	schedulePointer [npcCount + 1]uint16
	schedule        [scheduleSlots]scheduleEntry
}

// animation states (StateAnimData)
const (
	animStopped  = 0
	animForward  = 1
	animBackward = 2
)

func (t *tables) isWall(tile int) bool       { return t.TerrainType[tile]&terrainWall != 0 }
func (t *tables) isImpassable(tile int) bool { return t.TerrainType[tile]&terrainImpassable != 0 }
func (t *tables) isDamaging(tile int) bool   { return t.TerrainType[tile]&terrainDamaging != 0 }
func (t *tables) tileLight(tile int) int     { return int(t.TileFlag[tile] & tileLightMask) }
func (t *tables) isOpaque(tile int) bool     { return t.TileFlag[tile]&tileOpaque != 0 }
func (t *tables) isWindow(tile int) bool     { return t.TileFlag[tile]&tileWindow != 0 }
func (t *tables) isForeground(tile int) bool { return t.TileFlag[tile]&tileForeground != 0 }
func (t *tables) isWarm(tile int) bool       { return t.TileFlag2[tile]&tileWarm != 0 }
func (t *tables) isBackground(tile int) bool { return t.TileFlag2[tile]&tileBackground != 0 }
func (t *tables) isBreakthrough(tile int) bool {
	return t.TileFlag2[tile]&tileBreakthrough != 0
}
func (t *tables) isIgnored(tile int) bool { return t.TileFlag2[tile]&tileIgnore != 0 }

// readTable reads the parts of a file one after the other.
func readTable(files fs.FS, name string, parts ...any) error {
	raw, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	at := 0
	for _, p := range parts {
		n, err := binary.Decode(raw[at:], binary.LittleEndian, p)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		at += n
	}
	return nil
}

// loadTileTables reads ANIMDATA and TILEFLAG.
// C: seg_0903:04DB (C_0903_04DB)
func (t *tables) loadTileTables(files fs.FS) error {
	var count uint16
	if err := readTable(files, "ANIMDATA", &count, &t.animTile, &t.animFrame, &t.animMask, &t.animShift); err != nil {
		return err
	}
	t.animCount = min(int(count), animSlots)
	return readTable(files, "TILEFLAG", &t.TerrainType, &t.TileFlag, &t.TypeWeight, &t.TileFlag2)
}

// loadSchedules reads SCHEDULE and BASETILE (the first part of C_0C9C_042A).
func (t *tables) loadSchedules(files fs.FS) error {
	raw, err := fs.ReadFile(files, "SCHEDULE")
	if err != nil {
		return err
	}
	n, err := binary.Decode(raw, binary.LittleEndian, &t.schedulePointer)
	if err != nil {
		return fmt.Errorf("SCHEDULE: %w", err)
	}
	entries := min(int(t.schedulePointer[npcCount]), scheduleSlots)
	if _, err := binary.Decode(raw[n:], binary.LittleEndian, t.schedule[:entries]); err != nil {
		return fmt.Errorf("SCHEDULE: %w", err)
	}
	return readTable(files, "BASETILE", &t.BaseTile)
}

// Monster classes (seg_3522): the types that are creatures, and what each can do.
var classTypes = [...]ObjectType{
	ObjAcidSlug, ObjAlligator, ObjGiantAnt, ObjAvatar, ObjGiantBat, ObjPeasant, ObjBird, ObjChild,
	ObjCorpser, ObjCow, ObjCyclops, ObjDaemon, ObjDeer, ObjDog, ObjDragon, ObjDrake,
	ObjWoman, ObjFarmer, ObjFighter, ObjWingedGargoyle, ObjGargoyle, ObjGazer, ObjGhost, ObjGremlin,
	ObjGuard, ObjHeadless, ObjHorse, ObjMountedHorse, ObjHydra, ObjInsects, ObjJester, ObjLordBritish,
	ObjMusician, ObjMage, ObjMerchant, ObjOpenChest, ObjMongbat, ObjMouse, ObjRabbit, ObjGiantRat,
	ObjReaper, ObjRotWorms, ObjGiantScorpion, ObjSeaSerpent, ObjSheep, ObjShip, ObjSilverSerpent, ObjSkeleton,
	ObjSlime, ObjSnake, ObjGiantSpider, ObjGiantSquid, ObjSwashbuckler, ObjTangleVine2, ObjTroll, ObjVillager,
	ObjWisp, ObjWolf, ObjTangleVine, ObjSkiff, ObjRaft, ObjBalloon2, ObjCat, ObjStocks,
}

// Monster class bits (D_3522_0242, MONSTER_xx).
const (
	immuneSleep      = 1
	immuneDisease    = 2
	immunePoison     = 4
	immuneFire       = 8
	doubleFireDamage = 16
	undead           = 128
	flies            = 256
	swims            = 512
	ethereal         = 1024
	opensDoors       = 16384
)

var classFlags = [...]uint16{
	4113, 10276, 10273, 24608, 10496, 24608, 10496, 24608, 2065, 10272, 8224, 8619, 8192, 8192, 4360, 8456,
	24608, 24608, 24608, 8481, 8224, 4385, 5263, 2048, 24608, 8224, 10272, 8224, 4137, 2305, 24608, 24608,
	24608, 24608, 24608, 37, 8448, 8192, 10240, 10308, 49, 2133, 10305, 2560, 8192, 512, 4160, 175,
	4177, 8256, 8256, 2625, 24608, 49, 8224, 24608, 5163, 8192, 49, 512, 512, 256, 8192, 2048,
}

// noClassFlags are what the original reads for a type that is no creature (class -1):
// the word before the table, the last two alignments of D_3522_0202 (neutral, chaotic).
const noClassFlags = 24576

// monsterClass returns the class of a creature type, or -1.
// C: seg_2E2D:004B (GetMonsterClass)
func monsterClass(t ObjectType) int {
	for i, c := range classTypes {
		if c == t {
			return i
		}
	}
	return -1
}

// classHas says whether class cls has flag.
func classHas(cls int, flag uint16) bool {
	if cls < 0 {
		return noClassFlags&flag != 0
	}
	return classFlags[cls]&flag != 0
}

// Colours of the interface (D_2A4A..D_2A58).
type colours struct {
	highlight      byte // D_2A4A, low hit points
	poisoned       byte // D_2A4C
	defaultCommand byte // D_2A4E, the line under the default command's icon
	viewBorder     byte // D_2A50
	background     byte // D_2A54
	text           byte // D_2A56
	clockHands     byte // D_2A58
}

var defaultColours = colours{highlight: 12, poisoned: 10, defaultCommand: 9, viewBorder: 13, background: 49, text: 72, clockHands: 0}

// highBit selects the second half of the font.
const highBit = 128
