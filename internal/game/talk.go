package game

// Talking to characters (seg_16E1 "talk", the driver half of seg_1703 "talkdr"): who can
// be talked to, the conversation variables, CONVERSE.A/B and the portrait shown while
// talking. The script interpreter is in talk_script.go.

import (
	"bytes"
	"encoding/binary"
	"io/fs"
	"strings"

	"github.com/memmaker/u6-go/internal/data"
)

// Conversations: CONVERSE.A holds the NPCs up to lastNPCInA, CONVERSE.B the others and
// the generic wisp, guard and gargoyle talks.
const (
	lastNPCInA           = 98  // 0x62
	wispConversation     = 102 // 0x66
	guardConversation    = 103
	gargoyleConversation = 104
	// maxPackedScript is the largest size of a script stored compressed; bigger ones are
	// stored as they are, and that many bytes are read (0x2800).
	maxPackedScript = 10240
	// npcNameLength bounds an NPC's name taken from its script (D_E764[50])
	npcNameLength = 50
	// drCatNPC is the NPC whose name "Dr_ Cat" gets its dot back (0x6e)
	drCatNPC = 110
	// pushmePullyuNPC is talked about when NPC 0 is shown (0x82)
	pushmePullyuNPC = 130
	// typedLength is the most one types at "you say:" (D_E732)
	typedLength = 50
)

// Party membership results (JoinParty, LeaveParty).
const (
	partyChangeDone       = 0
	partyChangeInVehicle  = 1
	partyChangeFullOrGone = 2
)

// alignments (GetAlignment)
const (
	alignNeutral NPCStatus = 0
	alignEvil    NPCStatus = 32
	alignGood    NPCStatus = 64
	alignChaotic NPCStatus = 96
)

// talk is the state of a conversation.
type talk struct {
	// script is the loaded conversation (TalkBuf), pc where the interpreter reads
	// (Talk_PC).
	script []byte
	pc     int
	// npc is the character talked to and speaker the party member talking (D_E796).
	npc, speaker int
	// typed is the player's last input (D_E732), keyword the keyword being matched
	// (D_E728).
	typed string
	// answered is set once a keyword answered the input (D_E79C).
	answered bool
	// descriptionPC is where the "You see" description starts (D_E79D).
	descriptionPC int
	// mustLeave ends the conversation (mustLeave).
	mustLeave bool
	// seance lets the dead talk (SeanceFlag); armageddon silences everyone (IsArmageddon).
	seance     bool
	armageddon bool
}

// initTalk sets the conversation variables for partyID talking to n, and talks.
// C: seg_16E1:0007 (TALK_initTalk)
func (g *Game) initTalk(partyID, n int) {
	member := int(g.Party[partyID])
	g.VarStr[variableIndex('G')] = "milord"
	if g.AvatarFemale != 0 {
		g.VarStr[variableIndex('G')] = "milady"
	}
	g.VarStr[variableIndex('N')] = ""
	g.VarStr[variableIndex('P')] = cString(g.Names[partyID][:])
	switch {
	case g.Hour < noon:
		g.VarStr[variableIndex('T')] = "morning"
	case g.Hour < eveningHour:
		g.VarStr[variableIndex('T')] = "afternoon"
	default:
		g.VarStr[variableIndex('T')] = "evening"
	}
	set := func(name byte, v int) { g.VarInt[variableIndex(int(name))] = int(int16(v)) }
	set('A', g.dexterityOf(member))
	set('D', int(g.Day))
	set('E', int(g.NPCs[member].Experience))
	set('G', 0)
	if g.AvatarFemale != 0 {
		set('G', 1)
	}
	set('H', int(g.Hour))
	set('I', g.intelligenceOf(member))
	set('K', int(g.Karma))
	set('L', int(g.SpeakGargish))
	set('M', int(g.Month))
	set('O', int(g.PartySize)-1)
	set('P', int(g.NPCs[member].HitPoints))
	set('S', g.strengthOf(member))
	set('W', int(g.NPCs[n].Mode))
	set('Y', int(g.Year))
	// $0-$9 point again at their buffers, which keep their text; #0-#9 are cleared
	for i := range 10 {
		g.VarInt[i] = 0
	}
	g.current.attr |= clipPaging
	g.playSong(0)
	g.talkDriver(member, n)
	g.playAreaSong()
	g.pagingDisabled = true
	g.SpeakGargish = int16(g.VarInt[variableIndex('L')])
}

