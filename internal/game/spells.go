package game

// The spells (seg_1944): their tables, the cast command and what each spell does.

import "github.com/memmaker/u6-go/internal/sound"

// Spell numbers (SPELL_xx): the circle (from 0) times circleSlots plus the spell's place in
// its circle.
const (
	castCreateFood = iota + 0*circleSlots
	castDetectMagic
	castDetectTrap
	castDispelMagic
	castDouse
	castHarm
	castHeal
	castHelp
	castIgnite
	castLight
)

const (
	castInfravision = iota + 1*circleSlots
	castMagicArrow
	castPoison
	castReappear
	castSleep
	castTelekinesis
	castTrap
	castUnlockMagic
	castUntrap
	castVanish
)

const (
	castCurse = iota + 2*circleSlots
	castDispelField
	castFireball
	castGreatLight
	castLock
	castMassAwaken
	castMassSleep
	castPeer
	castProtection
	castRepelUndead
)

const (
	castAnimate = iota + 3*circleSlots
	castConjure
	castDisable
	castFireField
	castGreatHeal
	castLocate
	castMassDispel
	castPoisonField
	castSleepField
	castWindChange
)

const (
	castEnergyField = iota + 4*circleSlots
	castExplosion
	castInsectSwarm
	castInvisibility
	castLightning
	castParalyze
	castPickpocket
	castReveal
	castSeance
	castXray
)

const (
	castCharm = iota + 5*circleSlots
	castClone
	castConfuse
	castFlameWind
	castHailStorm
	castMassProtect
	castNegateMagic
	castPoisonWind
	castReplicate
	castWeb
)

const (
	castChainBolt = iota + 6*circleSlots
	castEnchant
	castEnergyWind
	castFear
	castGateTravel
	castKill
	castMassCurse
	castMassInvisibility
	castWingStrike
	castWizardEye
)

const (
	castArmageddon = iota + 7*circleSlots
	castDeathWind
	castEclipse
	castMassCharm
	castMassKill
	castResurrect
	castSlime
	castSummon
	castTimeStop
	castTremor
)

const (
	// castSelfInvisibility makes the caster invisible (SPELL_80); no spellbook holds it.
	castSelfInvisibility = circleCount * circleSlots
	// castWispTeleport is the wisp's teleport in combat (SPELL_81); it is never cast.
	castWispTeleport = castSelfInvisibility + 1

	circleCount   = 8
	circleSlots   = 16 // spell numbers per circle
	circleSpells  = 10 // spells per circle
	spellCount    = circleCount * circleSlots
	allSpellsMask = 1<<circleSpells - 1
	// spellLetters is how many letters an incantation may have.
	spellLetters = 4
)

// spellCodes are the first letters of each spell's words of power (SpellCode).
var spellCodes = [spellCount]string{
	castCreateFood:       "IMY",
	castDetectMagic:      "WO",
	castDetectTrap:       "WJ",
	castDispelMagic:      "AJO",
	castDouse:            "AF",
	castHarm:             "AM",
	castHeal:             "IM",
	castHelp:             "KL",
	castIgnite:           "IF",
	castLight:            "IL",
	castInfravision:      "QL",
	castMagicArrow:       "OJ",
	castPoison:           "INP",
	castReappear:         "IY",
	castSleep:            "IZ",
	castTelekinesis:      "OPY",
	castTrap:             "IJ",
	castUnlockMagic:      "EP",
	castUntrap:           "AJ",
	castVanish:           "AY",
	castCurse:            "AS",
	castDispelField:      "AG",
	castFireball:         "PF",
	castGreatLight:       "VL",
	castLock:             "AP",
	castMassAwaken:       "AVZ",
	castMassSleep:        "VZ",
	castPeer:             "VWY",
	castProtection:       "IS",
	castRepelUndead:      "AXC",
	castAnimate:          "OY",
	castConjure:          "KX",
	castDisable:          "AVM",
	castFireField:        "IFG",
	castGreatHeal:        "VM",
	castLocate:           "IW",
	castMassDispel:       "VAJO",
	castPoisonField:      "ING",
	castSleepField:       "IZG",
	castWindChange:       "RH",
	castEnergyField:      "ISG",
	castExplosion:        "VPF",
	castInsectSwarm:      "KBX",
	castInvisibility:     "SL",
	castLightning:        "OG",
	castParalyze:         "AXP",
	castPickpocket:       "PY",
	castReveal:           "ASL",
	castSeance:           "KMC",
	castXray:             "WY",
	castCharm:            "AXE",
	castClone:            "IQX",
	castConfuse:          "VQ",
	castFlameWind:        "FH",
	castHailStorm:        "KDY",
	castMassProtect:      "VIS",
	castNegateMagic:      "AO",
	castPoisonWind:       "NH",
	castReplicate:        "IQY",
	castWeb:              "IDP",
	castChainBolt:        "VOG",
	castEnchant:          "IOY",
	castEnergyWind:       "GH",
	castFear:             "QC",
	castGateTravel:       "VRP",
	castKill:             "IC",
	castMassCurse:        "VAS",
	castMassInvisibility: "VSL",
	castWingStrike:       "KOX",
	castWizardEye:        "POW",
	castArmageddon:       "VCBM",
	castDeathWind:        "CH",
	castEclipse:          "VAL",
	castMassCharm:        "VAXE",
	castMassKill:         "VC",
	castResurrect:        "IMC",
	castSlime:            "VRX",
	castSummon:           "KXC",
	castTimeStop:         "AT",
	castTremor:           "VPY",
}

// powerWords are the words of power, by first letter (D_0BF0).
var powerWords = [26]string{"An", "Bet", "Corp", "Des", "Ex", "Flam", "Grav", "Hur", "In", "Jux", "Kal",
	"Lor", "Mani", "Nox", "Ort", "Por", "Quas", "Rel", "Sanct", "Tym", "Uus", "Vas", "Wis", "Xen",
	"Ylem", "Zu"}

// spellNames are the names of the spells (SpellName).
var spellNames = [spellCount]string{
	castCreateFood:       "Create Food",
	castDetectMagic:      "Detect Magic",
	castDetectTrap:       "Detect Trap",
	castDispelMagic:      "Dispel Magic",
	castDouse:            "Douse",
	castHarm:             "Harm",
	castHeal:             "Heal",
	castHelp:             "Help",
	castIgnite:           "Ignite",
	castLight:            "Light",
	castInfravision:      "Infravision",
	castMagicArrow:       "Magic Arrow",
	castPoison:           "Poison",
	castReappear:         "Reappear",
	castSleep:            "Sleep",
	castTelekinesis:      "Telekinesis",
	castTrap:             "Trap",
	castUnlockMagic:      "Unlock Magic",
	castUntrap:           "Untrap",
	castVanish:           "Vanish",
	castCurse:            "Curse",
	castDispelField:      "Dispel Field",
	castFireball:         "Fireball",
	castGreatLight:       "Great Light",
	castLock:             "Lock",
	castMassAwaken:       "Mass Awaken",
	castMassSleep:        "Mass Sleep",
	castPeer:             "Peer",
	castProtection:       "Protection",
	castRepelUndead:      "Repel Undead",
	castAnimate:          "Animate",
	castConjure:          "Conjure",
	castDisable:          "Disable",
	castFireField:        "Fire Field",
	castGreatHeal:        "Great Heal",
	castLocate:           "Locate",
	castMassDispel:       "Mass Dispel",
	castPoisonField:      "Poison Field",
	castSleepField:       "Sleep Field",
	castWindChange:       "Wind Change",
	castEnergyField:      "Energy Field",
	castExplosion:        "Explosion",
	castInsectSwarm:      "Insect Swarm",
	castInvisibility:     "Invisibility",
	castLightning:        "Lightning",
	castParalyze:         "Paralyze",
	castPickpocket:       "Pickpocket",
	castReveal:           "Reveal",
	castSeance:           "Seance",
	castXray:             "X-ray",
	castCharm:            "Charm",
	castClone:            "Clone",
	castConfuse:          "Confuse",
	castFlameWind:        "Flame Wind",
	castHailStorm:        "Hail Storm",
	castMassProtect:      "Mass Protect",
	castNegateMagic:      "Negate Magic",
	castPoisonWind:       "Poison Wind",
	castReplicate:        "Replicate",
	castWeb:              "Web",
	castChainBolt:        "Chain Bolt",
	castEnchant:          "Enchant",
	castEnergyWind:       "Energy Wind",
	castFear:             "Fear",
	castGateTravel:       "Gate Travel",
	castKill:             "Kill",
	castMassCurse:        "Mass Curse",
	castMassInvisibility: "Mass Invis",
	castWingStrike:       "Wing Strike",
	castWizardEye:        "Wizard Eye",
	castArmageddon:       "Armageddon",
	castDeathWind:        "Death Wind",
	castEclipse:          "Eclipse",
	castMassCharm:        "Mass Charm",
	castMassKill:         "Mass Kill",
	castResurrect:        "Resurrect",
	castSlime:            "Slime",
	castSummon:           "Summon",
	castTimeStop:         "Time Stop",
	castTremor:           "Tremor",
}

// Reagent bits (REAGENT_xx), in the order of reagentNames and reagentTypes.
const (
	reagentMandrake = 1 << iota
	reagentNightshade
	reagentBlackPearl
	reagentBloodMoss
	reagentSpiderSilk
	reagentGarlic
	reagentGinseng
	reagentAsh

	noReagents   = 0
	reagentKinds = 8
)

// reagentNames and reagentTypes are the reagents (Reagents_name, ReagType).
var (
	reagentNames = [reagentKinds]string{"mandrake root", "nightshade", "black pearl", "blood moss",
		"spider silk", "garlic", "ginseng", "sulfurous ash"}
	reagentTypes = [reagentKinds]ObjectType{ObjMandrake3, ObjNightshade3, ObjBlackPearl3, ObjBloodMoss3,
		ObjSpidersilk3, ObjGarlic3, ObjGinseng3, ObjAsh3}
)

