package game

// What things are called (LOOK.LZD) and how many of them there are.

import (
	"encoding/binary"
	"io/fs"

	"github.com/memmaker/u6-go/internal/data"
)

// lookEntry is a run of tiles that share a name: from the tile after the previous entry's
// up to tile.
type lookEntry struct {
	tile int
	name string
}

// tileString returns the name of a tile: LOOK.LZD lists names by the last tile they
// cover. An empty name is "nothing".
// C: seg_1184:3C1B (GetTileString)
func (g *Game) tileString(tile int) string {
	if g.lookNames == nil {
		g.lookNames = readLook(g.files)
	}
	for _, e := range g.lookNames {
		if e.tile >= tile {
			if e.name == "" {
				return "nothing"
			}
			return e.name
		}
	}
	return "nothing"
}

// readLook reads LOOK.LZD: entries of a 16-bit tile number and a NUL-terminated name.
// The original reads it again at each lookup.
func readLook(files fs.FS) []lookEntry {
	packed, err := fs.ReadFile(files, "LOOK.LZD")
	if err != nil {
		return []lookEntry{}
	}
	raw, err := data.Decompress(packed)
	if err != nil {
		return []lookEntry{}
	}
	var list []lookEntry
	for at := 0; at+2 <= len(raw); {
		e := lookEntry{tile: int(binary.LittleEndian.Uint16(raw[at:]))}
		at += 2
		end := at
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		e.name = string(raw[at:end])
		at = end + 1
		list = append(list, e)
	}
	return list
}

// objectString returns the name of an object: a party member's name, what a dead or
// sleeping character was, or the name of its tile.
// C: seg_1184:3C9C (GetObjectString)
func (g *Game) objectString(n int) string {
	switch {
	case n < 0:
		return "Nothing"
	case g.isPlayerControlled(n):
		for i := range int(g.PartySize) {
			if int(g.Party[i]) == n {
				return cString(g.Names[i][:])
			}
		}
		return g.tileString(g.tileFrame(n))
	case n < npcCount && g.NPCs[n].Has(Dead|Asleep):
		shape := g.NPCs[n].OrigShape
		return g.tileString(int(g.BaseTile[shape&typeMask]) + int(shape>>typeBits))
	}
	return g.tileString(g.tileFrame(n))
}

// How objects of a type are counted (QuanType).
const (
	countSingle   = 0 // one object
	countQuantity = 2 // the low byte of Amount is the quantity
	countGold     = 4 // all of Amount is the quantity
	// maxStack is the most a stack of countQuantity objects holds.
	maxStack = 144
)

// stackedTypes are the types counted by quantity (D_0765).
var stackedTypes = []ObjectType{
	ObjTorch, ObjLockpick, ObjGem, ObjArrow, ObjBolt, ObjBlackPearl3, ObjBloodMoss3,
	ObjGarlic3, ObjGinseng3, ObjMandrake3, ObjNightshade3, ObjSpidersilk3, ObjAsh3, ObjEffect,
	ObjLoafOfBread, ObjPortionOfMeat, ObjFlaskOfOil, ObjEgg, ObjGoldNugget, ObjZuYlem, ObjSilverSnakeVenom,
}

// quantityType says how objects of type t are counted.
// C: seg_1184:21C7 (QuanType)
func quantityType(t ObjectType) int {
	if t == ObjGold {
		return countGold
	}
	for _, s := range stackedTypes {
		if s == t {
			return countQuantity
		}
	}
	return countSingle
}

// amountOf returns how many things object n is.
// C: seg_1184:21FD (C_1184_21FD)
func (g *Game) amountOf(n int) int {
	o := g.object(n)
	switch quantityType(ObjectType(o.Shape)) {
	case countGold:
		return int(o.Amount)
	case countQuantity:
		return o.Quantity()
	}
	return 1
}