// talkTo is the Talk command: partyID talks to n; withName prints who that is first.
// C: seg_16E1:015C (TALK_talkTo)
func (g *Game) talkTo(partyID, n int, withName bool) {
	// fix: talking to nothing read the type of object -1
	if n < 0 {
		g.printf("nothing!\n")
		return
	}
	t := g.object(n).Type()
	if (t == ObjHorse || t == ObjMountedHorse) && n >= npcCount {
		n = g.objectHead(n)
	}
	isStatue := t == ObjStatueOfMondain || t == ObjStatueOfMinax || t == ObjStatueOfExodus
	if n >= 0 && n < npcCount || t == ObjShrine || isStatue {
		if withName {
			switch {
			case t == ObjShrine:
				g.printf("shrine")
			case isStatue:
				g.printf("statue")
			case g.NPCs[n].TalkFlags&talkFlagMet != 0:
				g.printf(g.characterName(n))
			default:
				g.printf(g.objectString(n))
			}
		}
		g.initTalk(partyID, n)
	} else {
		g.printf("nothing!\n")
	}
}

// talkFlagMet is the talk flag set once the player learned an NPC's name.
const talkFlagMet = 1

// loadConversation returns the script of n (a shrine or statue gives its quality as n), or
// nil when there is none.
// C: seg_2FC1:1D40 (LoadConversation)
func (g *Game) loadConversation(n int, forTalk bool) []byte {
	var file string
	var index int
	switch {
	case n >= firstMonster:
		file = "CONVERSE.B"
		switch g.object(n).Type() {
		case ObjWisp:
			index = wispConversation
		case ObjGuard:
			index = guardConversation
		case ObjGargoyle:
			index = gargoyleConversation
		default:
			return nil
		}
	case n > lastNPCInA:
		file, index = "CONVERSE.B", n-lastNPCInA-1
	default:
		file, index = "CONVERSE.A", n
	}
	raw, err := fs.ReadFile(g.files, file)
	const offsetBytes = 4
	if err != nil || len(raw) < (index+1)*offsetBytes {
		return nil
	}
	at := int(binary.LittleEndian.Uint32(raw[index*offsetBytes:]))
	if at == 0 || at+offsetBytes > len(raw) {
		return nil
	}
	if forTalk {
		g.playSong(0)
	}
	if size := binary.LittleEndian.Uint32(raw[at:]); size != 0 && size < maxPackedScript {
		script, err := data.Decompress(raw[at:])
		if err != nil {
			return nil
		}
		return script
	}
	end := min(at+offsetBytes+maxPackedScript, len(raw))
	return append([]byte(nil), raw[at+offsetBytes:end]...)
}

// npcName returns the name at the start of n's script (after its opcode and number).
// C: seg_1703:00A0 (C_1703_00A0)
func (g *Game) npcName(n int) string {
	const nameStart = 2
	var name []byte
	if script := g.loadConversation(n, false); len(script) > nameStart {
		for _, c := range script[nameStart:min(len(script), nameStart+npcNameLength)] {
			if c&highBit != 0 || c == 0 {
				break
			}
			name = append(name, c)
		}
	}
	if len(name) == 0 {
		g.printf("\nUndefined.\n")
	} else if n == drCatNPC && len(name) > 2 {
		name[2] = '.'
	}
	return string(name)
}

// characterName is a party member's name or an NPC's.
// C: seg_1703:0116 (C_1703_0116)
func (g *Game) characterName(n int) string {
	if g.isPlayerControlled(n) {
		i := 0
		for i < partyMax/2 && int(g.Party[i]) != n {
			i++
		}
		// fix: the original read Names[8] for a member it did not find
		if i < partyMax/2 {
			return cString(g.Names[i][:])
		}
	}
	return g.npcName(n)
}

// tileBlack is the unseen ground of the view (TIL_0FF).
const tileBlack = 255

// isOnScreen says whether n is in view.
// C: seg_1703:0153 (C_1703_0153)
func (g *Game) isOnScreen(n int) bool {
	if g.inVehicle() && g.isPlayerControlled(n) {
		return true
	}
	p := g.object(n).Pos
	x, y := p.X-(int(g.MapX)-g.viewCenterX), p.Y-(int(g.MapY)-g.viewCenterY)
	if x <= 0 || x >= g.viewTilesX-1 || y <= 0 || y >= g.viewTilesY-1 {
		return false
	}
	if g.objectView[y][x] != 0 || g.tileView[y][x] >= firstObjectTile {
		return true
	}
	tile := g.tileView[y][x]
	return tile != tileBlack && tile != tileDarkness
}