// reagentsNeeded are the reagents of each spell (Reagents_needed).
var reagentsNeeded = [spellCount]byte{
	castCreateFood:       reagentGinseng | reagentGarlic | reagentMandrake,
	castDetectMagic:      reagentAsh | reagentNightshade,
	castDetectTrap:       reagentAsh | reagentNightshade,
	castDispelMagic:      reagentGinseng | reagentGarlic,
	castDouse:            reagentGarlic | reagentBlackPearl,
	castHarm:             reagentSpiderSilk | reagentNightshade,
	castHeal:             reagentGinseng | reagentSpiderSilk,
	castHelp:             noReagents,
	castIgnite:           reagentAsh | reagentBlackPearl,
	castLight:            reagentAsh,
	castInfravision:      reagentAsh | reagentNightshade,
	castMagicArrow:       reagentAsh | reagentBlackPearl,
	castPoison:           reagentBloodMoss | reagentBlackPearl | reagentNightshade,
	castReappear:         reagentSpiderSilk | reagentBloodMoss | reagentBlackPearl,
	castSleep:            reagentSpiderSilk | reagentBlackPearl | reagentNightshade,
	castTelekinesis:      reagentBloodMoss | reagentBlackPearl | reagentMandrake,
	castTrap:             reagentSpiderSilk | reagentNightshade,
	castUnlockMagic:      reagentAsh | reagentBloodMoss,
	castUntrap:           reagentAsh | reagentBloodMoss,
	castVanish:           reagentGarlic | reagentBloodMoss | reagentBlackPearl,
	castCurse:            reagentAsh | reagentGarlic | reagentNightshade,
	castDispelField:      reagentAsh | reagentBlackPearl,
	castFireball:         reagentAsh | reagentBlackPearl,
	castGreatLight:       reagentAsh | reagentMandrake,
	castLock:             reagentAsh | reagentGarlic | reagentBloodMoss,
	castMassAwaken:       reagentGinseng | reagentGarlic,
	castMassSleep:        reagentGinseng | reagentSpiderSilk | reagentNightshade,
	castPeer:             reagentNightshade | reagentMandrake,
	castProtection:       reagentAsh | reagentGinseng | reagentGarlic,
	castRepelUndead:      reagentAsh | reagentGarlic,
	castAnimate:          reagentAsh | reagentBloodMoss | reagentMandrake,
	castConjure:          reagentSpiderSilk | reagentMandrake,
	castDisable:          reagentSpiderSilk | reagentNightshade | reagentMandrake,
	castFireField:        reagentAsh | reagentSpiderSilk | reagentBlackPearl,
	castGreatHeal:        reagentGinseng | reagentSpiderSilk | reagentMandrake,
	castLocate:           reagentNightshade,
	castMassDispel:       reagentSpiderSilk | reagentBlackPearl | reagentNightshade,
	castPoisonField:      reagentGinseng | reagentSpiderSilk | reagentBlackPearl,
	castSleepField:       reagentGinseng | reagentGarlic,
	castWindChange:       reagentAsh | reagentBloodMoss,
	castEnergyField:      reagentSpiderSilk | reagentBlackPearl | reagentMandrake,
	castExplosion:        reagentAsh | reagentBloodMoss | reagentBlackPearl | reagentMandrake,
	castInsectSwarm:      reagentAsh | reagentSpiderSilk | reagentBloodMoss,
	castInvisibility:     reagentBloodMoss | reagentNightshade,
	castLightning:        reagentAsh | reagentBlackPearl | reagentMandrake,
	castParalyze:         reagentAsh | reagentSpiderSilk | reagentBlackPearl | reagentNightshade,
	castPickpocket:       reagentSpiderSilk | reagentBloodMoss | reagentNightshade,
	castReveal:           reagentSpiderSilk | reagentNightshade | reagentMandrake,
	castSeance:           reagentAsh | reagentSpiderSilk | reagentBloodMoss | reagentNightshade | reagentMandrake,
	castXray:             reagentAsh | reagentMandrake,
	castCharm:            reagentSpiderSilk | reagentBlackPearl | reagentNightshade,
	castClone:            reagentAsh | reagentGinseng | reagentSpiderSilk | reagentBloodMoss | reagentNightshade | reagentMandrake,
	castConfuse:          reagentNightshade | reagentMandrake,
	castFlameWind:        reagentAsh | reagentBloodMoss | reagentMandrake,
	castHailStorm:        reagentBloodMoss | reagentBlackPearl | reagentMandrake,
	castMassProtect:      reagentAsh | reagentGinseng | reagentGarlic | reagentMandrake,
	castNegateMagic:      reagentAsh | reagentGarlic | reagentMandrake,
	castPoisonWind:       reagentAsh | reagentBloodMoss | reagentNightshade,
	castReplicate:        reagentAsh | reagentGinseng | reagentSpiderSilk | reagentBloodMoss | reagentNightshade,
	castWeb:              reagentSpiderSilk,
	castChainBolt:        reagentAsh | reagentBloodMoss | reagentBlackPearl | reagentMandrake,
	castEnchant:          reagentAsh | reagentSpiderSilk | reagentMandrake,
	castEnergyWind:       reagentAsh | reagentBloodMoss | reagentNightshade | reagentMandrake,
	castFear:             reagentGarlic | reagentNightshade | reagentMandrake,
	castGateTravel:       reagentAsh | reagentBlackPearl | reagentMandrake,
	castKill:             reagentAsh | reagentBlackPearl | reagentNightshade,
	castMassCurse:        reagentAsh | reagentGarlic | reagentNightshade | reagentMandrake,
	castMassInvisibility: reagentBloodMoss | reagentBlackPearl | reagentNightshade | reagentMandrake,
	castWingStrike:       reagentAsh | reagentSpiderSilk | reagentBloodMoss | reagentMandrake,
	castWizardEye:        reagentAsh | reagentSpiderSilk | reagentBloodMoss | reagentBlackPearl | reagentNightshade | reagentMandrake,
	castArmageddon:       noReagents,
	castDeathWind:        reagentAsh | reagentBloodMoss | reagentNightshade | reagentMandrake,
	castEclipse:          reagentAsh | reagentGarlic | reagentBloodMoss | reagentNightshade | reagentMandrake,
	castMassCharm:        reagentSpiderSilk | reagentBlackPearl | reagentNightshade | reagentMandrake,
	castMassKill:         reagentAsh | reagentBlackPearl | reagentNightshade | reagentMandrake,
	castResurrect:        reagentAsh | reagentGinseng | reagentGarlic | reagentSpiderSilk | reagentBloodMoss | reagentMandrake,
	castSlime:            reagentBloodMoss | reagentNightshade | reagentMandrake,
	castSummon:           reagentGarlic | reagentSpiderSilk | reagentBloodMoss | reagentMandrake,
	castTimeStop:         reagentGarlic | reagentBloodMoss | reagentMandrake,
	castTremor:           reagentAsh | reagentBloodMoss | reagentMandrake,
}

// Messages of the spells (D_0DDC, NotInBookMsg).
const (
	successMsg   = "Success"
	onWhatMsg    = "On what: "
	onWhomMsg    = "On whom: "
	locationMsg  = "Location: "
	notInBookMsg = "\nThat spell is not in thy spellbook!\n"
)

// What a spell's effect did (SpellResult).
const (
	resultSuccess  = 0
	resultFailed   = 1
	resultNoEffect = 2
	resultShown    = -1 // the effect printed what happened
	resultHit      = -2 // the hit on the target showed what happened
)

// Kinds of targets of the aimed spells.
const (
	onCharacter = 1
	onObject    = -1
)

// Missile tiles of the spells.
const (
	tilePoisonBolt  = 380 // TIL_17C
	tileSleepBolt   = 381 // TIL_17D
	tileFireball    = 382 // TIL_17E
	tileMagicBolt   = 383 // TIL_17F
	tileCharmBolt   = 394 // TIL_18A
	tileDispelBolt  = 395 // TIL_18B
	tileDeathBolt   = 396 // TIL_18C
	tileFlameBolt   = 397 // TIL_18D
	tileMagicArrow  = 566 // TIL_236
	spellRange      = 8
	nearbyRange     = 5 // the area the spells around the caster reach
	widerRange      = 7
	spellFoodMax    = 10
	attemptsToPlace = 8 // tries at a free place for what a spell makes
	placeSpread     = 10
)

// magic is the state of the spellbook and of the spell being cast.
type magic struct {
	// spellList are the spells of the spellbooks, a bit per spell of each circle
	// (SpellList).
	spellList [circleCount]uint16
	// spellbooks are the readied spellbooks (Spellbook), noObject for none.
	spellbooks [2]int
	// selectedSpell is the spell the arrow points at (D_EBB3).
	selectedSpell int
	// pageStart counts the spells before the page shown (D_EBB1).
	pageStart int
	// corners are the corners of the page (D_EBAD): the circle the left corner turns to,
	// what the left and right corners turn to (another circle or another page of the same
	// circle) and the circle the right corner turns to; noCorner where they are blank.
	corners [4]int
	// pageSpells are the spells of the page (D_EBB6), pageCounts how often each can be
	// cast with the reagents held (D_EBBE), pageMarked the row of the arrow (bit 7 of
	// D_EBBE) and pageLength how many there are (D_EBAA).
	pageSpells [8]int
	pageCounts [8]int
	pageMarked [8]bool
	pageLength int
	// savedStatus is the status display under the spellbook (D_EBAB).
	savedStatus int
	// the incantation typed: its letters (D_B6DB), where each was printed (D_B6A7, D_B6AB)
	// and how many there are (D_04B5)
	letters          [spellLetters]byte
	letterX, letterY [spellLetters]int
	letterCount      int
	// pageTurned makes the arrow go to the top of the page turned with the keys (D_0496).
	pageTurned bool
	// spellResult is what the effect did (SpellResult).
	spellResult int
	// npcCaster is the monster casting in combat (Party[16], past the party).
	npcCaster int
}

// circleOf is the circle of spell (MK_CIRCLE).
func circleOf(spell int) int { return spell/circleSlots + 1 }

// hasSpell says whether the spellbooks hold spell index of circle (HAS_SPELL).
func (m *magic) hasSpell(circle, index int) bool { return m.spellList[circle]&(1<<index) != 0 }

// incantation is the words of power of spell.
func incantation(spell int) string {
	s := ""
	for i, c := range spellCodes[spell] {
		if i > 0 {
			s += " "
		}
		s += powerWords[c-'A']
	}
	return s
}

// caster is the party member who casts (Party[Active]), or the monster casting in combat
// when Active is partyMax.
func (g *Game) caster() int {
	if int(g.Active) == partyMax {
		return g.npcCaster
	}
	return int(g.Party[g.Active])
}

// spellFailed prints why a spell failed and sounds the failure.
func (g *Game) spellFailed(msg string) {
	g.printf("\n%s\n", msg)
	g.sfx(sound.EffectFailure, 0)
}

// spellSucceeded prints Success and sounds it.
func (g *Game) spellSucceeded() {
	g.printf("\n%s\n", successMsg)
	g.sfx(sound.EffectSuccess, 0)
}

// flashNotes are the notes of the flash of each circle: the step of the note, the start
// width of the first and of the second waved note and how fast it changes (D_0E92, D_0EA4,
// D_0EB6, D_0EC8).
var flashNotes = [circleCount + 1]struct{ step, width, widthBack, delta int }{
	{8810, 2700, 32700, 3}, {7830, 3000, 31000, 2}, {7060, 1000, 37000, 2}, {6550, 100, 45000, 2},
	{5950, 5000, 31000, 1}, {5570, 4000, 34000, 1}, {5180, 2500, 36500, 1}, {4820, 1000, 39000, 1},
	{4480, 1, 42000, 1},
}

// spellFlash flashes the view and sounds the magic of circle.
// C: seg_1944:0085 (C_1944_0085)
func (g *Game) spellFlash(circle int) {
	if circle >= len(flashNotes) {
		const step, steps, width = 4480, 65000, 300
		speaker(sound.PlayWavedNote(step, 1, steps, width, 1))
		return
	}
	const noiseStep, noiseLength, noiseHz = 800, 8000, 700
	speaker(sound.PlayNoise(noiseStep, circle*1600+noiseLength, noiseHz))
	const white = 15
	g.gr.SetColor(white)
	g.gr.Offscreen, g.gr.XorMode = true, true
	g.drawMap(false)
	g.invertView()
	n := flashNotes[circle]
	const baseSteps = 10000
	speaker(sound.PlayWavedNote(n.step, 1, circle*4000+baseSteps, n.width, n.delta))
	speaker(sound.PlayWavedNote(n.step, 1, circle*4000+baseSteps, n.widthBack, -n.delta))
	g.invertView()
	g.gr.XorMode, g.gr.Offscreen = false, false
}

