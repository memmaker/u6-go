package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/memmaker/u6-go/internal/data"
)

// SaveBlock is the block of game variables saved at the end of SAVEGAME/OBJLIST (the
// globals from D_2C4A to D_2CCC). The field order and sizes are the file layout.
type SaveBlock struct {
	IsOnQuest uint8
	NextSleep uint8
	Minute    uint8 // Time_M
	Hour      uint8 // Time_H
	Day       uint8 // Date_D
	Month     uint8 // Date_M
	Year      uint16
	Karma     uint8
	WindDir   int8
	// Active is the party member the player controls (an index into Party); PartySize
	// when the party travels in a vehicle.
	Active uint8
	// AmbientLight is the light level outdoors (D_2C55).
	AmbientLight int8
	MapX, MapY   int16
	MapZ         int16
	SpellFx      [16]uint8
	// VanishedObject is the object the Vanish spell removed (D_2C6C).
	VanishedObject int16
	Unused2C6E     uint8
	CaughtFish     uint8
	DrunkCounter   uint8
	MustRingBell   uint8
	SoundOn        int16
	// Moonstones are where the eight moonstones lie (x, y, z).
	Moonstones [8][3]int16
	// KegMinutes and KegObject are the lit powder keg about to explode.
	KegMinutes   int16
	KegObject    int16
	SpeakGargish int16
	PathObject   [pathSlots]uint8
	PathCounter  [pathSlots]uint8
	PathTries    [pathSlots]uint8
	InCombat     int8
	// SoloMember is the party member in solo mode, -1 for none (D_2CC3).
	SoloMember int8
	// SavedMode is D_2CC4.
	SavedMode      int8
	DefaultCommand uint8
	MoonPhases     [4]uint8 // D_2CC6..D_2CC9
	AvatarFemale   uint8    // D_2CCA
	// PortraitPlusOne is the avatar's portrait + 1; 0 means no character was created.
	PortraitPlusOne uint8 // D_2CCB
}

// saveBlockBytes is the size of SaveBlock in the file (0x82).
const saveBlockBytes = 130

// newSaveBlock returns the initial values of the block (D_2C4A.c).
func newSaveBlock() SaveBlock {
	return SaveBlock{
		Hour: 8, Day: 4, Month: 7, Year: 161, Karma: 75, AmbientLight: 7,
		MapX: 307, MapY: 352, SoundOn: 1,
		Moonstones: [8][3]int16{
			{935, 262, 0}, {503, 358, 0}, {159, 942, 0}, {295, 38, 0},
			{831, 166, 0}, {327, 822, 0}, {23, 22, 1}, {919, 934, 0},
		},
		InCombat: 1, SoloMember: -1, DefaultCommand: 255,
	}
}

// File names. The original lowercases them; the port uses the names as the installer
// wrote them on disk.
const (
	saveDir         = "SAVEGAME"
	objlistFile     = "OBJLIST"
	objblkPrefix    = "OBJBLK"
	tmpSuffix       = ".TMP"
	objlistFileSize = 7283 // 0x1c73
)

// coordinate bytes in a file (struct coord)
const coordBytes = 3

func encodeCoord(c Coord) [coordBytes]byte {
	return [coordBytes]byte{
		byte(c.X),
		byte(c.X>>byteBits) | byte(c.Y)<<2,
		byte(c.Y>>(byteBits-2)) | byte(c.Z)<<4,
	}
}

func decodeCoord(b [coordBytes]byte) Coord {
	return Coord{
		X: int(b[0]) | int(b[1]&3)<<byteBits,
		Y: int(b[1])>>2 | int(b[2]&15)<<(byteBits-2),
		Z: int(b[2]) >> 4,
	}
}

// objlist is the layout of SAVEGAME/OBJLIST.
type objlist struct {
	Status        [npcCount]uint8
	Pos           [npcCount][coordBytes]uint8
	Shape         [npcCount]uint16
	Amount        [npcCount]uint16
	NPCStatus     [npcCount]uint8
	Strength      [npcCount]uint8
	Dexterity     [npcCount]uint8
	Intelligence  [npcCount]uint8
	Experience    [npcCount]uint16
	HitPoints     [npcCount]uint8
	Names         [partyMax][nameLength]uint8
	Party         [partyMax]uint8
	PartySize     uint8
	Level         [npcCount]uint8
	ScheduleIndex [npcCount]uint8
	Mode          [npcCount]uint8
	CombatMode    [npcCount]uint8
	Magic         [npcCount]uint8
	MovePoints    [npcCount]int8
	OrigShape     [npcCount]uint16
	TalkFlags     [npcCount]uint8
	Leader        [npcCount]uint8
	Flags         [npcCount]uint8
	Paths         [pathSlots][pathSteps]uint8
	Block         SaveBlock
}