// countNearby sets #N to the number of other party members in view.
// C: seg_1703:028B (C_1703_028B)
func (g *Game) countNearby() {
	n := 0
	for i := range int(g.PartySize) {
		if g.isOnScreen(int(g.Party[i])) {
			n++
		}
	}
	g.VarInt[variableIndex('N')] = n - 1
}

// setName writes a party member's name, cut to fit.
func (g *Game) setName(i int, name string) {
	g.Names[i] = [nameLength]byte{}
	copy(g.Names[i][:nameLength-1], name)
}

// joinParty makes n a party member.
// C: seg_1703:02B6 (JoinParty)
func (g *Game) joinParty(n int) int {
	if g.inVehicle() {
		return partyChangeInVehicle
	}
	if g.PartySize == partyMax/2 {
		return partyChangeFullOrGone
	}
	g.PartySize++
	g.Party[g.PartySize-1] = byte(n)
	g.setName(int(g.PartySize)-1, g.npcName(n))
	npc := &g.NPCs[n]
	npc.Status |= PlayerControl
	if npc.Status&alignmentMask == alignNeutral {
		npc.Status = npc.Status&^alignmentMask | alignGood
	}
	npc.Mode = AIFollow
	npc.CombatMode = AIFront
	if g.object(n).Type() == ObjMusician2 {
		g.setType(n, ObjMusician)
	}
	return partyChangeDone
}

// setType changes the type of n, keeping its frame.
func (g *Game) setType(n int, t ObjectType) {
	o := g.object(n)
	o.Shape = shapeOf(t, o.Frame())
}

// leaveParty sends n away from the party, dropping what it carries.
// C: seg_1703:0398 (LeaveParty)
func (g *Game) leaveParty(n int) int {
	if g.inVehicle() {
		return partyChangeInVehicle
	}
	at := 0
	for at < int(g.PartySize) && int(g.Party[at]) != n {
		at++
	}
	if at == int(g.PartySize) {
		return partyChangeFullOrGone
	}
	for i := at; i < int(g.PartySize); i++ {
		// fix: the original copied the member past the end of the arrays for the last one
		if i+1 < partyMax {
			g.Party[i] = g.Party[i+1]
			g.Names[i] = g.Names[i+1]
		}
	}
	g.PartySize--
	npc := &g.NPCs[n]
	npc.Status &^= PlayerControl
	npc.Mode = AIWander
	if npc.Status&alignmentMask == alignGood {
		npc.Status = npc.Status&^alignmentMask | alignNeutral
	}
	p := g.object(n).Pos
	for i := g.FindInv(n); i >= 0; i = g.FindInv(n) {
		g.MoveObj(i, p.X, p.Y, int(g.MapZ))
		g.setOkToGet(i)
	}
	g.MoveObj(n, p.X, p.Y, int(g.MapZ))
	return partyChangeDone
}

// heldOfQuality returns an object of type t (and quality, unless 0) that n holds, or -1.
// C: seg_1944:0A43 (C_1944_0A43)
func (g *Game) heldOfQuality(n int, t ObjectType, quality int) int {
	for _, i := range g.heldBy(n) {
		if o := g.object(i); o.Type() == t && (quality == 0 || o.Quality() == quality) {
			return i
		}
	}
	return noObject
}

// partyMemberHolding returns a party member in view who holds an object of type t and
// quality, or -1.
// C: seg_1703:04C6 (C_1703_04C6)
func (g *Game) partyMemberHolding(t ObjectType, quality int) int {
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		if g.isOnScreen(m) && g.heldOfQuality(m, t, quality) != noObject {
			return m
		}
	}
	return noObject
}

// Attributes as they count: a cursed character is weaker; time stop and sleep make one
// clumsy.
const cursedPenalty = 3

// strengthOf returns n's strength.
// C: seg_2337:03E7 (GetStr)
func (g *Game) strengthOf(n int) int {
	return g.cursed(n, int(g.NPCs[n].Strength))
}

