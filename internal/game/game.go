// Package game is GAME.EXE, the Ultima VI game program, ported to Go.
//
// The C globals of the original are fields of Game. The parts that are saved (the object
// tables, the NPC tables and the block of game variables at D_2C4A) keep the original
// file layout; savegame.go reads and writes them field by field.
package game

import (
	"io/fs"

	"github.com/memmaker/u6-go/internal/vga"
)

const (
	// objectCount is the number of object slots (ObjStatus[] and friends, 0xc00 entries).
	objectCount = 3072
	// npcCount is the number of slots that are characters (NPCs, party members, monsters);
	// their extra tables (NPCStatus[], STREN[], ...) have this many entries.
	npcCount = 256
	// firstMonster is the first NPC slot used for monsters created at run time (0xe0).
	firstMonster = 224
	monsterSlots = npcCount - firstMonster
	// mapList is the slot whose link starts the list of objects in memory (Link[0x100]).
	mapList = 256
	// firstFreeObject is the first slot for ordinary objects (0x101).
	firstFreeObject = mapList + 1

	partyMax   = 16
	nameLength = 14

	pathSlots = 8
	pathSteps = 32

	// the view of the map: 11x11 tiles around the party
	viewTiles  = 11
	viewCenter = 5
	// AreaTiles and friends hold 40x40 tiles around the view
	areaSize = 40
)

// ObjectType is the type of an object, the low 10 bits of its shape (OBJ_xxx).
type ObjectType int

// Coordinate encoding (struct coord): x in 10 bits, y in the next 10, z in the top 4 bits.
// For an object that is not on the map the low 16 bits are the number of its owner.
const (
	coordBits = 10
	coordMask = 1<<coordBits - 1
	// ownerYBits are the bits of y that hold the top of an owner's number
	ownerYBits = 1<<(16-coordBits) - 1
	// dungeonCoordMask keeps dungeon coordinates (256x256 levels) in range
	dungeonCoordMask = 255
)

// Coord is where an object is (struct coord).
type Coord struct {
	X, Y, Z int
}

// Owner is the object that holds an object that is not on the map (GetAssoc): the original
// shares the bits of x and the low bits of y with it.
func (c Coord) Owner() int { return c.X | (c.Y&ownerYBits)<<coordBits }

// SetOwner stores the owner's number over x and the low bits of y (SetAssoc).
func (c *Coord) SetOwner(owner int) {
	owner &= 1<<16 - 1
	c.X = owner & coordMask
	c.Y = c.Y&^ownerYBits | owner>>coordBits
}

// ObjStatus is an object's status bits (ObjStatus[]).
type ObjStatus byte

const (
	StatusOwned     ObjStatus = 1
	StatusInvisible ObjStatus = 2
	StatusCharmed   ObjStatus = 4
	// coordUseMask selects what Pos means: OnMap, Contained, InInventory or Equipped.
	coordUseMask ObjStatus = 24
	OnMap        ObjStatus = 0
	Contained    ObjStatus = 8
	InInventory  ObjStatus = 16
	Equipped     ObjStatus = 24
	StatusLocal  ObjStatus = 32
	// StatusCursed is also "hatched" for eggs and "mutant".
	StatusCursed ObjStatus = 64
	StatusLit    ObjStatus = 128
)

// Shape encoding: the type in the low 10 bits, the frame above.
const (
	typeBits   = 10
	typeMask   = 1<<typeBits - 1
	byteBits   = 8
	byteMask   = 1<<byteBits - 1
	signedByte = 1 << (byteBits - 1)
)

// Object is one entry of the object tables.
type Object struct {
	Status ObjStatus
	Pos    Coord
	// Shape is the type and the frame (ObjShapeType[]); 0 means the slot is unused.
	Shape uint16
	// Amount is the quantity in the low byte and the quality in the high byte, or, for
	// money, a 16-bit amount.
	Amount uint16
}

func shapeOf(t ObjectType, frame int) uint16 { return uint16(int(t) | frame<<typeBits) }