// invertView flips the colours of the view on the back buffer and shows it.
func (g *Game) invertView() {
	const first, last = viewPixelStart, viewPixelEnd
	g.gr.FillRect(first, first, last, last)
	g.redrawFrame(first, first, last-1, last-1)
	g.gr.CopyBackBufferToScreen(first, first, last-1, last-1, first, first)
}

// strengthResists says whether b resists a's strength.
// C: seg_1944:01EC (C_1944_01EC)
func (g *Game) strengthResists(a, b int) bool {
	return (g.strengthOf(b)/2+30-g.strengthOf(a))/2 > g.rand(1, 30)
}

// resistsByStrength says whether b's strength resists a's intelligence.
// C: seg_1944:0232 (C_1944_0232)
func (g *Game) resistsByStrength(a, b int) bool {
	return (g.strengthOf(b)/2+30-g.intelligenceOf(a))/2 > g.rand(1, 30)
}

// dodges says whether b's dexterity dodges a's spell.
// C: seg_1944:0278 (C_1944_0278)
func (g *Game) dodges(a, b int) bool {
	return (g.dexterityOf(b)/2+30-g.intelligenceOf(a))/2 > g.rand(1, 30)
}

// magicTiles are the tiles of the magical objects (D_0DF6).
var magicTiles = map[int]bool{519: true, 527: true, 534: true, 559: true, 562: true, 565: true, 568: true,
	569: true, 573: true, 584: true, 585: true, 586: true, 587: true, 588: true, 589: true, 590: true,
	591: true, 593: true, 594: true, 595: true, 597: true, 598: true, 599: true, 602: true, 603: true,
	604: true, 605: true, 607: true, 622: true, 694: true, 798: true, 799: true, 806: true, 872: true,
	873: true, 874: true, 875: true, 876: true, 877: true, 878: true, 879: true, 893: true, 894: true,
	895: true, 936: true, 937: true, 938: true, 939: true, 940: true, 941: true, 942: true, 943: true,
	1021: true, 1022: true, 1036: true, 1037: true, 1038: true, 1039: true, 1052: true, 1053: true,
	1054: true, 1055: true, 1068: true, 1069: true, 1070: true, 1071: true, 1084: true, 1085: true,
	1086: true, 1087: true, 1156: true, 1157: true, 1166: true, 1167: true, 1744: true, 1745: true,
	1746: true, 1747: true}

// isMagical says whether object n is magical.
// C: seg_1944:0006 (IsMagical)
func (g *Game) isMagical(n int) bool { return magicTiles[g.tileFrame(n)] }

// canTurnInvisible says whether a creature of type t may turn invisible.
// C: seg_1944:0054 (C_1944_0054)
func canTurnInvisible(t ObjectType) bool {
	return t < ObjGiantScorpion && t != ObjDragon && t != ObjHydra && t != ObjSilverSerpent && t != ObjSlime
}

// reagentCount counts the reagents of type t (and quality) that n carries.
// C: seg_1944:02BE (C_1944_02BE)
func (g *Game) reagentCount(n int, t ObjectType, quality int) int {
	count := 0
	for i := g.link[n]; i >= 0 && g.object(i).CoordUse() != OnMap; i = g.link[i] {
		if o := g.object(i); o.Type() == t && o.Quality() == quality {
			count += o.Quantity()
		}
	}
	return count
}

// takeReagents uses up the reagents of spell that n carries; row is the spell's row on the
// page.
// C: seg_1944:06EF (TakeReagents)
func (g *Game) takeReagents(n, spell, row int) {
	need := reagentsNeeded[spell]
	for r := range reagentKinds {
		if need>>r&1 == 0 {
			continue
		}
		for i := g.link[n]; i >= 0; i = g.link[i] {
			if t := g.object(i).Type(); t == reagentTypes[r] {
				g.takeObj(n, t, 1)
				break
			}
		}
	}
	// fix: the original counted down the row of Selection.obj, the target when the spell
	// was aimed at once
	if spell != castHelp && row >= 0 {
		g.pageCounts[row]--
	}
}

// heldSpellbooks returns the spellbooks the active member has readied.
func (g *Game) heldSpellbooks() (first, second int) {
	first, second = noObject, noObject
	for i := g.FindInv(g.caster()); i >= 0; i = g.NextInv() {
		if o := g.object(i); o.CoordUse() == Equipped && o.Type() == ObjSpellbook {
			if first == noObject {
				first = i
			} else {
				second = i
			}
		}
	}
	return first, second
}

// cast is the cast command: it casts a spell of the readied spellbooks.
// C: seg_1944:4C2F (C_1944_4C2F)
func (g *Game) cast() {
	first, second := g.heldSpellbooks()
	if first == noObject {
		g.printf("No spellbook is readied.\n")
		return
	}
	g.castFrom(first, second)
}

// castFrom lets the player pick a spell of the spellbooks and casts it.
// C: seg_1944:4926 (C_1944_4926)
func (g *Game) castFrom(first, second int) {
	g.spellbooks = [2]int{first, second}
	if g.SpellFx[spellNegateMagic] != 0 {
		g.printf("No magic at this time!\n")
		return
	}
	n := g.caster()
	g.refreshSpellList()
	if g.spellList == [circleCount]uint16{} {
		g.printf("No spells in the spellbook!\n")
		return
	}
	g.drawSpellbook()
	g.refreshPage()
	g.savedStatus = g.StatusDisplay
	g.StatusDisplay = cmdStatusSpellbook
	ch := g.getch()
	sel := &g.Selection
	if (ch != cmdSelected || sel.obj == noObject) && sel.x == -1 {
		g.printf("nothing\n")
		g.StatusDisplay = g.savedStatus
		g.StatusDirty = true
		return
	}
	targeted := false
	var spell int
	if sel.x != -1 && sel.y != -1 {
		spell = g.selectedSpell
		targeted = true
	} else {
		spell = g.pageSpells[sel.obj]
	}
	if g.SelectMode != 0 && g.letterCount != 0 {
		g.printf("(%s)", spellNames[spell])
		g.letterCount = 0
	} else {
		g.printf("%s\n\"%s\"", spellNames[spell], incantation(spell))
	}
	circle := circleOf(spell)
	if !targeted {
		g.setClip(clipParty)
		for i, marked := range g.pageMarked {
			if marked {
				g.gotoxy(0, i+1)
				g.putch0(' ')
			}
		}
		g.gotoxy(0, sel.obj+1)
		g.putch0(arrowChar)
		g.setClip(clipMessage)
		g.selectSpell(spell)
	}
	g.StatusDisplay = g.savedStatus
	g.StatusDirty = true
	g.RefreshStatus()
	row := g.pageRow(spell)
	switch {
	case int(g.NPCs[n].Level) < circle:
		g.sfx(sound.EffectFailure, 0)
		g.printf("\n\nYour level is not high enough.\n")
	case int(g.NPCs[n].Magic) < circle:
		g.sfx(sound.EffectFailure, 0)
		g.printf("\n\nNot enough magic points.\n")
	// fix: the original read before the counts when the spell was not on the page
	case row < 0 || g.pageCounts[row] == 0:
		g.sfx(sound.EffectFailure, 0)
		g.printf("\n\nNo Reagents.\n")
	default:
		g.takeReagents(n, spell, row)
		g.NPCs[n].Magic -= byte(circle)
		const baseMoves, movesPerCircle = 10, 3
		g.subMove(n, circle*movesPerCircle+baseMoves)
		g.castSpell(spell, targeted)
		g.StatusDisplay = g.savedStatus
		g.StatusDirty = true
		g.RefreshStatus()
	}
}

// castSpell does what spell does; targeted says the target is already in Selection.
// C: seg_1944:43D5 (C_1944_43D5)
func (g *Game) castSpell(spell int, targeted bool) {
	g.printf("\n")
	caster := g.caster()
	// the spells that only change the world around the caster
	ambient := func(fx, minutes int) {
		g.magicalHalo(caster, 0)
		g.spellFlash(circleOf(spell))
		g.SpellFx[fx] = byte(minutes)
	}
	switch spell {
	case castDouse:
		g.aimSpell(spell, tileDispelBolt, spellRange, onObject, targeted)
	case castVanish, castDispelField, castAnimate, castReplicate:
		g.aimSpell(spell, tileMagicBolt, spellRange, onObject, targeted)
	case castIgnite:
		g.aimSpell(spell, tileFlameBolt, spellRange, onObject, targeted)
	case castDisable:
		g.touchSpell(spell, spellRange, onCharacter, targeted)
	case castResurrect:
		g.resurrectSpell(targeted)
	case castLight:
		const lightMinutes = 100
		ambient(spellLight, lightMinutes)
		g.passTime(0)
	case castHelp:
		g.spellFlash(circleOf(spell))
		g.resurrectParty()
	case castDetectMagic, castDetectTrap, castTrap, castUnlockMagic, castUntrap, castLock:
		g.touchSpell(spell, spellRange, onObject, targeted)
	case castSlime:
		g.slime()
	case castCreateFood:
		g.createFood(spell)
	case castReappear:
		g.reappear(targeted)
	case castSleep:
		g.aimSpell(spell, tileSleepBolt, spellRange, onCharacter, targeted)
	case castMagicArrow:
		g.aimSpell(spell, tileMagicArrow, spellRange, onCharacter, targeted)
	case castFireball:
		g.aimSpell(spell, tileFireball, spellRange, onCharacter, targeted)
	case castLightning, castChainBolt:
		g.aimSpell(spell, tileLightningBolt, spellRange, onCharacter, targeted)
	case castTelekinesis:
		g.telekinesis(targeted)
	case castInfravision:
		const infravisionMinutes = 20
		ambient(spellInfravision, infravisionMinutes)
	case castMassAwaken, castMassDispel, castMassProtect, castMassCurse:
		g.areaSpell(spell, tileDispelBolt, tileMagicBolt, targeted)
	case castMassInvisibility, castMassCharm:
		g.areaSpell(spell, tileLightningBolt, tileCharmBolt, targeted)
	case castMassKill:
		g.areaSpell(spell, tileLightningBolt, tileDeathBolt, targeted)
	case castMassSleep:
		g.areaSpell(spell, tileDispelBolt, tileSleepBolt, targeted)
	case castExplosion:
		g.areaSpell(spell, tileFireBolt, tileFireball, targeted)
	case castRepelUndead, castReveal, castConfuse, castFear:
		g.surroundingSpell(spell)
	case castPoison:
		g.aimSpell(spell, tilePoisonBolt, spellRange, onCharacter, targeted)
	case castGreatLight:
		const greatLightMinutes = 255
		ambient(spellLight, greatLightMinutes)
		g.passTime(0)
	case castPeer:
		g.magicalHalo(caster, 0)
		g.spellFlash(circleOf(spell))
		g.useGem(0)
		g.printf("\n%s\n", doneMsg)
	case castFireField:
		g.fieldSpell(spell, tileFireball, targeted)
	case castPoisonField:
		g.fieldSpell(spell, tilePoisonBolt, targeted)
	case castLocate:
		g.magicalHalo(caster, 0)
		g.spellFlash(circleOf(spell))
		g.locate()
	case castSleepField:
		g.fieldSpell(spell, tileSleepBolt, targeted)
	case castConjure, castSummon:
		g.summon(spell, caster)
	case castPickpocket:
		g.pickpocket(targeted)
	case castWindChange:
		g.windChange()
	case castEnergyField:
		g.fieldSpell(spell, tileMagicBolt, targeted)
	case castInsectSwarm:
		g.insectSwarm(spell)
	case castXray:
		g.xrayVision(spell)
	case castInvisibility:
		g.aimSpell(spell, tileDispelBolt, spellRange, onCharacter, targeted)
	case castDispelMagic, castHarm, castHeal, castCurse, castProtection, castGreatHeal, castParalyze,
		castCharm, castClone:
		g.aimSpell(spell, tileMagicBolt, spellRange, onCharacter, targeted)
	case castKill:
		g.aimSpell(spell, tileDeathBolt, spellRange, onCharacter, targeted)
	case castFlameWind:
		g.windSpell(spell, tileFlameBolt, targeted)
	case castPoisonWind:
		g.windSpell(spell, tilePoisonBolt, targeted)
	case castEnergyWind:
		g.windSpell(spell, tileLightningBolt, targeted)
	case castDeathWind:
		g.windSpell(spell, tileDeathBolt, targeted)
	case castWeb:
		g.webSpell(targeted)
	case castNegateMagic:
		const negateMinutes = 20
		ambient(spellNegateMagic, negateMinutes)
	case castEnchant:
		g.enchant(targeted)
	case castWizardEye:
		g.wizardEye(int(g.MapX), int(g.MapY), true)
	case castGateTravel:
		g.gateTravelSpell()
	case castTimeStop:
		const timeStopMinutes = 10
		ambient(spellTimeStop, timeStopMinutes)
	case castEclipse:
		const eclipseMinutes = 20
		ambient(spellEclipse, eclipseMinutes)
	case castTremor:
		g.tremor()
	case castHailStorm:
		g.hailStormSpell(targeted)
	case castWingStrike:
		g.wingStrike()
	case castSelfInvisibility:
		g.selfInvisibility()
	case castSeance:
		g.seanceSpell(targeted)
	case castArmageddon:
		g.spellFlash(circleOf(spell))
		g.armageddonSpell()
	}
	g.computeView()
	g.OtherAnimations()
}