// dexterityOf returns n's dexterity.
// C: seg_2337:0411 (GetDex)
func (g *Game) dexterityOf(n int) int {
	v := g.cursed(n, int(g.NPCs[n].Dexterity))
	if g.SpellFx[spellTimeStop] != 0 && !g.isPlayerControlled(n) || g.NPCs[n].Has(Asleep) {
		v = 1
	}
	return v
}

// intelligenceOf returns n's intelligence.
// C: seg_2337:0458 (GetInt)
func (g *Game) intelligenceOf(n int) int {
	return g.cursed(n, int(g.NPCs[n].Intelligence))
}

func (g *Game) cursed(n, v int) int {
	if g.object(n).Has(StatusCursed) {
		if v -= cursedPenalty; v < 1 {
			v = 1
		}
	}
	return v
}

// The portrait shown while talking.
const (
	portraitAloneX    = 216
	portraitBesideX   = 248
	portraitTalkY     = 24
	dollBesideX       = 192
	dollAloneX        = 224
	dollTalkY         = 40
	equipmentBesideDX = 0
	equipmentAloneDX  = 32
	equipmentTalkDY   = 8
)

// showCharacter shows n in the status area: its portrait (for a shrine or statue, the
// portrait of its quality), and its equipment when it wears some. Outside a conversation
// it waits for a key.
// C: seg_27A1:02D9 (C_27A1_02D9)
func (g *Game) showCharacter(n int) {
	shrine, generic, showEquipment := 0, false, false
	savedDisplay := g.StatusDisplay
	g.npcOneHanded = false
	member := 0
	if g.isPlayerControlled(n) {
		g.showNPC = 0
		for i := range partyMax {
			if int(g.Party[i]) == n {
				member = i
				g.statusMember = i
				break
			}
		}
	} else {
		g.showNPC = n
	}
	g.equipment = [slotCount]int{}
	for i := g.FindInv(n); i >= 0; i = g.NextInv() {
		if t := g.object(i).Type(); g.object(i).CoordUse() == Equipped && t != ObjEffect && t != ObjCharge {
			showEquipment = true
			g.placeEquipped(i, member, g.isPlayerControlled(n))
		}
	}
	savedMember := g.statusMember
	switch g.object(n).Type() {
	case ObjShrine, ObjStatueOfMondain, ObjStatueOfMinax, ObjStatueOfExodus:
		shrine = g.object(n).Quality()
	case ObjWisp, ObjGuard, ObjGargoyle:
		generic = true
	}
	shown := showEquipment || n < firstMonster || shrine != 0 || generic
	if shown {
		// the original clears one column less than clearStatus
		g.gr.SetColor(g.colours.background)
		g.gr.FillRect(statusX0, statusY0, statusX1-1, statusY1)
		g.setClip(clipParty)
		g.drawSky()
	}
	switch {
	case showEquipment && (n < firstMonster || generic):
		g.drawPortrait(n, portraitBesideX, portraitTalkY)
		g.drawDoll(dollBesideX, dollTalkY)
		g.drawEquipment(equipmentBesideDX, equipmentTalkDY)
		g.StatusDirty = true
	case showEquipment:
		g.drawDoll(dollAloneX, dollTalkY)
		g.drawEquipment(equipmentAloneDX, equipmentTalkDY)
		g.StatusDirty = true
	case n < firstMonster || generic:
		g.drawPortrait(n, portraitAloneX, portraitTalkY)
		g.StatusDirty = true
	case shrine != 0:
		g.drawPortrait(shrine, portraitAloneX, portraitTalkY)
		g.StatusDirty = true
	}
	g.statusMember = savedMember
	if shown {
		if n == 0 {
			n = pushmePullyuNPC
		}
		var name string
		// the original tests (!TalkFlags[n] & 1), that is TalkFlags[n] == 0
		if shrine != 0 || g.NPCs[n].TalkFlags == 0 {
			name = g.objectString(n)
		} else {
			name = g.characterName(n)
		}
		g.centreName(name, nameRow)
		g.StatusDisplay = cmdStatusContainer
		if !g.InConversation {
			g.getch()
		}
	}
	g.setClip(clipMessage)
	g.showNPC = 0
	g.npcOneHanded = false
	g.StatusDisplay = savedDisplay
}

// Results of parseNumber besides the number.
const (
	numberEmpty     = -1
	numberCancelled = -2
	numberBadSign   = 100
)

