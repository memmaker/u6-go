package game

// The conversation script interpreter (seg_1703 "talkdr"). A script is text to print,
// interleaved with opcodes (bytes from 128 up): statements, and expressions in reverse
// Polish notation ending with opEndOfFactor.

import (
	"encoding/binary"
	"strings"
)

// Script opcodes.
const (
	// comparisons
	opGreater      = 129
	opGreaterEqual = 130
	opLess         = 131
	opLessEqual    = 132
	opNotEqual     = 133
	opEqual        = 134
	// arithmetic and logic
	opAdd = 144
	opSub = 145
	opMul = 146
	opDiv = 147
	opOr  = 148
	opAnd = 149
	// expression functions
	opCanCarry = 154
	opWeight   = 155
	opGetHorse = 156
	opHorsed   = 157
	opRest     = 158
	opOwns     = 159
	opRandom   = 160
	// statements
	opIf          = 161
	opEndIf       = 162
	opElse        = 163
	opSet         = 164
	opClear       = 165
	opLet         = 166
	opEndOfFactor = 167
	opLetValue    = 168
	opTest        = 171
	opGoto        = 176
	opCall        = 177
	opVarInt      = 178
	opVarStr      = 179
	// opArrayItem reads a 16-bit item of an array in the script
	opArrayItem     = 180
	opPrintStr      = 181
	opLeave         = 182
	opStrSearch     = 183
	opEndSearch     = 184
	opGiveObj       = 185
	opTakeObj       = 186
	opTestObj       = 187
	opHundredBC     = 188 // pops two values and pushes 100
	opHundredBD     = 189 // the same
	opShowInventory = 190
	opShowConverse  = 191
	opSelectObject  = 192
	opOwner         = 193
	opObjType       = 194
	opValSearch     = 195
	opAddKarma      = 196
	opSubKarma      = 197
	opIsInParty     = 198
	opWhosGot       = 199
	opMoveObj       = 200
	opTransferObj   = 201
	opJoin          = 202
	opWait          = 203
	opLeaveParty    = 204
	opSetMode       = 205
	opDelay         = 208
	opAddress       = 210
	opByte          = 211
	opWord          = 212
	// opPrintNothing as the operand of opPrintStr prints nothing (OP__D5)
	opPrintNothing = 213
	opResurrect    = 214
	opIsOnScreen   = 215
	// opNameToY sets $Y to a character's name (OP__D8)
	opNameToY    = 216
	opHeal       = 217
	opWounded    = 218
	opCure       = 219
	opPoisoned   = 220
	opPartyNPC   = 221 // the party member at an index (OP__DD)
	opYouToY     = 223 // sets $Y to "you" or a character's name (OP__DF)
	opAddExp     = 224
	opAddLevel   = 225
	opAddStr     = 226
	opAddInt     = 227
	opAddDex     = 228
	opNPC        = 235 // the NPC talked to
	opEndRes     = 238
	opKey        = 239
	opFirstInput = 240 // input opcodes start here (OP__F0)
	opDesc       = 241
	opMain       = 242
	opRes        = 246
	opAskTop     = 247
	opGet        = 248
	opGetStr     = 249
	opGetChr     = 250
	opGetInt     = 251
	opGetDigit   = 252
	opFF         = 255 // the start (and the end) of a script
)

// scriptNone is what expressions give for "no object" or "not found" (0x8001).
const scriptNone = 32769

// Limits the scripts are written for.
const (
	maxStat         = 30
	maxExperience   = 9999
	maxKarma        = 99
	delayFrames     = 20
	keywordLength   = 4
	restUntilHour   = 5
	partyIndexEmpty = 255
)

// talkValue is an item of the expression stack: a number, or a string variable's text.
type talkValue struct {
	n     int32
	s     string
	isStr bool
}

// scriptByte is the byte of the script at i (0 past its end).
func (g *Game) scriptByte(i int) byte {
	if i < 0 || i >= len(g.script) {
		return 0
	}
	return g.script[i]
}

// next reads a byte (PARSE_U8).
func (g *Game) next() byte {
	b := g.scriptByte(g.pc)
	g.pc++
	return b
}

func (g *Game) scriptWord(i int) uint16 {
	return uint16(g.scriptByte(i)) | uint16(g.scriptByte(i+1))<<byteBits
}