// spellNegateMagic is the effect of Negate Magic, which the storm cloak also has.
const spellNegateMagic = spellStormCloak

// askTarget prompts for the target of a spell unless it is targeted already; rng, when
// not keepRange, limits the aim cursor.
func (g *Game) askTarget(prompt string, rng int, targeted bool) {
	if g.isPlayerControlled(g.caster()) {
		g.printf("%s", prompt)
	}
	if !targeted {
		g.MouseMode = 1
		if rng != keepRange && g.SelectMode != 0 {
			g.SelectRange = rng
		}
		g.getch()
	}
}

const keepRange = 0

// echoTarget prints the target picked, or nothing; needPlace says that a target without a
// place counts as nothing.
func (g *Game) echoTarget(needPlace bool) {
	sel := g.Selection
	if sel.obj == noObject || needPlace && (sel.x == -1 || sel.y == -1) {
		g.printf("%s\n", nothingMsg)
		return
	}
	g.printf("%s", g.objectString(sel.obj))
	g.printf("\n")
}

// aimSpell casts spell with a missile of tile at a character (kind onCharacter) or an
// object (onObject) within rng.
// C: seg_1944:12FC (C_1944_12FC)
func (g *Game) aimSpell(spell, tile, rng, kind int, targeted bool) {
	caster := g.caster()
	prompt := onWhatMsg
	if kind == onCharacter {
		prompt = onWhomMsg
	}
	g.askTarget(prompt, rng, targeted)
	sel := &g.Selection
	g.echoTarget(true)
	g.magicalHalo(caster, 0)
	reached := false
	if sel.x != -1 && sel.y != -1 {
		reached = g.throw(caster, &sel.x, &sel.y, rng, tile)
	}
	if sel.obj == noObject || sel.x == -1 || sel.y == -1 {
		g.spellFailed(noEffectMsg)
		return
	}
	if !reached {
		p := g.object(caster).Pos
		if !closeEnough(rng, sel.x, sel.y, p.X, p.Y) {
			g.printf("\n%s!\n", outOfRangeMsg)
		} else {
			g.printf("\n%s!\n", blockedMsg)
		}
		g.sfx(sound.EffectFailure, 0)
		return
	}
	if kind == onCharacter && sel.obj >= npcCount || kind == onObject && sel.obj < npcCount && spell != castDispelField {
		g.spellFailed(noEffectMsg)
		return
	}
	sel.obj = g.combatHead(sel.obj)
	target := sel.obj
	t := g.object(target).Type()
	g.spellResult = resultSuccess
	switch spell {
	case castDispelMagic:
		if !g.dispel(target) {
			g.spellResult = resultNoEffect
		}
	case castHarm:
		g.drainSpell(spell)
	case castHeal, castGreatHeal:
		g.magicalHalo(target, 1)
		if !g.healSpell(spell, target) {
			g.spellResult = resultNoEffect
		}
	case castDouse, castIgnite:
		g.douseOrIgnite(target, spell)
	case castVanish:
		if g.TileFlag2[g.tileFrame(target)]&tileGeneric != 0 {
			g.VanishedObject = int16(g.object(target).Shape)
			g.DeleteObj(target)
			g.screenFade = true
			g.computeView()
			g.OtherAnimations()
		} else {
			g.spellResult = resultNoEffect
		}
	case castSleep:
		g.spellResult = g.sleepSpell(caster, target)
	case castMagicArrow:
		g.magicArrow(caster)
	case castLightning:
		const damage = 30
		xp := g.loseHitPoints(target, g.rand(1, damage))
		g.addExperience(caster, xp)
		g.printf("\n")
		g.displaySeverity(target)
		g.provoke(target, caster, 1)
		g.spellResult = resultShown
	case castFireball:
		if g.burn(target) {
			g.spellResult = resultShown
		} else {
			g.spellResult = resultNoEffect
		}
	case castChainBolt:
		g.chainBolt(target)
		g.spellResult = resultShown
	case castPoison:
		g.spellResult = g.poison(caster, target)
	case castCharm:
		g.charm(caster, target)
	case castParalyze:
		g.NPCs[target].Status |= Paralyzed
		g.showHit(target)
		g.printf("\n")
		g.printf("%s", g.objectString(target))
		g.printf(" is paralyzed.\n")
	case castCurse:
		g.object(target).Status |= StatusCursed
	case castProtection:
		g.NPCs[target].Status |= Protected
	case castInvisibility:
		if !g.object(target).Has(StatusInvisible) && canTurnInvisible(t) {
			g.object(target).Status |= StatusInvisible
			g.magicalHalo(target, 1)
		} else {
			g.spellResult = resultNoEffect
		}
	case castKill:
		g.spellResult = g.killSpell(caster, target)
	case castAnimate:
		if g.equipSlot(target) != noSlot {
			p := g.object(target).Pos
			n := g.generate(int(t), p.X, p.Y, int(g.MapZ))
			// fix: the original used the creature even when there was no room for it
			if n >= 0 {
				g.setAlignment(n, g.alignment(caster))
				g.NPCs[n].Mode = AIAssault
			}
			g.DeleteObj(target)
		} else {
			g.spellResult = resultNoEffect
		}
	case castReplicate:
		g.replicate(target)
	case castClone:
		g.clone(target)
	case castDispelField:
		g.dispelField()
	}
	switch g.spellResult {
	case resultSuccess:
		g.spellSucceeded()
	case resultFailed:
		g.spellFailed(failedMsg)
	case resultNoEffect:
		g.spellFailed(noEffectMsg)
	}
}

// replicate copies object n somewhere around the party.
// C: seg_1944:12FC (C_1944_12FC), SPELL_58
func (g *Game) replicate(n int) {
	if g.TileFlag2[g.tileFrame(n)]&tileGeneric == 0 {
		g.spellResult = resultFailed
		return
	}
	o := g.object(n)
	z := int(g.MapZ)
	for range attemptsToPlace {
		x := g.rand(0, placeSpread) + int(g.MapX) - nearbyRange
		y := g.rand(0, placeSpread) + int(g.MapY) - nearbyRange
		c := g.AddMapObj(shapeOf(o.Type(), o.Frame()), 0, 1, x, y, z)
		if c < 0 { // fix: the original used a copy it could not make
			break
		}
		g.setOkToGet(c)
		if g.passable(c, x, y) {
			g.screenFade = true
			g.computeView()
			g.OtherAnimations()
			return
		}
		g.DeleteObj(c)
	}
	g.spellResult = resultFailed
}

// clone makes a copy of character n somewhere around the party.
// C: seg_1944:12FC (C_1944_12FC), SPELL_51
func (g *Game) clone(n int) {
	found := false
	var x, y int
	for range attemptsToPlace {
		x = g.rand(0, placeSpread) + int(g.MapX) - nearbyRange
		y = g.rand(0, placeSpread) + int(g.MapY) - nearbyRange
		if g.passable(n, x, y) {
			found = true
			break
		}
	}
	if !found {
		g.spellResult = resultNoEffect
		return
	}
	c := g.generate(int(g.object(n).Type()), x, y, int(g.MapZ))
	if c >= 0 { // fix: the original used the copy even when there was no room for it
		g.setAlignment(c, g.alignment(n))
		g.NPCs[c].Mode = g.NPCs[n].Mode
	}
	g.screenFade = true
	g.computeView()
	g.OtherAnimations()
}

// dispelField removes the first field at the selected place.
// C: seg_1944:12FC (C_1944_12FC), SPELL_21
func (g *Game) dispelField() {
	sel := g.Selection
	for i := g.FindLoc(sel.x, sel.y, int(g.MapZ)); i >= 0; i = g.NextLoc() {
		switch g.object(i).Type() {
		case ObjFireField, ObjPoisonField, ObjProtectionField, ObjSleepField:
			g.DeleteObj(i)
			g.showHit(i)
			g.computeView()
			return
		}
	}
	g.spellResult = resultNoEffect
}

// poison poisons n unless it resists caster or is immune.
// C: seg_1944:0B32 (C_1944_0B32)
func (g *Game) poison(caster, n int) int {
	if classHas(monsterClass(g.object(n).Type()), immunePoison) {
		return resultNoEffect
	}
	if g.resistsByStrength(caster, n) {
		return resultFailed
	}
	g.NPCs[n].Status |= Poisoned
	g.showHit(n)
	return resultHit
}