// parseNumber reads the digits of s; kind 1 and 2 read a latitude or longitude ending in
// S/N or E/W, 0 a plain number.
// C: seg_27A1:0205 (C_27A1_0205)
func (g *Game) parseNumber(s string, kind int) int {
	const latitude, longitude = 1, 2
	v := 0
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
			continue
		}
		switch kind {
		case latitude:
			switch toUpper(int(c)) {
			case 'S':
				return v
			case 'N':
				return -v
			}
			return numberBadSign
		case longitude:
			switch toUpper(int(c)) {
			case 'E':
				return v
			case 'W':
				return -v
			}
			return numberBadSign
		}
		return numberCancelled
	}
	switch {
	case kind != 0:
		return numberBadSign
	case s == "" && !g.cancelled:
		return numberEmpty
	case s == "":
		return numberCancelled
	}
	return v
}

// talkDriver runs the conversation between party member speaker and n.
// C: seg_1703:1E23 (TalkDriver)
func (g *Game) talkDriver(speaker, n int) {
	g.speaker, g.npc = speaker, n
	if g.isPlayerControlled(g.npc) && !g.isOnScreen(g.npc) {
		g.printf("\nNot on screen.\n")
		return
	}
	if g.SoloMember != -1 && g.SoloMember != 0 {
		g.printf("\nNot in solo mode.\n")
		return
	}
	t := g.object(n).Type()
	if t == ObjStatueOfMondain || t == ObjStatueOfMinax || t == ObjStatueOfExodus || t == ObjShrine {
		g.npc = g.object(n).Quality()
	}
	if (g.npc == 0 || g.npc >= firstMonster) && g.NPCs[g.npc].Has(Dead) && g.seance {
		g.printf("\nYou hear a deep moan.\n")
		return
	}
	if g.npc == n {
		npc := &g.NPCs[g.npc]
		align := npc.Status & alignmentMask
		if npc.Has(Dead) && !g.seance || npc.Has(Asleep) || npc.Has(Paralyzed) ||
			npc.Mode == AIVigilante || npc.Mode == AIFear || npc.Mode == AIRetreat || npc.Mode == AIArrest ||
			align == alignEvil || align == alignChaotic {
			g.printf("\n\nNo response\n")
			return
		}
	}
	if g.armageddon {
		g.printf("\nNo response\n")
		return
	}
	if g.speaker == g.npc {
		g.printf("\nTalking to yourself?\n")
		return
	}
	if dir := g.pointerDirection(g.PointerX, g.PointerY); dir != -1 {
		g.turnTo(int(g.Party[g.Active]), dir)
		g.computeView()
	}
	isGeneric := t == ObjGuard || t == ObjWisp || t == ObjGargoyle || t == ObjWingedGargoyle
	if g.npc == 0 || g.npc >= firstMonster && !isGeneric {
		g.printf("\nFunny, no response.\n")
		return
	}
	if g.script = g.loadConversation(g.npc, true); g.script == nil {
		g.printf("\nFunny, no response.\n")
		return
	}
	g.countNearby()
	g.pc = 0
	// the script starts with opFF and the NPC's number, then its name
	const nameStart = 2
	var name strings.Builder
	for i := nameStart; g.scriptByte(i) != opDesc; i++ {
		name.WriteByte(g.scriptByte(i))
	}
	g.VarStr[variableIndex('N')] = name.String()
	g.InConversation = true
	savedClip := g.startTalkLayout()
	defer g.endTalkLayout(savedClip)
	savedDisplay, savedMember := g.StatusDisplay, g.statusMember
	if !g.seance {
		if g.object(n).Type() != ObjShrine {
			g.showCharacter(n)
		}
		g.printf("\n\nYou see ")
		for g.next() != opDesc {
		}
		g.descriptionPC = g.pc
		g.parseStatement(g.npc)
	}
	g.printf("\n")
	for g.next() != opMain {
	}
	if g.next() != opAskTop {
		g.pc--
		g.parseStatement(g.npc)
	} else {
		g.pc--
	}
	for !g.mustLeave && g.converse() {
	}
	g.mustLeave = false
	g.InConversation = false
	if int(g.PartySize) == g.statusMember {
		g.statusMember = 0
		g.partyScroll = 0
		g.StatusDisplay = cmdStatusParty
	} else {
		g.StatusDisplay = savedDisplay
		g.statusMember = savedMember
	}
	g.StatusDirty = true
}