func (g *Game) toObjlist() *objlist {
	f := &objlist{Names: g.Names, Party: g.Party, PartySize: g.PartySize, Paths: g.Paths, Block: g.SaveBlock}
	for i := range npcCount {
		o, n := &g.Objects[i], &g.NPCs[i]
		f.Status[i], f.Pos[i], f.Shape[i], f.Amount[i] = uint8(o.Status), encodeCoord(o.Pos), o.Shape, o.Amount
		f.NPCStatus[i], f.Strength[i], f.Dexterity[i], f.Intelligence[i] = uint8(n.Status), n.Strength, n.Dexterity, n.Intelligence
		f.Experience[i], f.HitPoints[i], f.Level[i], f.ScheduleIndex[i] = n.Experience, n.HitPoints, n.Level, n.ScheduleIndex
		f.Mode[i], f.CombatMode[i], f.Magic[i], f.MovePoints[i] = uint8(n.Mode), uint8(n.CombatMode), n.Magic, n.MovePoints
		f.OrigShape[i], f.TalkFlags[i], f.Leader[i], f.Flags[i] = n.OrigShape, n.TalkFlags, n.Leader, uint8(n.Flags)
	}
	return f
}

func (g *Game) fromObjlist(f *objlist) {
	g.Names, g.Party, g.PartySize, g.Paths, g.SaveBlock = f.Names, f.Party, f.PartySize, f.Paths, f.Block
	for i := range npcCount {
		g.Objects[i] = Object{Status: ObjStatus(f.Status[i]), Pos: decodeCoord(f.Pos[i]), Shape: f.Shape[i], Amount: f.Amount[i]}
		g.NPCs[i] = NPC{
			Status: NPCStatus(f.NPCStatus[i]), Strength: f.Strength[i], Dexterity: f.Dexterity[i],
			Intelligence: f.Intelligence[i], Experience: f.Experience[i], HitPoints: f.HitPoints[i],
			Level: f.Level[i], ScheduleIndex: f.ScheduleIndex[i], Mode: AIMode(f.Mode[i]),
			CombatMode: AIMode(f.CombatMode[i]), Magic: f.Magic[i], MovePoints: f.MovePoints[i],
			OrigShape: f.OrigShape[i], TalkFlags: f.TalkFlags[i], Leader: f.Leader[i], Flags: NPCFlags(f.Flags[i]),
		}
	}
}

func readObjlist(path string) (*objlist, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &objlist{}
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

func writeObjlist(path string, f *objlist) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, f); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// objectRecord is an object in an OBJBLK file.
type objectRecord struct {
	Status uint8
	Pos    [coordBytes]uint8
	Shape  uint16
	Amount uint16
}

func readObjblk(r io.Reader) ([]objectRecord, error) {
	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	// the original reads at most as many objects as there are slots
	records := make([]objectRecord, min(int(count), objectCount))
	if err := binary.Read(r, binary.LittleEndian, records); err != nil {
		return nil, err
	}
	return records, nil
}

func writeObjblk(path string, records []objectRecord) error {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(len(records)))
	binary.Write(&buf, binary.LittleEndian, records)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// Object blocks: the surface is 8x8 regions of 128x128 tiles; dungeon level z (1..5) has
// one block. Region i is file OBJBLK + ('A'+i%8) + ('A'+i/8); dungeon z is ('A'+z-1) + 'I'.
const (
	regionsPerSide = 8
	surfaceRegions = regionsPerSide * regionsPerSide
	dungeonLevels  = 5
	objectBlocks   = surfaceRegions + dungeonLevels
)

// objblkName is the file name of object block b (0..63 surface regions, 64.. dungeons).
func objblkName(block int) string {
	return objblkPrefix + string(rune('A'+block%regionsPerSide)) + string(rune('A'+block/regionsPerSide))
}

func (g *Game) savePath(name string) string { return filepath.Join(g.dir, saveDir, name) }

// Initial game files: LZOBJBLK holds the 64 surface blocks one after the other, LZDNGBLK
// the 5 dungeon blocks followed by OBJLIST; LZMAP is MAP.
const (
	initialSurfaceObjects = "LZOBJBLK"
	initialDungeonObjects = "LZDNGBLK"
	initialMap            = "LZMAP"
	mapFile               = "MAP"
)

// ensureGameFiles creates MAP and SAVEGAME from the packed initial files when they are
// missing, as the installer did on the original disks.
func ensureGameFiles(dir string) error {
	unpack := func(name string) ([]byte, error) {
		packed, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		return data.Decompress(packed)
	}
	if _, err := os.Stat(filepath.Join(dir, mapFile)); errors.Is(err, os.ErrNotExist) {
		m, err := unpack(initialMap)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, mapFile), m, 0o644); err != nil {
			return err
		}
	}
	saves := filepath.Join(dir, saveDir)
	if _, err := os.Stat(filepath.Join(saves, objlistFile)); err == nil {
		return nil
	}
	if err := os.MkdirAll(saves, 0o755); err != nil {
		return err
	}
	surface, err := unpack(initialSurfaceObjects)
	if err != nil {
		return err
	}
	dungeons, err := unpack(initialDungeonObjects)
	if err != nil {
		return err
	}
	// split a file of blocks: each a 16-bit count and that many 8-byte records
	split := func(raw []byte, first, count int) ([]byte, error) {
		const recordBytes = 8
		for b := first; b < first+count; b++ {
			if len(raw) < 2 {
				return nil, errors.New("short object block file")
			}
			size := 2 + int(binary.LittleEndian.Uint16(raw))*recordBytes
			if len(raw) < size {
				return nil, errors.New("short object block file")
			}
			if err := os.WriteFile(filepath.Join(saves, objblkName(b)), raw[:size], 0o644); err != nil {
				return nil, err
			}
			raw = raw[size:]
		}
		return raw, nil
	}
	if _, err := split(surface, 0, surfaceRegions); err != nil {
		return err
	}
	rest, err := split(dungeons, surfaceRegions, dungeonLevels)
	if err != nil {
		return err
	}
	if len(rest) < objlistFileSize {
		return errors.New("LZDNGBLK: short OBJLIST")
	}
	return os.WriteFile(filepath.Join(saves, objlistFile), rest[:objlistFileSize], 0o644)
}