// magicArrow hurts the target unless it dodges.
// C: seg_1944:0B85 (C_1944_0B85)
func (g *Game) magicArrow(caster int) {
	target := g.Selection.obj
	if g.dodges(caster, target) {
		g.spellResult = resultFailed
		return
	}
	const damage = 10
	hp := g.rand(1, damage)
	xp := g.loseHitPoints(target, hp)
	g.addExperience(caster, xp)
	g.provoke(target, caster, hp)
	g.printf("\n")
	g.displaySeverity(target)
	g.spellResult = resultShown
}

// drainSpell hurts the target with Harm or Disable unless it resists.
// C: seg_1944:0BF4 (C_1944_0BF4)
func (g *Game) drainSpell(spell int) {
	caster := g.caster()
	target := g.Selection.obj
	if g.testIntelligence(caster, target) {
		g.spellResult = resultFailed
		return
	}
	hp := 0
	switch spell {
	case castHarm:
		const damage = 10
		hp = g.rand(1, damage)
	case castEnergyWind:
		const damage = 30
		hp = g.rand(1, damage)
	case castDisable:
		hp = int(g.NPCs[target].HitPoints) - 1
	}
	g.printf("\n")
	xp := g.loseHitPoints(target, hp)
	g.addExperience(caster, xp)
	g.provoke(target, caster, hp)
	g.displaySeverity(target)
	g.spellResult = resultShown
}

// charm makes n fight for caster unless it resists; a charmed n is freed.
// C: seg_1944:0CA3 (C_1944_0CA3)
func (g *Game) charm(caster, n int) {
	if g.testIntelligence(caster, n) {
		g.spellResult = resultFailed
		return
	}
	if g.object(n).Has(StatusCharmed) {
		g.stopCharm(n)
		return
	}
	npc := &g.NPCs[n]
	npc.Flags = npc.Flags&^NPCFlags(alignmentMask) | NPCFlags(g.alignment(n))
	g.object(n).Status |= StatusCharmed
	g.setAlignment(n, g.alignment(caster))
	npc.Mode = AIAssault
	g.showHit(n)
	g.printf("\n")
	g.printf("%s", g.objectString(n))
	g.printf(" is charmed.\n")
	g.spellResult = resultHit
}

// killSpell kills n unless it resists caster or is immune.
// C: seg_1944:0D46 (C_1944_0D46)
func (g *Game) killSpell(caster, n int) int {
	if g.NPCs[n].Has(Dead) || classHas(monsterClass(g.object(n).Type()), immuneDisease) {
		return resultNoEffect
	}
	if g.testIntelligence(caster, n) {
		return resultFailed
	}
	xp := g.loseHitPoints(n, int(g.NPCs[n].HitPoints))
	g.addExperience(caster, xp)
	g.provoke(n, caster, 1)
	g.displaySeverity(n)
	return resultHit
}

// douseOrIgnite puts out or lights the fire n.
// C: seg_1944:0DCF (C_1944_0DCF)
func (g *Game) douseOrIgnite(n, spell int) {
	o := g.object(n)
	t, frame := o.Type(), o.Frame()
	lit := frame&1 != 0
	switch {
	case t != ObjTorch && t != ObjFireplace && t != ObjCandle && t != ObjCampfire && t != ObjBrazier &&
		t != ObjCandelabra && t != ObjPowderKeg,
		t == ObjBrazier && frame > 1,
		spell == castDouse && !lit,
		spell == castIgnite && lit:
		g.spellResult = resultNoEffect
	case t == ObjPowderKeg && frame == 0:
		o.SetFrame(1)
		const fuseMinutes = 3
		g.KegMinutes = fuseMinutes
		g.KegObject = int16(n)
	case lit:
		if t == ObjTorch {
			g.DeleteObj(n)
			g.printf("\n")
			g.printf(burnOutMsg)
		} else {
			o.SetFrame(frame & 2)
			o.Status &^= StatusLit
		}
	default:
		o.SetFrame(frame | 1)
		o.Status |= StatusLit
	}
	g.computeView()
}

// nearestFoe returns the character nearest to n that n sees, or noObject.
// C: seg_1944:0F02 (C_1944_0F02)
func (g *Game) nearestFoe(n int) int {
	best := noObject
	bestDistance := 1<<15 - 1
	p := g.object(n).Pos
	mx, my := int(g.MapX), int(g.MapY)
	for i := range npcCount {
		if g.object(i).Shape == 0 || g.NPCs[i].Has(Dead) || i == n || !g.canSee(n, i) {
			continue
		}
		q := g.object(i).Pos
		if q.X <= mx-nearbyRange || q.X >= mx+nearbyRange || q.Y <= my-nearbyRange || q.Y >= my+nearbyRange {
			continue
		}
		d := (q.X-p.X)*(q.X-p.X) + (q.Y-p.Y)*(q.Y-p.Y)
		if d <= 0 || d > bestDistance {
			continue
		}
		if d == bestDistance {
			if g.rand(0, 1) == 0 {
				best = i
			}
		} else {
			bestDistance = d
			best = i
		}
	}
	return best
}

// chainBolt hurts n and then jumps to the nearest characters.
// C: seg_1944:1038 (C_1944_1038)
func (g *Game) chainBolt(n int) {
	caster := g.caster()
	const damage, jumps = 30, 6
	xp := g.loseHitPoints(n, g.rand(1, damage))
	g.printf("\n")
	g.addExperience(caster, xp)
	g.provoke(n, caster, 1)
	g.displaySeverity(n)
	for range jumps {
		next := g.nearestFoe(n)
		if next == noObject {
			break
		}
		q := g.object(next).Pos
		x, y := q.X, q.Y
		if !g.throw(n, &x, &y, spellRange, tileLightningBolt) {
			break
		}
		n = next
		xp := g.loseHitPoints(n, g.rand(1, damage))
		g.addExperience(caster, xp)
		g.printf("\n")
		g.displaySeverity(n)
	}
}

// healSpell gives n hit points with Heal or Great Heal; it returns false when n is dead.
// C: seg_1944:114D (C_1944_114D)
func (g *Game) healSpell(spell, n int) bool {
	npc := &g.NPCs[n]
	if npc.Has(Dead) {
		return false
	}
	switch spell {
	case castHeal:
		const healing = 30
		gain := g.rand(1, healing)
		if g.maxHitPoints(n) > int(npc.HitPoints)+gain {
			npc.HitPoints += byte(gain)
			return true
		}
	case castGreatHeal:
	default:
		return true
	}
	npc.HitPoints = byte(g.maxHitPoints(n))
	return true
}

// dispel wakes, cures, frees and uncharms n; it returns false when there was nothing to
// dispel.
// C: seg_1944:11BB (C_1944_11BB)
func (g *Game) dispel(n int) bool {
	done := false
	g.magicalHalo(n, 1)
	npc := &g.NPCs[n]
	if npc.Has(Asleep) {
		npc.Status &^= Asleep
		g.object(n).Shape = npc.OrigShape
		done = true
	}
	if npc.Has(Poisoned) {
		npc.Status &^= Poisoned
		done = true
	}
	if npc.Has(Paralyzed) {
		npc.Status &^= Paralyzed
		done = true
	}
	if g.object(n).Has(StatusCharmed) {
		g.stopCharm(n)
		done = true
	}
	return done
}

// burn hurts n with fire; it returns false when n does not burn.
// C: seg_1944:1232 (C_1944_1232)
func (g *Game) burn(n int) bool {
	cls := monsterClass(g.object(n).Type())
	if classHas(cls, immuneFire) {
		return false
	}
	const damage = 20
	hp := g.rand(1, damage)
	if classHas(cls, doubleFireDamage) {
		hp <<= 1
	}
	caster := g.caster()
	xp := g.loseHitPoints(n, hp)
	g.addExperience(caster, xp)
	g.printf("\n")
	g.provoke(n, caster, 1)
	g.displaySeverity(n)
	return true
}

// resurrectSpell brings a body back to life.
// C: seg_1944:1C51 (C_1944_1C51)
func (g *Game) resurrectSpell(targeted bool) {
	g.printf("%s", onWhomMsg)
	if !targeted {
		saved := g.StatusDisplay
		g.StatusDisplay = cmdStatusEquipment
		g.equipmentView()
		g.MouseMode = 1
		g.getch()
		g.StatusDisplay = saved
	}
	sel := g.Selection
	g.echoTarget(false)
	g.magicalHalo(g.caster(), 0)
	if sel.obj == noObject || g.armageddon {
		g.spellFailed(noEffectMsg)
		return
	}
	g.magicalHalo(sel.obj, 1)
	o := g.object(sel.obj)
	const oldestBody = 224 // bodies of monsters, which do not come back
	if t := o.Type(); t != ObjDeadBody && t != ObjMouse && t != ObjDeadGargoyle && t != ObjDeadCyclops ||
		o.Quality() >= oldestBody {
		g.spellFailed(noEffectMsg)
		return
	}
	if o.Quality() == 0 {
		g.printf("\nThe corpse is too old!\n")
		return
	}
	g.resurrect(sel.obj)
	g.spellSucceeded()
	g.spellResult = resultSuccess
}

// potionNames are the potions Detect Magic names, by frame (D_0EE2).
var potionNames = [8]string{"an awaken", "a cure", "a heal", "a poison", "a sleep", "a protection",
	"an invisibility", "an xray vision"}