// converse reads the player's next input as the script asks for it and answers it; it
// returns false when the script ends or is broken.
// C: seg_1703:1E23 (TalkDriver), its loop
func (g *Game) converse() bool {
	resume := g.pc
	switch op := g.next(); op {
	case opAskTop:
		if answer := g.copyProtectionAnswer(); answer != "" {
			g.typed = answer
			g.printf("\nyou say:%s", answer)
		} else {
			g.typed = g.askTopic()
		}
		g.printf("\n")
		g.answered = false
	case opGetStr:
		g.typed = g.gets(typedLength)
		v := int(g.next())
		if g.next() != opVarStr {
			return false
		}
		g.VarStr[v] = g.typed
		g.printf("\n\n")
		g.answered = false
	case opGetInt:
		g.typed = g.gets(typedLength)
		v := int(g.next())
		if g.next() != opVarInt {
			return false
		}
		n := g.parseNumber(g.typed, 0)
		if n == numberEmpty || n == numberCancelled {
			n = 0
		}
		g.VarInt[v] = int(int16(n))
		g.printf("\n\n")
		g.answered = false
	case opGetDigit:
		n := numberEmpty
		for n == numberEmpty || n == numberCancelled {
			ch := g.getch()
			g.typed = string([]byte{byte(ch)})
			n = g.parseNumber(g.typed, 0)
			if ch == '\r' || ch == keyEscape {
				n = 0
			}
		}
		v := int(g.next())
		if g.next() != opVarInt {
			return false
		}
		g.VarInt[v] = n
		g.printf("%d\n\n", n)
		g.answered = false
	case opGetChr:
		g.typed = string([]byte{byte(g.getch())})
		g.printf("%s\n\n", g.typed)
		g.answered = false
	case opWait:
		g.getch()
	case opGet:
		// GET c0 .. cn KEY: wait for one of the keys
		var keys []byte
		for c := g.next(); c != opKey; c = g.next() {
			keys = append(keys, c)
		}
		g.pc--
		ch := 0
		for found := false; !found; {
			ch = g.getch()
			for _, k := range keys {
				found = found || toUpper(int(k)) == ch
			}
		}
		if ch >= ' ' {
			g.printf("%c", ch)
		}
		g.typed = string([]byte{byte(ch)})
		g.printf("\n")
		g.answered = false
	case opKey:
		g.printf("\nCommand for input key expected.")
		return false
	case opFF:
		return false
	case opEndRes:
	default:
		g.printf("\nLogical error occured.")
		return false
	}
	// $Z is the player's last input
	g.VarStr[variableIndex('Z')] = g.typed
	if strings.EqualFold(g.typed, "look") {
		g.printf("You see ")
		g.pc = g.descriptionPC
		g.parseStatement(g.npc)
		g.pc = resume
	} else {
		g.parseStatement(g.npc)
	}
	return true
}

// manualName is the manual the copy protection questions are about; the scripts' reply to a
// wrong answer points to it.
const manualName = "Compendium"

// copyProtectionAnswer is the right answer when the prompt at pc is one of the copy
// protection questions of Lord British, Mariah and Selganor, which take their answer from the
// manual: three answers, the right keywords, "bye" and anything else, which sends the
// player to the manual. The port answers them itself; "" for any other prompt.
func (g *Game) copyProtectionAnswer() string {
	rest := g.script[min(g.pc, len(g.script)):]
	if len(rest) == 0 || rest[0] != opKey {
		return ""
	}
	end := bytes.IndexByte(rest, opRes)
	if end < 0 {
		return ""
	}
	keywords := string(rest[1:end])
	rest = rest[end:]
	bye := bytes.Index(rest, []byte{opKey, 'b', 'y', 'e', opRes})
	other := bytes.Index(rest, []byte{opKey, '*', opRes})
	if bye < 0 || other < bye || bytes.Count(rest[:other], []byte{opKey}) != 1 {
		return ""
	}
	reply := rest[other:]
	if next := bytes.IndexByte(reply[1:], opKey); next >= 0 {
		reply = reply[:next+1]
	}
	if !bytes.Contains(reply, []byte(manualName)) {
		return ""
	}
	answer, _, _ := strings.Cut(keywords, ",")
	return answer
}