func (g *Game) scriptLong(i int) uint32 {
	return uint32(g.scriptWord(i)) | uint32(g.scriptWord(i+2))<<(2*byteBits)
}

// npcNumber turns opNPC into the NPC talked to.
// C: seg_1703:008A (mk_npcnum)
func (g *Game) npcNumber(n int) int {
	if n == opNPC {
		return g.npc
	}
	return n
}

// keywordMatches says whether one of the words of input starts with the first n
// characters of keyword ('?' matching any character), ignoring case.
// C: seg_1703:0006 (str_i_compare)
func keywordMatches(keyword, input string, n int) bool {
	at := func(s string, i int) byte {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	for _, word := range strings.Fields(input) {
		i := 0
		for ; i < n; i++ {
			k := at(keyword, i)
			if toUpper(int(k)) != toUpper(int(at(word, i))) && k != '?' {
				break
			}
		}
		if i == n {
			return true
		}
	}
	return false
}

// parseFactor evaluates the expression at pc, up to opEndOfFactor or opLetValue.
// C: seg_1703:050E (parse_factor)
func (g *Game) parseFactor() int {
	var stack []talkValue
	// pop takes the top n values; the original read garbage when there were fewer
	pop := func(n int) []talkValue {
		args := make([]talkValue, n)
		have := min(n, len(stack))
		copy(args[n-have:], stack[len(stack)-have:])
		stack = stack[:len(stack)-have]
		return args
	}
	push := func(v int32) { stack = append(stack, talkValue{n: v}) }
	pushBool := func(b bool) {
		if b {
			push(1)
		} else {
			push(0)
		}
	}
	compareStrings := false
	for op := g.next(); op != opEndOfFactor && op != opLetValue; op = g.next() {
		switch op {
		case opAddress:
			push(int32(g.scriptLong(g.pc)))
			g.pc += 4
			continue
		case opByte:
			push(int32(g.next()))
			continue
		case opWord:
			push(int32(g.scriptWord(g.pc)))
			g.pc += 2
			continue
		}
		switch op {
		case opRandom:
			a := pop(2)
			push(int32(g.rand(int(byte(a[0].n)), int(byte(a[1].n)))))
		case opTestObj:
			a := pop(2)
			push(int32(g.countObj(g.npcNumber(int(int16(a[0].n))), ObjectType(int16(a[1].n)))))
		case opOwns:
			a := pop(3)
			pushBool(g.heldOfQuality(g.npcNumber(int(int16(a[0].n))), ObjectType(int16(a[1].n)), int(int16(a[2].n))) != noObject)
		case opWeight:
			a := pop(2)
			t, quantity := ObjectType(int16(a[0].n)), int(int16(a[1].n))
			weight := int(g.TypeWeight[t])
			light := false
			for _, l := range lightTypes {
				light = light || l == t
			}
			if light {
				push(int32(int16(quantity * weight)))
			} else {
				push(int32(int16(quantity * weight * 10)))
			}
		case opJoin:
			a := pop(1)
			push(int32(g.joinParty(g.npcNumber(int(int16(a[0].n))))))
		case opCanCarry:
			a := pop(1)
			n := g.npcNumber(int(int16(a[0].n)))
			g.weighInventory(n, true)
			const weightPerStrength, unitsPerWeight = 200, 10
			can := int(g.NPCs[n].Strength) * weightPerStrength
			carried := (g.weightEquip + g.weightInven) * unitsPerWeight
			if can > carried {
				push(int32(can - carried))
			} else {
				push(0)
			}
		case opLeaveParty:
			a := pop(1)
			push(int32(g.leaveParty(g.npcNumber(int(int16(a[0].n))))))
		case opTest:
			a := pop(2)
			n := g.npcNumber(int(int16(a[0].n)))
			push(int32(g.NPCs[n].TalkFlags>>uint(a[1].n)) & 1)
		case opSelectObject:
			g.printf("\n\nSelect item: ")
			g.MouseMode = 1
			g.StatusDisplay = cmdStatusEquipment
			g.getch()
			if g.Selection.obj != noObject && g.Selection.x != -1 {
				push(int32(g.Selection.obj))
			} else {
				push(scriptNone)
			}
		case opOwner:
			a := pop(1)
			push(int32(int16(g.object(int(int16(a[0].n))).Pos.Owner())))
			// the original rewrites the script: the NPC operand that follows becomes the
			// NPC's number
			if i := g.pc + 1; i < len(g.script) {
				g.script[i] = byte(g.npcNumber(int(g.script[i])))
			}
		case opWounded:
			a := pop(1)
			n := g.npcNumber(int(int16(a[0].n)))
			pushBool(g.maxHitPoints(n) > int(g.NPCs[n].HitPoints))
		case opPoisoned:
			a := pop(1)
			pushBool(g.NPCs[g.npcNumber(int(int16(a[0].n)))].Has(Poisoned))
		case opAddExp:
			a := pop(2)
			npc := &g.NPCs[g.npcNumber(int(int16(a[0].n)))]
			npc.Experience += uint16(a[1].n)
			npc.Experience = min(npc.Experience, maxExperience)
			push(int32(npc.Experience))
		case opAddLevel:
			a := pop(2)
			npc := &g.NPCs[g.npcNumber(int(int16(a[0].n)))]
			npc.Level += byte(a[1].n)
			push(int32(npc.Level))
		case opAddStr:
			a := pop(2)
			npc := &g.NPCs[g.npcNumber(int(int16(a[0].n)))]
			npc.Strength = min(npc.Strength+byte(a[1].n), maxStat)
			push(int32(npc.Strength))
		case opAddInt:
			a := pop(2)
			npc := &g.NPCs[g.npcNumber(int(int16(a[0].n)))]
			npc.Intelligence = min(npc.Intelligence+byte(a[1].n), maxStat)
			push(int32(npc.Intelligence))
		case opAddDex:
			a := pop(2)
			npc := &g.NPCs[g.npcNumber(int(int16(a[0].n)))]
			npc.Dexterity = min(npc.Dexterity+byte(a[1].n), maxStat)
			push(int32(npc.Dexterity))
		case opPartyNPC:
			// the index-th party member; unless anywhere is set, of those in view
			a := pop(2)
			index, anywhere := int(int16(a[0].n)), a[1].n != 0
			push(int32(g.partyNPC(index, anywhere)))
		case opIsOnScreen:
			a := pop(1)
			pushBool(g.isOnScreen(g.npcNumber(int(int16(a[0].n)))))
		case opWhosGot:
			a := pop(2)
			if m := g.partyMemberHolding(ObjectType(int16(a[0].n)), int(int16(a[1].n))); m == noObject {
				push(scriptNone)
			} else {
				push(int32(m))
			}
		case opIsInParty:
			a := pop(1)
			n := g.npcNumber(int(int16(a[0].n)))
			npc := &g.NPCs[n]
			pushBool(g.isPlayerControlled(n) && g.isOnScreen(n) && !npc.Has(Dead) && !npc.Has(Asleep) && !npc.Has(Paralyzed))
		case opHorsed:
			a := pop(1)
			pushBool(g.object(g.npcNumber(int(int16(a[0].n)))).Type() == ObjMountedHorse)
		case opObjType:
			a := pop(1)
			push(int32(g.object(int(int16(a[0].n))).Type()))
		case opHundredBC, opHundredBD:
			pop(2)
			const hundred = 100
			push(hundred)
		case opArrayItem:
			a := pop(2)
			index := int(byte(a[1].n))
			push(int32(int16(g.scriptWord(int(a[0].n) + index*2))))
		case opVarInt:
			a := pop(1)
			push(int32(uint16(g.VarInt[byte(a[0].n)])))
		case opVarStr:
			a := pop(1)
			stack = append(stack, talkValue{s: g.VarStr[byte(a[0].n)], isStr: true})
			compareStrings = true
		case opAdd:
			a := pop(2)
			push(a[0].n + a[1].n)
		case opSub:
			a := pop(2)
			push(a[0].n - a[1].n)
		case opMul:
			a := pop(2)
			push(a[0].n * a[1].n)
		case opDiv:
			a := pop(2)
			if a[1].n != 0 {
				push(int32(uint32(a[0].n) / uint32(a[1].n)))
			} else {
				g.printf("\nDivide error from talkdr.\n")
				// the original left the dividend in place
				stack = append(stack, a[0])
			}
		case opAnd:
			a := pop(2)
			pushBool(a[0].n != 0 && a[1].n != 0)
		case opOr:
			a := pop(2)
			pushBool(a[0].n != 0 || a[1].n != 0)
		case opEqual:
			a := pop(2)
			if compareStrings {
				pushBool(strings.EqualFold(a[0].s, a[1].s))
			} else {
				pushBool(a[0].n == a[1].n)
			}
		case opGreater:
			a := pop(2)
			pushBool(uint32(a[0].n) > uint32(a[1].n))
		case opGreaterEqual:
			a := pop(2)
			pushBool(uint32(a[0].n) >= uint32(a[1].n))
		case opLess:
			a := pop(2)
			pushBool(uint32(a[0].n) < uint32(a[1].n))
		case opLessEqual:
			a := pop(2)
			pushBool(uint32(a[0].n) <= uint32(a[1].n))
		case opNotEqual:
			a := pop(2)
			pushBool(a[0].n != a[1].n)
		case opStrSearch:
			a := pop(2)
			push(int32(g.searchStrings(int(a[0].n), g.VarStr[byte(a[1].n)])))
		case opValSearch:
			a := pop(2)
			push(int32(g.searchValues(int(a[0].n), g.VarInt[byte(a[1].n)])))
		default:
			// any other byte is a constant
			push(int32(op))
		}
	}
	if len(stack) > 1 {
		g.printf("Error occured.\n")
	}
	if len(stack) == 0 || stack[0].isStr {
		return 0
	}
	return int(int16(stack[0].n))
}

// partyNPC returns the index-th party member (anywhere, or among those in view), or 0.
func (g *Game) partyNPC(index int, anywhere bool) int {
	if index >= int(g.PartySize) {
		return 0
	}
	if anywhere {
		return int(g.Party[index])
	}
	var inView []int
	for i := range int(g.PartySize) {
		if g.isOnScreen(int(g.Party[i])) {
			inView = append(inView, i)
		}
	}
	// fix: a negative index read before the list
	if index < 0 || index >= len(inView) {
		return 0
	}
	return int(g.Party[inView[index]])
}

// searchStrings returns the index of the first string of the list at address (ended by
// opEndSearch) whose first (up to four) characters match a word of input, or the length
// of the list.
func (g *Game) searchStrings(address int, input string) int {
	at := address
	i := 0
	for g.scriptByte(at) != opEndSearch {
		i++
		end := at
		for g.scriptByte(end) != 0 {
			end++
		}
		s := string(g.script[at:end])
		at = end + 1
		if keywordMatches(s, input, min(len(s), keywordLength)) {
			return i - 1
		}
	}
	return i
}

// searchValues returns the index of v in the list of 16-bit values at address (ended by
// opEndSearch), or scriptNone.
func (g *Game) searchValues(address, v int) int {
	for i, at := 0, address; g.scriptByte(at) != opEndSearch; i, at = i+1, at+2 {
		if int(int16(g.scriptWord(at))) == int(int16(v)) {
			return i
		}
	}
	return scriptNone
}

// indexedAddress reads an address operand: an address followed by opCall, or by an index
// expression that picks the index-th NUL-terminated string (strings) or 16-bit item
// there. pc moves to the place addressed; it returns where to continue afterwards.
// C: seg_1703:1494 (C_1703_1494)
func (g *Game) indexedAddress(ofStrings bool) int {
	address := int(g.scriptLong(g.pc))
	g.pc += 4
	if g.next() == opCall {
		resume := g.pc
		g.pc = address
		return resume
	}
	g.pc--
	index := g.parseFactor()
	resume := g.pc
	g.pc = address
	if ofStrings {
		for index > 0 {
			if g.next() == 0 {
				index--
			}
		}
	} else {
		g.pc += index * 2
		// the operand ended with opLetValue, which the caller reads again
		resume--
	}
	return resume
}

// readString reads a NUL-terminated string at pc.
func (g *Game) readString() string {
	var s []byte
	for c := g.next(); c != 0; c = g.next() {
		s = append(s, c)
	}
	return string(s)
}

// executeOp runs statement op for the NPC n; it returns 0, 1 for a malformed statement
// and -1 for an unknown one.
// C: seg_1703:153B (execute_op)
func (g *Game) executeOp(op byte, n int) int {
	switch op {
	case opEndIf, opEndOfFactor:
	case opLeave:
		g.mustLeave = true
	case opPrintStr:
		switch kind := g.next(); kind {
		case opAddress:
			resume := g.indexedAddress(true)
			for c := g.next(); c != 0; c = g.next() {
				g.putch(int(c))
			}
			g.pc = resume
		case opPrintNothing:
		default:
			v := g.next()
			if g.next() != opVarStr {
				return 1
			}
			g.printf(g.VarStr[v])
			g.pc++
		}
	case opLet:
		v := g.next()
		const scriptWordTarget = 0
		kind := byte(scriptWordTarget)
		target := 0
		if v == opAddress {
			resume := g.indexedAddress(false)
			target = g.pc
			g.pc = resume
		} else {
			kind = g.next()
		}
		if g.next() != opLetValue {
			return 1
		}
		switch kind {
		case scriptWordTarget:
			value := g.parseFactor()
			if target+1 < len(g.script) {
				binary.LittleEndian.PutUint16(g.script[target:], uint16(value))
			}
		case opVarInt:
			g.VarInt[v] = g.parseFactor()
		default:
			if from := g.next(); from == opAddress {
				resume := g.indexedAddress(true)
				g.VarStr[v] = g.readString()
				g.pc = resume
			} else {
				if g.next() != opVarStr {
					return 1
				}
				g.VarStr[v] = g.VarStr[from]
			}
		}
	case opIf:
		v := g.parseFactor()
		if v == -1 {
			return 1
		}
		if v == 0 {
			g.skipTo(opElse, opEndIf)
		}
	case opElse:
		g.skipTo(opEndIf, opEndIf)
	case opGoto:
		g.pc = int(g.scriptLong(g.pc))
	case opGiveObj, opTakeObj:
		npc := g.npcNumber(g.parseFactor())
		t := ObjectType(g.parseFactor())
		quality := g.parseFactor()
		quantity := g.parseFactor()
		amount := quality<<byteBits | quantity
		if op == opTakeObj {
			g.takeObj(npc, t, amount)
		} else {
			g.giveObj(npc, t, amount)
		}
	case opWait:
		g.promptChar = morePrompt
		g.getch()
		g.promptChar = promptChar
	case opSetMode:
		npc := g.npcNumber(g.parseFactor())
		mode := AIMode(g.parseFactor())
		g.NPCs[npc].Mode = mode
		if mode == AISleep {
			g.NPCs[npc].Status |= Asleep
			g.setUnconscious(npc)
		}
		if mode != AIPlay && g.object(npc).Type() == ObjMusician2 {
			g.setType(npc, ObjMusician)
		}
	case opMoveObj:
		npc := g.npcNumber(g.parseFactor())
		dest := g.parseFactor()
		g.InsertObj(npc, dest, InInventory)
	case opTransferObj:
		t := ObjectType(g.parseFactor())
		quality := g.parseFactor()
		from := g.npcNumber(g.parseFactor())
		to := g.npcNumber(g.parseFactor())
		g.InsertObj(g.FindInvType(from, t, quality), to, InInventory)
	case opAddKarma:
		karma := int(byte(int(g.Karma) + g.parseFactor()))
		g.Karma = uint8(min(karma, maxKarma))
	case opSubKarma:
		v := g.parseFactor()
		if int(g.Karma) > v {
			g.Karma -= uint8(v)
		} else {
			g.Karma = 0
		}
	case opDelay:
		for range g.parseFactor() * delayFrames {
			g.OtherAnimations()
		}
	case opSet, opClear:
		npc := g.npcNumber(g.parseFactor())
		bit := byte(1) << uint(g.parseFactor())
		if op == opSet {
			g.NPCs[npc].TalkFlags |= bit
		} else {
			g.NPCs[npc].TalkFlags &^= bit
		}
	case opResurrect:
		npc := g.npcNumber(g.parseFactor())
		body := g.link[npc]
		for body >= 0 && g.object(body).Type() != ObjDeadBody {
			body = g.link[body]
		}
		g.resurrect(body)
	case opHeal:
		npc := g.npcNumber(g.parseFactor())
		g.NPCs[npc].HitPoints = byte(g.maxHitPoints(npc))
	case opCure:
		g.NPCs[g.npcNumber(g.parseFactor())].Status &^= Poisoned
	case opNameToY:
		g.VarStr[variableIndex('Y')] = g.characterName(g.npcNumber(g.parseFactor()))
	case opYouToY:
		if npc := g.npcNumber(g.parseFactor()); npc == avatar {
			g.VarStr[variableIndex('Y')] = "you"
		} else {
			g.VarStr[variableIndex('Y')] = g.characterName(npc)
		}
	case opRest:
		for i := range int(g.PartySize) {
			if g.isOnScreen(int(g.Party[i])) {
				g.heal(i, false)
			}
		}
		g.redrawWorld()
		g.allowNPCTeleport = true
		if g.Hour == restUntilHour {
			g.passTime(minutesPerHour)
		}
		for g.Hour != restUntilHour {
			g.passTime(minutesPerHour)
		}
		g.allowNPCTeleport = false
		g.screenFade = true
		g.computeView()
		g.OtherAnimations()
	case opShowInventory:
		v := g.parseFactor()
		g.showNPC = 0
		g.npcOneHanded = false
		switch {
		case v == 1:
			g.statusMember = int(g.Active)
		case g.isPlayerControlled(n):
			for i := range partyMax {
				if int(g.Party[i]) == n {
					g.statusMember = i
					break
				}
			}
		default:
			g.showNPC = n
		}
		g.equipmentView()
	case opShowConverse:
		g.showCharacter(g.npcNumber(g.parseFactor()))
	case opGetHorse:
		npc := g.npcNumber(g.parseFactor())
		g.mount(npc, int(g.MapX), int(g.MapY), directionFrame(ObjMountedHorse, g.NPCs[npc].Direction()), false)
	default:
		g.printf("Unknown command.\n")
		return -1
	}
	return 0
}

// skipTo skips statements up to one of the opcodes a or b, stepping over the operands of
// goto, byte and word.
func (g *Game) skipTo(a, b byte) {
	for op := g.next(); op != a && op != b; op = g.next() {
		switch op {
		case opGoto:
			g.pc += 4
		case opByte:
			g.pc++
		case opWord:
			g.pc += 2
		}
	}
}

// parseStatement prints text and runs statements for the NPC n up to an input opcode, the
// end of an answer or the end of the script.
// C: seg_1703:1C45 (parse_statement)
func (g *Game) parseStatement(n int) {
	const firstPrintable = ' '
	for op := g.next(); op < opFirstInput && op != 0 && op != opEndRes; op = g.next() {
		switch {
		case op < highBit && op < firstPrintable && op != '\n':
			g.printf("\nTry to print out a control char\n")
			g.pc--
			return
		case op == '~' && g.scriptByte(g.pc) == 'P':
			g.pc++
			g.speechMark(n)
		case op < highBit:
			g.putch(int(op))
		case op == opKey && !g.answered:
			g.matchKeywords(n)
		case op == opKey:
			// an answer was given: skip the others
			for op != opEndRes {
				op = g.next()
			}
		default:
			g.executeOp(op, n)
		}
		if g.mustLeave {
			break
		}
	}
	g.pc--
}

// matchKeywords reads the keywords of a KEY ... RES answer; when one matches the input
// (an empty input is "bye", the keyword '*' matches anything) the answer runs, otherwise
// pc moves to the next KEY.
// C: seg_1703:1D01 (C_1703_1D01)
func (g *Game) matchKeywords(n int) {
	if g.typed == "" {
		g.typed = "bye"
		g.VarStr[variableIndex('Z')] = g.typed
	}
	for more := true; more; {
		var keyword []byte
		c := g.next()
		for ; c != ',' && c != opRes; c = g.next() {
			keyword = append(keyword, c)
		}
		more = c == ','
		if !more {
			g.pc--
		}
		if keywordMatches(string(keyword), g.typed, len(keyword)) || len(keyword) > 0 && keyword[0] == '*' {
			for g.next() != opRes {
			}
			g.printf("\n")
			g.answered = true
			g.parseStatement(n)
			return
		}
	}
	op := g.next()
	for ; op != opKey && op != opEndRes && op != opFF; op = g.next() {
		switch op {
		case opGoto:
			g.pc += 4
		case opByte:
			g.pc++
		case opWord:
			g.pc += 2
		}
	}
	if op == opKey || op == opFF {
		g.pc--
	}
}