// touchSpell casts spell on a character (kind onCharacter) or object (onObject) within
// rng, without a missile.
// C: seg_1944:1DF4 (C_1944_1DF4)
func (g *Game) touchSpell(spell, rng, kind int, targeted bool) {
	caster := g.caster()
	prompt := onWhatMsg
	if kind == onCharacter {
		prompt = onWhomMsg
	}
	g.askTarget(prompt, rng, targeted)
	sel := &g.Selection
	placeless := spell == castDetectMagic || spell == castDetectTrap
	g.echoTarget(!placeless)
	g.magicalHalo(caster, 0)
	if sel.obj == noObject || sel.x == -1 && !placeless {
		g.spellFailed(noEffectMsg)
		return
	}
	p := g.object(caster).Pos
	if !closeEnough(rng, sel.x, sel.y, p.X, p.Y) && !placeless && spell != castTrap &&
		spell != castUntrap && spell != castLock && spell != castUnlockMagic {
		g.printf("\n%s!\n", outOfRangeMsg)
		g.sfx(sound.EffectFailure, 0)
		return
	}
	if kind == onCharacter && sel.obj >= npcCount || kind == onObject && sel.obj < npcCount {
		g.spellFailed(noEffectMsg)
		return
	}
	sel.obj = g.combatHead(sel.obj)
	n := sel.obj
	o := g.object(n)
	t := o.Type()
	g.spellResult = resultSuccess
	const magicallyLocked = 12 // the frame bits of a door locked by magic
	const chestMagicallyLocked = 3
	switch spell {
	case castDisable:
		g.drainSpell(spell)
	case castLock:
		switch {
		case isDoor(t):
			g.useDoor(n, false, true, false)
		case t == ObjOpenChest:
			g.useChest(n, false, true, false)
			g.spellResult = resultShown
		default:
			g.spellResult = resultNoEffect
		}
	case castUnlockMagic:
		switch {
		case isDoor(t) && o.Frame()&magicallyLocked == magicallyLocked:
			g.useDoor(n, false, false, true)
			g.spellResult = resultShown
		case t == ObjOpenChest && o.Frame() == chestMagicallyLocked:
			g.useChest(n, false, false, true)
			g.spellResult = resultShown
		default:
			g.spellResult = resultNoEffect
		}
	case castUntrap:
		g.magicalHalo(n, 1)
		if (t == ObjOpenChest || isDoor(t)) && g.FindInvType(n, ObjEffect, castTrap) != noObject {
			g.DeleteObj(g.heldOfQuality(n, ObjEffect, castTrap))
		} else {
			g.spellResult = resultNoEffect
		}
	case castTrap:
		g.magicalHalo(n, 1)
		if (t == ObjOpenChest || isDoor(t)) && g.FindInvType(n, ObjEffect, castTrap) == noObject {
			g.giveObj(n, ObjEffect, qualQuan(castTrap, 1))
		} else {
			g.spellResult = resultNoEffect
		}
	case castDetectTrap:
		g.magicalHalo(n, 1)
		const odds = 30
		if g.rand(1, odds) < g.intelligenceOf(caster) {
			trapped := false
			for i := g.FindInv(n); i >= 0; i = g.NextInv() {
				if g.object(i).Type() == ObjEffect {
					g.printf("\nIt's trapped.\n")
					trapped = true
				}
			}
			if trapped {
				g.spellResult = resultShown
				break
			}
		}
		g.printf("\nNo trap\n")
		g.spellResult = resultShown
	case castDetectMagic:
		g.magicalHalo(n, 1)
		const (
			notMagical = iota
			magical
			charged
		)
		found := notMagical
		if g.isMagical(n) {
			found = magical
		} else {
			for i := g.FindInv(n); i >= 0; i = g.NextInv() {
				if g.object(i).Type() == ObjCharge {
					g.printf("\nIt shows a charge of %s.\n", spellNames[g.object(i).Quality()])
					found = charged
					break
				}
			}
		}
		switch found {
		case notMagical:
			g.printf("\nIt's not magical.\n")
		case magical:
			if t == ObjBluePotion {
				g.printf("\nIt's %s potion.\n", potionNames[o.Frame()])
			} else {
				g.printf("\nIt's magical.\n")
			}
		}
		g.spellResult = resultShown
	}
	switch g.spellResult {
	case resultSuccess:
		g.spellSucceeded()
	case resultFailed:
		g.spellFailed(failedMsg)
	case resultNoEffect:
		g.spellFailed(noEffectMsg)
	}
}

// classAlligator is the one swimming creature Slime turns to slime.
const classAlligator = 1

// slime turns the evil creatures around the party into slimes.
// C: seg_1944:22F4 (C_1944_22F4)
func (g *Game) slime() {
	caster := g.caster()
	g.magicalHalo(caster, 0)
	mx, my := int(g.MapX), int(g.MapY)
	for x := mx - nearbyRange; x <= mx+nearbyRange; x++ {
		for y := my - nearbyRange; y <= my+nearbyRange; y++ {
			n := g.FindLoc(x, y, int(g.MapZ))
			if n == noObject || n >= npcCount || g.alignment(n) != alignEvil || g.testIntelligence(caster, n) {
				continue
			}
			if cls := monsterClass(g.object(n).Type()); !classHas(cls, swims) || cls == classAlligator {
				g.object(n).Shape = shapeOf(ObjSlime, 0)
				if n >= firstMonster {
					for i := g.FindInv(n); i >= 0; i = g.FindInv(n) {
						g.DeleteObj(i)
					}
				}
			}
		}
	}
	g.screenFade = true
	g.computeView()
	g.OtherAnimations()
}

// aimAtPlace prompts for the place of a spell, prints the newline after it and stops
// the caster's halo; it returns false (saying so) when no place was picked.
func (g *Game) aimAtPlace(targeted bool) bool {
	g.askTarget(locationMsg, keepRange, targeted)
	g.printf("\n")
	g.magicalHalo(g.caster(), 0)
	if g.Selection.x == -1 || g.Selection.y == -1 {
		g.spellFailed(noEffectMsg)
		return false
	}
	return true
}

// reappear brings back what Vanish took.
// C: seg_1944:23F2 (C_1944_23F2)
func (g *Game) reappear(targeted bool) {
	caster := g.caster()
	g.askTarget(locationMsg, keepRange, targeted)
	g.printf("\n")
	g.magicalHalo(caster, 0)
	sel := &g.Selection
	if sel.x == -1 || sel.y == -1 || g.VanishedObject == 0 {
		g.spellFailed(noEffectMsg)
		return
	}
	if !g.throw(caster, &sel.x, &sel.y, spellRange, tileMagicBolt) {
		g.spellFailed(blockedMsg)
		return
	}
	n := g.AddMapObj(uint16(g.VanishedObject), 0, 1, sel.x, sel.y, int(g.MapZ))
	if n < 0 { // fix: the original used the object it could not make
		g.spellFailed(noEffectMsg)
		return
	}
	g.setOkToGet(n)
	if g.passable(n, sel.x, sel.y) {
		g.screenFade = true
		g.computeView()
		g.OtherAnimations()
		g.spellSucceeded()
		g.VanishedObject = 0
		return
	}
	g.DeleteObj(n)
	g.spellFailed(noEffectMsg)
}

// areaSpell flies a missile of tile to a place and blows up blast there; spell then acts on
// every character caught.
// C: seg_1944:256A (C_1944_256A)
func (g *Game) areaSpell(spell, tile, blast int, targeted bool) {
	caster := g.caster()
	if !g.aimAtPlace(targeted) {
		return
	}
	sel := &g.Selection
	if !g.throw(caster, &sel.x, &sel.y, spellRange, tile) {
		g.spellFailed(blockedMsg)
		return
	}
	for _, n := range append([]int(nil), g.explosion(sel.x, sel.y, blast)...) {
		if n >= npcCount {
			continue
		}
		switch spell {
		case castMassAwaken:
			g.NPCs[n].Status &^= Asleep
			g.object(n).Shape = g.NPCs[n].OrigShape
		case castMassProtect:
			g.NPCs[n].Status |= Protected
		case castMassCurse:
			g.object(n).Status |= StatusCursed
		case castMassSleep:
			g.sleepSpell(caster, n)
		case castMassDispel:
			g.dispel(n)
		case castExplosion:
			g.burn(n)
			g.spellResult = resultShown
		case castMassCharm:
			g.charm(caster, n)
		case castMassKill:
			g.killSpell(caster, n)
		case castMassInvisibility:
			if !g.object(n).Has(StatusInvisible) && canTurnInvisible(g.object(n).Type()) {
				g.object(n).Status |= StatusInvisible
				g.magicalHalo(n, 1)
			}
		}
	}
	if spell == castExplosion || spell == castMassKill {
		g.afterExplosion(sel.x, sel.y)
	}
	if g.spellResult != resultShown {
		g.printf("\n%s\n", successMsg)
	}
}

// fieldSpell makes a fire, poison, sleep or energy field where a missile of tile lands.
// C: seg_1944:27AF (C_1944_27AF)
func (g *Game) fieldSpell(spell, tile int, targeted bool) {
	caster := g.caster()
	if !g.aimAtPlace(targeted) {
		return
	}
	sel := &g.Selection
	if !g.throw(caster, &sel.x, &sel.y, spellRange, tile) {
		g.spellFailed(blockedMsg)
		return
	}
	field := map[int]ObjectType{castFireField: ObjFireField, castPoisonField: ObjPoisonField,
		castSleepField: ObjSleepField, castEnergyField: ObjProtectionField}[spell]
	n := g.AddMapObj(shapeOf(field, 0), 0, 1, sel.x, sel.y, int(g.MapZ))
	if n >= 0 && g.passable(n, sel.x, sel.y) {
		g.screenFade = true
		g.computeView()
		g.OtherAnimations()
		g.printf("\n%s\n", successMsg)
		return
	}
	if n >= 0 {
		g.DeleteObj(n)
	}
	g.spellFailed(notPossibleMsg)
}

// frighten makes n flee unless it is good, resists or cannot fear.
// C: seg_1944:29AA (C_1944_29AA)
func (g *Game) frighten(n int) {
	if n >= npcCount || g.alignment(n) == alignGood || g.testIntelligence(g.caster(), n) {
		return
	}
	if !classHas(monsterClass(g.object(n).Type()), immuneDisease) {
		g.NPCs[n].Mode = AIFear
	}
}

// surroundingSpell casts Repel Undead, Reveal, Fear or Confuse on everything around the
// party.
// C: seg_1944:2A15 (C_1944_2A15)
func (g *Game) surroundingSpell(spell int) {
	caster := g.caster()
	g.magicalHalo(caster, 0)
	mx, my := int(g.MapX), int(g.MapY)
	for x := mx - widerRange; x <= mx+widerRange; x++ {
		for y := my - widerRange; y <= my+widerRange; y++ {
			n := g.FindLoc(x, y, int(g.MapZ))
			if n < 0 {
				continue
			}
			switch spell {
			case castRepelUndead:
				if classHas(monsterClass(g.object(n).Type()), undead) && !g.testIntelligence(caster, n) {
					g.NPCs[n].Mode = AIFear
				}
			case castReveal:
				g.object(n).Status &^= StatusInvisible
				if ring := g.FindInvType(n, ObjInvisibilityRing, 0); ring != noObject && g.object(ring).CoordUse() == Equipped {
					g.object(ring).SetCoordUse(InInventory)
				}
			case castFear:
				g.frighten(n)
			case castConfuse:
				if n < npcCount && g.alignment(n) == alignEvil && !g.testIntelligence(caster, n) {
					g.NPCs[n].Mode = AIAssault
					g.setAlignment(n, alignNeutral)
				}
			}
		}
	}
	g.spellSucceeded()
}

// createFood gives the caster some meat.
// C: seg_1944:2B9A (C_1944_2B9A)
func (g *Game) createFood(spell int) {
	g.spellFlash(circleOf(spell))
	g.magicalHalo(g.caster(), 0)
	n := g.rand(1, spellFoodMax)
	g.giveObj(g.caster(), ObjPortionOfMeat, n)
	g.printf("\nCreate %d food.\n", n)
}

// summon calls a creature (Conjure) or a daemon (Summon) to fight for n.
// C: seg_1944:2BFD (C_1944_2BFD)
func (g *Game) summon(spell, n int) {
	var t ObjectType
	switch r := g.rand(0, 15); {
	case r <= 3:
		t = ObjTroll
	case r <= 6:
		t = ObjGiantRat
	case r <= 9:
		t = ObjGiantSpider
	case r <= 12:
		t = ObjGiantBat
	default:
		t = ObjSnake
	}
	g.spellFlash(circleOf(spell))
	g.magicalHalo(n, 0)
	mx, my, z := int(g.MapX), int(g.MapY), int(g.MapZ)
	if spell == castSummon {
		t = ObjDaemon
	}
	c := g.generate(int(t), mx, my, z)
	// fix: the original used the creature even when there was no room for it
	if c < 0 {
		g.sfx(sound.EffectFailure, 0)
		return
	}
	for range attemptsToPlace {
		x := mx + g.rand(0, placeSpread) - nearbyRange
		y := my + g.rand(0, placeSpread) - nearbyRange
		if !g.passable(c, x, y) {
			continue
		}
		g.MoveObj(c, x, y, z)
		var a NPCStatus
		if spell == castSummon && g.testIntelligence(n, c) {
			a = alignChaotic
			g.printf("\nHostile daemon summoned.\n")
			g.sfx(sound.EffectFailure, 0)
		} else {
			a = g.alignment(n)
			g.spellSucceeded()
		}
		g.setAlignment(c, a)
		g.NPCs[c].Mode = AIAssault
		g.screenFade = true
		g.computeView()
		g.OtherAnimations()
		return
	}
	g.DeleteObj(c)
	g.sfx(sound.EffectFailure, 0)
}