// heldBy returns the objects n holds, directly or inside others, in list order.
func (g *Game) heldBy(n int) []int {
	var list []int
	for i := g.link[n]; i >= 0 && g.object(i).CoordUse() != OnMap && g.isHeldBy(i, n); i = g.link[i] {
		list = append(list, i)
	}
	return list
}

// setQuality and setQuantity change the bytes of Amount.
func (o *Object) setQuantity(v int) { o.Amount = o.Amount&^byteMask | uint16(v)&byteMask }
func (o *Object) setQuality(v int)  { o.Amount = o.Amount&byteMask | uint16(v)<<byteBits }

// giveObj gives owner amount (quality in the high byte, quantity in the low byte; all of
// it for gold) objects of type t; it returns the last object made or added to.
// C: seg_1184:2241 (GiveObj)
func (g *Game) giveObj(owner int, t ObjectType, amount int) int {
	if owner < 0 {
		return noObject
	}
	quality, count := amount>>byteBits, amount&byteMask
	how := Contained
	if owner < npcCount {
		how = InInventory
		if t == ObjAmuletOfSubmission {
			how = Equipped
			for si := g.FindInv(owner); si >= 0; si = g.NextInv() {
				if g.object(si).CoordUse() == Equipped && g.equipSlot(si) == slotNeck {
					g.object(si).SetCoordUse(InInventory)
					break
				}
			}
		}
	}
	si := noObject
	switch quantityType(t) {
	case countGold:
		if si = g.FindInvType(owner, t, -1); si < 0 {
			si = g.AddInvObj(uint16(t), how|StatusOwned, 0, owner)
		}
		if si >= 0 {
			g.object(si).Amount += uint16(amount)
		}
	case countQuantity:
		g.invCarriedOnly = true
		si = g.FindInvType(owner, t, quality)
		g.invCarriedOnly = false
		for count != 0 {
			if si < 0 {
				si = g.AddInvObj(uint16(t), how|StatusOwned, uint16(quality<<byteBits), owner)
			}
			if si < 0 {
				break
			}
			o := g.object(si)
			if o.Quantity()+count > maxStack {
				count -= maxStack - o.Quantity()
				o.setQuantity(maxStack)
				si = noObject
			} else {
				o.setQuantity(o.Quantity() + count)
				count = 0
			}
		}
	default:
		for range count {
			if si = g.AddInvObj(uint16(t), how|StatusOwned, uint16(quality<<byteBits), owner); si < 0 {
				break
			}
		}
	}
	return si
}

// takeObj takes amount objects of type t (of the quality in the high byte, unless it is
// 0) from owner.
// C: seg_1184:23F8 (TakeObj)
func (g *Game) takeObj(owner int, t ObjectType, amount int) int {
	if owner < 0 {
		return noObject
	}
	quality := amount >> byteBits
	kind := quantityType(t)
	if kind != countGold {
		amount &= byteMask
	}
	took := false
	si := g.link[owner]
	for si >= 0 && g.object(si).CoordUse() != OnMap && g.isHeldBy(si, owner) {
		o := g.object(si)
		if o.Type() == t && (kind == countGold || quality == 0 || o.Quality() == quality) {
			if have := g.amountOf(si); amount >= have {
				amount -= have
				next := g.link[si]
				g.DeleteObj(si)
				si = next
				took = true
				continue
			}
			if kind == countGold {
				o.Amount -= uint16(amount)
			} else if kind == countQuantity {
				o.setQuantity(o.Quantity() - amount)
			}
			took = true
			break
		}
		si = g.link[si]
	}
	if took && g.StatusDisplay == cmdStatusEquipment && int(g.Party[g.statusMember]) == owner {
		g.StatusDirty = true
	}
	return si
}

// countObj returns how many objects of type t n holds.
// C: seg_1184:252D (TestObj)
func (g *Game) countObj(n int, t ObjectType) int {
	total := 0
	for _, si := range g.heldBy(n) {
		if g.object(si).Type() == t {
			total += g.amountOf(si)
		}
	}
	return total
}