func (o *Object) Type() ObjectType        { return ObjectType(o.Shape & typeMask) }
func (o *Object) Frame() int              { return int(o.Shape >> typeBits) }
func (o *Object) SetFrame(frame int)      { o.Shape = shapeOf(o.Type(), frame) }
func (o *Object) CoordUse() ObjStatus     { return o.Status & coordUseMask }
func (o *Object) SetCoordUse(u ObjStatus) { o.Status = o.Status&^coordUseMask | u }
func (o *Object) Quantity() int           { return int(o.Amount & byteMask) }
func (o *Object) Quality() int            { return int(o.Amount >> byteBits) }
func (o *Object) Has(s ObjStatus) bool    { return o.Status&s != 0 }

// NPCStatus is a character's status bits (NPCStatus[]).
type NPCStatus byte

const (
	Protected NPCStatus = 1
	Paralyzed NPCStatus = 2
	Asleep    NPCStatus = 4
	Poisoned  NPCStatus = 8
	Dead      NPCStatus = 16
	// alignmentMask selects Neutral, Evil, Good or Chaotic
	alignmentMask NPCStatus = 96
	// PlayerControl marks party members.
	PlayerControl NPCStatus = 128
)

// NPCFlags are more character bits (NPCFlag[]).
type NPCFlags byte

const (
	directionMask NPCFlags = 7
	flagSkipTest  NPCFlags = 8
	// flagDraggedUnder hides a character carried by another (a dragged body).
	flagDraggedUnder NPCFlags = 16
	flagWalking      NPCFlags = 128
)

// NPC holds the tables that only characters have.
type NPC struct {
	Status        NPCStatus
	Strength      byte
	Dexterity     byte
	Intelligence  byte
	Experience    uint16
	HitPoints     byte
	Level         byte
	ScheduleIndex byte
	// Mode is the AI mode (NPCMode[], AI_xxx); CombatMode is the one used in combat
	// (NPCComMode[]).
	Mode       AIMode
	CombatMode AIMode
	Magic      byte
	MovePoints int8
	// OrigShape is the shape the character had before it died or was changed.
	OrigShape uint16
	TalkFlags byte
	Leader    byte
	Flags     NPCFlags
}

func (n *NPC) Has(s NPCStatus) bool { return n.Status&s != 0 }
func (n *NPC) Direction() int       { return int(n.Flags & directionMask) }
func (n *NPC) SetDirection(d int)   { n.Flags = n.Flags&^directionMask | NPCFlags(d)&directionMask }

// AIMode is how a character acts (ai.h).
type AIMode byte

const (
	AIMotionless AIMode = 0
	AIFollow     AIMode = 1
	AICommand    AIMode = 2
	AIAssault    AIMode = 8
	AISchedule   AIMode = 128
)

// Game is the state of GAME.EXE.
type Game struct {
	dir   string // the game directory, where SAVEGAME and MAP are written
	files fs.FS
	gr    *vga.Driver

	// saved in SAVEGAME/OBJLIST (the first npcCount objects) and SAVEGAME/OBJBLKxx
	Objects   [objectCount]Object
	NPCs      [npcCount]NPC
	Names     [partyMax][nameLength]byte
	Party     [partyMax]byte
	PartySize byte
	// Paths are the steps of the paths NPCs are following (D_8C42).
	Paths [pathSlots][pathSteps]byte
	SaveBlock

	tables
	objectLists
	worldMap
	view
	console
	input
	clock
	turns
	status
	colours colours

	random randomState
	// osiTicks counts timer ticks for osiDelay (OSILIB delay counter)
	osiTicks int
}

// object returns object n.
func (g *Game) object(n int) *Object { return &g.Objects[n] }

// isPlayerControlled says whether n is a party member (IsPlrControl).
func (g *Game) isPlayerControlled(n int) bool { return g.NPCs[n].Has(PlayerControl) }

// inVehicle says whether the party is travelling in a vehicle (IN_VEHICLE).
func (g *Game) inVehicle() bool { return g.Active == g.PartySize }

// tileFrame is the tile showing object n (TILE_FRAME).
func (g *Game) tileFrame(n int) int {
	o := &g.Objects[n]
	return int(g.BaseTile[o.Type()]) + o.Frame()
}

// Direction steps (DirIncrX/DirIncrY), north first, clockwise.
var (
	dirStepX = [8]int{0, 1, 1, 1, 0, -1, -1, -1}
	dirStepY = [8]int{-1, -1, 0, 1, 1, 1, 0, -1}
)

// cString returns the text of a NUL-terminated byte array.
func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