// telekinesis moves an object, works a crank or switches a lever from afar.
// C: seg_1944:2DA7 (C_1944_2DA7)
func (g *Game) telekinesis(targeted bool) {
	caster := g.caster()
	g.askTarget(onWhatMsg, keepRange, targeted)
	sel := &g.Selection
	g.echoTarget(true)
	g.magicalHalo(caster, 0)
	reached := false
	if sel.x != -1 && sel.y != -1 {
		reached = g.throw(caster, &sel.x, &sel.y, spellRange, tileMagicBolt)
	}
	if sel.obj == noObject || sel.obj < npcCount || sel.x == -1 || sel.y == -1 {
		g.spellFailed(noEffectMsg)
		return
	}
	if !reached {
		g.spellFailed(blockedMsg)
		return
	}
	n := sel.obj
	t := g.object(n).Type()
	switch {
	case t == ObjCrank:
		g.turnCrank(n)
		return
	case t == ObjLever:
		g.switchLever(n, ObjPortcullis, "\nSwitch the lever, ")
		return
	case g.TypeWeight[t] == 0:
		g.printf("\nNot Possible\n")
		return
	}
	g.magicalHalo(n, 1)
	p := g.object(n).Pos
	g.MouseMode = 0
	g.pointerCentreX = (p.X-int(g.MapX))*tileSize + mapCentrePixel
	g.pointerCentreY = (p.Y-int(g.MapY))*tileSize + mapCentrePixel
	g.printf("Direction-")
	if g.getch() != cmdAdvance {
		g.printf("nowhere\n")
	} else {
		d := g.AdvanceDir
		g.printf("%s\n", dirNames[d])
		if g.cannotPush(n, p.X, p.Y, d) {
			g.printf("\n%s\n", blockedMsg)
		} else {
			g.MoveObj(n, p.X+dirStepX[d], p.Y+dirStepY[d], int(g.MapZ))
			if t == ObjCannon {
				g.object(n).SetFrame(d >> 1 & 3)
			}
			g.computeView()
		}
	}
	g.pointerCentreX, g.pointerCentreY = mapCentrePixel, mapCentrePixel
	g.updatePointerShape()
}

// gateTravelSpell takes the party to the moonstone of a phase.
// C: seg_1944:305A (C_1944_305A)
func (g *Game) gateTravelSpell() {
	g.printf("To phase ")
	typed := g.gets(4)
	g.printf("\n")
	g.magicalHalo(g.caster(), 0)
	phase := g.parseNumber(typed, 0)
	if phase == numberEmpty || phase == numberCancelled || phase <= 0 || phase > len(g.Moonstones) ||
		g.Moonstones[phase-1] == [3]int16{} {
		g.sfx(sound.EffectFailure, 0)
		return
	}
	g.gateTravel(phase - 1)
}

// xrayVision shows everything around the party for a while.
// C: seg_1944:310C (C_1944_310C)
func (g *Game) xrayVision(spell int) {
	g.spellFlash(circleOf(spell))
	g.magicalHalo(g.caster(), 0)
	g.xray = true
	g.computeView()
	const frames = 100
	for range frames {
		g.OtherAnimations()
	}
	g.xray = false
	g.computeView()
	g.OtherAnimations()
	g.printf("\n%s\n", doneMsg)
}

// pickpocket takes an object from a character from afar.
// C: seg_1944:3175 (C_1944_3175)
func (g *Game) pickpocket(targeted bool) {
	caster := g.caster()
	if g.isPlayerControlled(caster) {
		g.printf("%s", onWhomMsg)
	}
	if !targeted {
		g.MouseMode = 1
		saved := g.SelectMode
		g.getch()
		g.SelectMode = saved
	}
	sel := &g.Selection
	g.echoTarget(true)
	g.magicalHalo(caster, 0)
	reached := false
	if sel.x != -1 && sel.y != -1 {
		reached = g.throw(caster, &sel.x, &sel.y, spellRange, tileDispelBolt)
	}
	if sel.obj == noObject || sel.obj >= npcCount || sel.x == -1 || sel.y == -1 {
		g.spellFailed(noEffectMsg)
		return
	}
	if !reached {
		g.spellFailed(blockedMsg)
		return
	}
	g.spellPeek = true
	p := g.object(caster).Pos
	casterX, casterY := p.X, p.Y
	victim := sel.obj
	g.showNPC = victim
	g.equipmentView()
	g.StatusDisplay = cmdStatusEquipment
	g.printf("Which object: ")
	if g.SelectMode != 0 {
		g.SelectMode = 2
		g.drawStatusCursor(1)
	}
	ch := g.getch()
	g.spellPeek = false
	if ch != cmdSelected || sel.obj == noObject {
		g.printf("%s\n", nothingMsg)
	} else {
		g.printf("%s", g.objectString(sel.obj))
		g.printf("\n")
	}
	if sel.obj == noObject || sel.obj < npcCount || g.object(sel.obj).CoordUse() == OnMap {
		g.spellFailed(noEffectMsg)
	} else if g.unreadyCheck(sel.obj) == unreadyFree {
		n := sel.obj
		t := g.object(n).Type()
		g.weighInventory(caster, false)
		if g.weightOf(n)+g.weightInven+g.weightEquip > int(g.NPCs[caster].Strength)*carryPerStrength {
			g.printf("\nCan't carry!\n")
		} else {
			g.throw(victim, &casterX, &casterY, spellRange, g.tileFrame(n))
			g.stopWornEffect(n, victim)
			if quantityType(t) != countSingle {
				amount := int(g.object(n).Amount)
				g.giveObj(caster, t, amount)
				g.takeObj(victim, t, amount)
			} else {
				g.InsertObj(n, caster, InInventory)
			}
			g.refreshInventory(caster)
			g.printf("\n%s\n", successMsg)
			const theft = 5
			g.Karma -= theft
		}
		g.statusMember = int(g.Active)
		g.inventoryScroll = 0
		g.StatusDirty = true
	}
	g.showNPC = 0
	g.npcOneHanded = false
}

// insectSwarm calls insects around the party to fight for it.
// C: seg_1944:34CF (C_1944_34CF)
func (g *Game) insectSwarm(spell int) {
	g.spellFlash(circleOf(spell))
	caster := g.caster()
	g.magicalHalo(caster, 0)
	mx, my, z := int(g.MapX), int(g.MapY), int(g.MapZ)
	const swarms = 8
	for range swarms {
		n := g.generate(int(ObjInsects), mx, my, z)
		if n < 0 { // fix: the original used the insects even when there was no room for them
			continue
		}
		placed := false
		for range attemptsToPlace {
			x := mx + g.rand(0, placeSpread) - nearbyRange
			y := my + g.rand(0, placeSpread) - nearbyRange
			if g.passable(n, x, y) {
				g.MoveObj(n, x, y, z)
				g.setAlignment(n, g.alignment(caster))
				g.NPCs[n].Mode = AIAssault
				placed = true
				break
			}
		}
		if !placed {
			g.DeleteObj(n)
		}
	}
	g.spellSucceeded()
}

// webSpell covers a place and the eight around it with webs.
// C: seg_1944:35EC (C_1944_35EC)
func (g *Game) webSpell(targeted bool) {
	if g.isPlayerControlled(g.caster()) {
		g.printf("%s", locationMsg)
	}
	if !targeted {
		g.MouseMode = 1
		g.getch()
	}
	g.printf("\n")
	caster := g.caster()
	g.magicalHalo(caster, 0)
	sel := g.Selection
	if sel.x == -1 || sel.y == -1 {
		g.spellFailed(noEffectMsg)
		return
	}
	if !g.webMissiles(caster, sel.x, sel.y) {
		g.spellFailed(blockedMsg)
		return
	}
	z := int(g.MapZ)
	for x := sel.x - 1; x <= sel.x+1; x++ {
		for y := sel.y - 1; y <= sel.y+1; y++ {
			w := g.AddMapObj(shapeOf(ObjWeb, 0), 0, 1, 0, 0, z)
			if w < 0 {
				continue
			}
			if g.passable(w, x, y) {
				g.MoveObj(w, x, y, z)
			} else {
				g.DeleteObj(w)
			}
		}
	}
	g.computeView()
	g.spellSucceeded()
}

// webMissiles flies webs from n to (x, y) and the eight places around it.
// C: seg_2FC1:131E (C_2FC1_131E)
func (g *Game) webMissiles(n, x, y int) bool {
	var deltaX, deltaY [len(dirStepX) + 1]int
	for d := range dirStepX {
		deltaX[d], deltaY[d] = x+dirStepX[d], y+dirStepY[d]
	}
	deltaX[len(dirStepX)], deltaY[len(dirStepX)] = x, y
	p := g.object(n).Pos
	return g.missiles(p.X, p.Y, deltaX[:], deltaY[:], widerRange, int(g.BaseTile[ObjWeb]), missileNoSpin|missileAltSound)
}

// enchant puts a charge of a spell of the spellbooks into a staff.
// C: seg_1944:3750 (C_1944_3750)
func (g *Game) enchant(targeted bool) {
	g.StatusDisplay = cmdStatusEquipment
	g.equipmentView()
	g.printf("%s", onWhatMsg)
	if !targeted {
		g.MouseMode = 1
		g.getch()
	}
	sel := &g.Selection
	g.echoTarget(false)
	// fix: the original read the type of object -1 when nothing was picked
	if sel.obj == noObject || g.object(sel.obj).Type() != ObjStaff {
		g.spellFailed(noEffectMsg)
		return
	}
	staff := sel.obj
	charges := 0
	for i := g.FindInv(staff); i >= 0; i = g.NextInv() {
		charges++
	}
	const maxCharges = 10
	if charges == maxCharges {
		g.printf("\nIt's full\n")
		g.sfx(sound.EffectFailure, 0)
		return
	}
	g.spellbooks[0], g.spellbooks[1] = g.heldSpellbooks()
	g.refreshSpellList()
	g.drawSpellbook()
	g.refreshPage()
	g.StatusDisplay = cmdStatusSpellbook
	g.printf("With: ")
	ch := g.getch()
	g.magicalHalo(g.caster(), 0)
	if ch != cmdSelected || sel.obj == noObject {
		g.printf("none\n")
	} else {
		spell := g.pageSpells[sel.obj]
		g.printf("%s", incantation(spell))
		g.giveObj(staff, ObjCharge, qualQuan(spell, 1))
	}
	g.printf("\n")
}