// Character is what U.EXE's character creation stores in the saved game.
type Character struct {
	Name                              string
	Female                            bool
	Portrait                          int
	Strength, Dexterity, Intelligence byte
	Karma                             byte
}

// DefaultCharacter is a character as U.EXE makes one when every gypsy answer is skipped.
func DefaultCharacter() Character {
	return Character{Name: "Avatar", Strength: 15, Dexterity: 15, Intelligence: 15, Karma: 75}
}

// the avatar's starting values written by U.EXE
const (
	avatar           = 1
	avatarExperience = 370
	avatarHitPoints  = 90
	avatarLevel      = 3
)

// WriteCharacter stores a new character in dir's saved game the way U.EXE does after
// character creation, creating the saved game first when it is missing. It does nothing
// when the saved game already has a character. It lets the game start without U.EXE.
// C: U.EXE seg_0406:0006 (SaveObjlist)
func WriteCharacter(dir string, c Character) error {
	if err := ensureGameFiles(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, saveDir, objlistFile)
	f, err := readObjlist(path)
	if err != nil {
		return err
	}
	if f.Block.PortraitPlusOne != 0 {
		return nil
	}
	f.Strength[avatar] = c.Strength
	f.Dexterity[avatar] = c.Dexterity
	f.Intelligence[avatar] = c.Intelligence
	f.Magic[avatar] = c.Intelligence * 2
	f.Block.AvatarFemale = 0
	if c.Female {
		f.Block.AvatarFemale = 1
	}
	f.Block.Karma = c.Karma
	f.Names[0] = [nameLength]byte{}
	copy(f.Names[0][:nameLength-1], c.Name)
	f.Experience[avatar] = avatarExperience
	f.HitPoints[avatar] = avatarHitPoints
	f.Level[avatar] = avatarLevel
	f.Block.PortraitPlusOne = byte(c.Portrait + 1)
	return writeObjlist(path, f)
}

// loadGame reads the saved game and puts the party on the map.
// C: seg_0C9C:042A (C_0C9C_042A)
func (g *Game) loadGame() error {
	g.forgetArea()
	if err := g.loadSchedules(g.files); err != nil {
		return err
	}
	f, err := readObjlist(g.savePath(objlistFile))
	if err != nil {
		return err
	}
	g.fromObjlist(f)
	g.linkCharacters()
	g.VarInt[questVariable] = int(g.IsOnQuest)
	p := g.object(int(g.Party[g.Active])).Pos
	g.MapX, g.MapY, g.MapZ = int16(p.X), int16(p.Y), int16(p.Z)
	for i := range g.animCount {
		switch g.animTile[i] {
		case g.BaseTile[ObjChain], g.BaseTile[ObjCrank], g.BaseTile[ObjBell]:
			g.animState[i] = animStopped
		default:
			g.animState[i] = animForward
		}
	}
	g.setDefaultCommand(int(g.DefaultCommand))
	g.LoadNewRegions(int(g.MapX), int(g.MapY), int(g.MapZ))
	g.gameLoaded = true
	g.enterArea()
	if g.inVehicle() {
		for i := range int(g.PartySize) {
			g.InsertObj(int(g.Party[i]), 0, InInventory)
		}
	}
	g.computeView()
	return nil
}

// saveGame writes the game to SAVEGAME.
// C: seg_0C9C:089F (C_0C9C_089F)
func (g *Game) saveGame() error {
	g.diskMode = ' '
	g.forgetArea()
	if g.inVehicle() {
		for i := range int(g.PartySize) {
			g.MoveObj(int(g.Party[i]), int(g.MapX), int(g.MapY), int(g.MapZ))
		}
	}
	g.commitObjects()
	g.IsOnQuest = uint8(g.VarInt[questVariable])
	err := writeObjlist(g.savePath(objlistFile), g.toObjlist())
	g.enterArea()
	if g.inVehicle() {
		for i := range int(g.PartySize) {
			g.InsertObj(int(g.Party[i]), 0, InInventory)
		}
	}
	g.StatusDirty = true
	return err
}