// wizardEye moves an eye over the map from (x, y) while the player steers it; aware says
// it is the spell (with its flash), not the crystal ball.
// C: seg_1944:39A1 (C_1944_39A1)
func (g *Game) wizardEye(x, y int, spell bool) {
	moves := 40
	tile := 0
	ch := 0
	if spell {
		const flashCircle = 6
		g.spellFlash(flashCircle)
		g.magicalHalo(g.caster(), 0)
	}
	g.peering = true
	savedX, savedY := g.MapX, g.MapY
	g.MapX, g.MapY = int16(x), int16(y)
	z := int(g.MapZ)
	eye := g.AddMapObj(shapeOf(ObjWizardSEye, 0), 0, 1, x, y, z)
	g.xray = true
	g.screenFade = true
	g.ShowObjects()
	g.enterArea()
	g.OtherAnimations()
	tile = g.GetTileAtXYZ(int(g.MapX), int(g.MapY), z)
	const firstObstacle = 252 // TIL_0FC: the eye stops at the walls and objects
	for moves >= 0 && ch != keyEscape && tile < firstObstacle {
		g.MouseMode = 0
		g.peering = false
		ch = g.getch()
		if ch == cmdAdvance {
			g.MapX += int16(dirStepX[g.AdvanceDir])
			g.MapY += int16(dirStepY[g.AdvanceDir])
			if eye >= 0 {
				g.MoveObj(eye, int(g.MapX), int(g.MapY), z)
			}
			g.ShowObjects()
			g.enterArea()
			g.OtherAnimations()
			moves--
			tile = g.GetTileAtXYZ(int(g.MapX), int(g.MapY), z)
		}
	}
	if eye >= 0 {
		g.DeleteObj(eye)
	}
	g.xray = false
	g.MapX, g.MapY = savedX, savedY
	g.ShowObjects()
	g.enterArea()
	g.screenFade = true
	g.OtherAnimations()
	g.printf("\n%s\n", doneMsg)
	if g.StatusDisplay == cmdStatusEquipment {
		g.refreshInventory(int(g.Party[g.statusMember]))
	}
}

// tremor shakes the ground and hurts the evil creatures around the party.
// C: seg_1944:3B2D (C_1944_3B2D)
func (g *Game) tremor() {
	caster := g.caster()
	g.magicalHalo(caster, 0)
	const shakes, amplitude = 20, 3
	g.shakeScreen(shakes, amplitude)
	mx, my := int(g.MapX), int(g.MapY)
	for x := mx - nearbyRange; x <= mx+nearbyRange; x++ {
		for y := my - nearbyRange; y <= my+nearbyRange; y++ {
			n := g.FindLoc(x, y, int(g.MapZ))
			if n < 0 || n >= npcCount || g.alignment(n) != alignEvil || classHas(monsterClass(g.object(n).Type()), flies) {
				continue
			}
			const damage = 30
			xp := g.loseHitPoints(n, g.rand(1, damage))
			g.addExperience(caster, xp)
			g.printf("\n")
			g.displaySeverity(n)
		}
	}
	g.printf("\n%s\n", successMsg)
}

// seanceSpell lets the party talk to the dead.
// C: seg_1944:3C30 (C_1944_3C30)
func (g *Game) seanceSpell(targeted bool) {
	g.printf("%s", onWhomMsg)
	if !targeted {
		g.MouseMode = 1
		g.getch()
	}
	g.printf("\n")
	g.magicalHalo(g.caster(), 0)
	n := g.Selection.obj
	// fix: the original read the type of object -1 when nothing was picked
	if n == noObject {
		g.spellFailed(noEffectMsg)
		return
	}
	if t := g.object(n).Type(); t != ObjDeadBody && t != ObjMouse && t != ObjDeadGargoyle && t != ObjDeadCyclops {
		g.spellFailed(noEffectMsg)
		return
	}
	g.printf("%s", g.objectString(n))
	g.printf("\n")
	g.seance = true
	g.initTalk(int(g.Active), g.object(n).Quality())
	g.seance = false
}

// hailStormSpell drops hail on a place.
// C: seg_1944:3D0F (C_1944_3D0F)
func (g *Game) hailStormSpell(targeted bool) {
	if !g.aimAtPlace(targeted) {
		return
	}
	g.hailStorm(g.Selection.x, g.Selection.y)
	g.printf("\n%s\n", successMsg)
}

// windSpell blows a wind of tile towards a place; spell then acts on every character
// caught.
// C: seg_1944:3DB8 (C_1944_3DB8)
func (g *Game) windSpell(spell, tile int, targeted bool) {
	caster := g.caster()
	g.askTarget(locationMsg, keepRange, targeted)
	g.printf("\n")
	g.magicalHalo(caster, 0)
	sel := g.Selection
	p := g.object(caster).Pos
	if sel.x == -1 || sel.y == -1 || p.X == sel.x && p.Y == sel.y {
		g.spellFailed(noEffectMsg)
		return
	}
	for _, n := range append([]int(nil), g.magicWind(caster, sel.x, sel.y, tile)...) {
		if n >= npcCount {
			continue
		}
		switch spell {
		case castFlameWind:
			g.burn(n)
		case castPoisonWind:
			g.poison(caster, n)
		case castEnergyWind:
			const damage = 30
			xp := g.loseHitPoints(n, g.rand(1, damage))
			g.addExperience(caster, xp)
			g.displaySeverity(n)
			g.provoke(n, caster, 1)
		case castDeathWind:
			g.killSpell(caster, n)
		}
	}
}

// magicWind blows five missiles of tile from n towards (x, y), in a fan; it returns the
// characters caught.
// C: seg_2FC1:11EB (MagicWind)
func (g *Game) magicWind(n, x, y, tile int) []int {
	p := g.object(n).Pos
	// fix: the original read outside its table for a place more than 5 tiles away
	row := min(max(y-p.Y+viewCenter, 0), viewTiles-1)
	col := min(max(x-p.X+viewCenter, 0), viewTiles-1)
	dirs := int(tripleDirections[row][col])
	left, right := dirs>>nibble, dirs&(1<<nibble-1)
	deltaX := []int{x, x + dirStepX[left], x + dirStepX[right], 0, 0}
	deltaY := []int{y, y + dirStepY[left], y + dirStepY[right], 0, 0}
	deltaX[3], deltaY[3] = deltaX[1]+dirStepX[left], deltaY[1]+dirStepY[left]
	deltaX[4], deltaY[4] = deltaX[2]+dirStepX[right], deltaY[2]+dirStepY[right]
	g.missiles(p.X, p.Y, deltaX, deltaY, widerRange, tile, missileAltSound|missileTrail)
	return g.hitList
}

// Wing Strike: the dragon flies in from the side of the view farther from the foe.
var (
	wingSide  = [2]int{1, -1}
	leftWing  = [2]int{26, 38} // the frames of the dragon's wings (D_0EF4, D_0EF6)
	rightWing = [2]int{34, 30}
)

// wingStrike sends a dragon over every evil creature in view.
// C: seg_1944:3F6C (C_1944_3F6C)
func (g *Game) wingStrike() {
	hit := false
	g.magicalHalo(g.caster(), 0)
	mx, my, z := int(g.MapX), int(g.MapY), int(g.MapZ)
	for n := range npcCount {
		o := g.object(n)
		a := g.alignment(n)
		if o.Shape == 0 || g.NPCs[n].Has(Dead) || o.Has(StatusInvisible) || a != alignEvil && a != alignChaotic {
			continue
		}
		x, y := o.Pos.X, o.Pos.Y
		if mx-nearbyRange > x || mx+nearbyRange < x || my-nearbyRange > y || my+nearbyRange < y {
			continue
		}
		side := 1
		if x-(mx-nearbyRange) > mx+nearbyRange-x {
			side = 0
		}
		startX := mx - wingSide[side]*nearbyRange
		wing0 := g.AddMapObj(shapeOf(ObjDragon, leftWing[side]), 0, 1, startX, y-1, z)
		wing1 := g.AddMapObj(shapeOf(ObjDragon, rightWing[side]), 0, 1, startX, y, z)
		// fix: the original flew dragons it could not make
		if wing0 < 0 || wing1 < 0 {
			if wing0 >= 0 {
				g.DeleteObj(wing0)
			}
			if wing1 >= 0 {
				g.DeleteObj(wing1)
			}
			continue
		}
		g.computeView()
		g.OtherAnimations()
		const flight = 11
		for i := 1; i < flight; i++ {
			w0, w1 := g.object(wing0), g.object(wing1)
			w0.SetFrame(w0.Frame() ^ 1)
			w1.SetFrame(w1.Frame() ^ 1)
			at := mx - (nearbyRange-i)*wingSide[side]
			g.MoveObj(wing0, at, y-1, z)
			g.MoveObj(wing1, at, y, z)
			g.computeView()
			g.OtherAnimations()
			if at == x {
				const damage = 20
				xp := g.loseHitPoints(n, g.rand(1, damage))
				g.addExperience(g.caster(), xp)
				g.displaySeverity(n)
			}
		}
		hit = true
		g.DeleteObj(wing0)
		g.DeleteObj(wing1)
	}
	if !hit {
		g.printf("\nNo foes.\n")
		g.sfx(sound.EffectFailure, 0)
	}
}

// selfInvisibility makes the caster invisible.
// C: seg_1944:4260 (C_1944_4260)
func (g *Game) selfInvisibility() {
	n := g.caster()
	if g.object(n).Has(StatusInvisible) {
		g.printf("\n%s\n", noEffectMsg)
		return
	}
	g.object(n).Status |= StatusInvisible
	g.magicalHalo(n, 1)
	g.printf("\n%s\n", successMsg)
}

// locate prints where the party is, in degrees of latitude and longitude.
// C: seg_1944:42AC (C_1944_42AC)
func (g *Game) locate() {
	x, y := int(g.MapX), int(g.MapY)
	if g.MapZ != 0 {
		const levelScale = 2 // a dungeon level is a quarter of the surface on each side
		x <<= levelScale
		y <<= levelScale
	}
	lat := (y - equatorY) >> degreeShift
	long := (x - meridianX) >> degreeShift
	g.printf("\n%d{", abs(lat))
	if lat > 0 {
		g.printf("S")
	} else {
		g.printf("N")
	}
	g.printf(", %d{", abs(long))
	if long > 0 {
		g.printf("E\n")
	} else {
		g.printf("W\n")
	}
}

// Where the degrees start on the surface, and the tiles per degree.
const (
	equatorY    = 360
	meridianX   = 304
	degreeShift = 3
)

// windChange turns the wind.
// C: seg_1944:435B (C_1944_435B)
func (g *Game) windChange() {
	g.printf("Direction-")
	g.MouseMode = 0
	if g.getch() != cmdAdvance {
		g.printf("nowhere.\n")
	} else {
		g.printf("%s.\n", dirNames[g.AdvanceDir])
	}
	g.WindDir = int8(g.AdvanceDir)
}

// armageddonSpell destroys every creature but the Avatar and Lord British.
// C: seg_1944:43A3 (C_1944_43A3)
func (g *Game) armageddonSpell() {
	const avatar, lordBritish = 1, 5
	for n := range npcCount - 1 {
		if n != avatar && n != lordBritish {
			g.DeleteObj(n)
		}
	}
	g.computeView()
	g.OtherAnimations()
	g.armageddon = true
	g.PartySize = 1
}
