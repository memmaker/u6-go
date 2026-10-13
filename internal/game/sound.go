package game

// Music and the speaker sounds of the objects around the party (seg_2F1A).

import (
	"time"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// soundSources is how many of the nearest sounding objects are remembered (D_ECA2).
const soundSources = 4

// maxSounds is how many objects sound in one frame (D_28BE).
const maxSounds = 1

// silenceSpell is the spell effect that silences the world (SpellFx[14]).
const silenceSpell = 14

// musicAdapter says that a music card plays the songs (D_01CA != 0). The port always
// plays them on its OPL2, so the musicians' speaker melody is silent, as on an AdLib.
const musicAdapter = true

// scaleNotes are the frequencies of the musicians' scale (D_28C2).
var scaleNotes = [12]int{3116, 3497, 3926, 4159, 4406, 4668, 5240, 5882, 6231, 6602, 6995, 7851}

// melodyEnd ends a melody.
const melodyEnd = 255

// melodies are what the musicians play: notes of scaleNotes plus one, 0 a rest (D_2951).
var melodies = [2][]byte{
	{2, 6, 6, 0, 0, 2, 7, 7, 0, 0, 6, 11, 8, 11, 9, 6, 8, 7,
		2, 6, 0, 0, 0, 2, 7, 0, 0, 0, 6, 11, 8, 7, 8, 6, 0, 0,
		0, 7, 10, 10, 11, 7, 8, 10, 11, 7, 6, 8, 7, 6, 5, 3, 2, melodyEnd},
	{3, 4, 6, 0, 12, 0, 0, 0, 9, 9, 11, 9, 8, 9, 7, 0,
		2, 3, 4, 0, 11, 0, 0, 0, 8, 8, 9, 8, 7, 8, 6, 0,
		3, 4, 6, 0, 12, 0, 9, 0, 9, 0, 11, 9, 8, 9, 7, 7,
		8, 9, 11, 0, 7, 0, 8, 0, 6, 0, 9, 0, 0, 0, 0, 0, melodyEnd},
}

// music is the state of the songs and the sounding objects.
type music struct {
	// song is the song playing (MUS_MIndex).
	song int
	// bellRings counts the strokes of the bell still to sound (MUS_Bell).
	bellRings int
	// melody and melodyNote are the musicians' melody and its next note (MUS_Song,
	// MUS_SongIndex).
	melody, melodyNote int
	// the nearest sounding objects in view and their distances (D_ECA2, D_ECAC, D_ECAA)
	soundTypes    [soundSources]ObjectType
	soundDistance [soundSources]int
	soundCount    int
}

// sfx plays a speaker effect when the sound is on and waits while it sounds.
// C: seg_2F1A:0065 (MUS_0065)
func (g *Game) sfx(kind, param int) {
	if g.SoundOn == 0 {
		return
	}
	host.Wait(sound.Effect(kind, param))
}

// speaker waits while a queued speaker sound plays.
func speaker(d time.Duration) { host.Wait(d) }

// playSong starts song n (0 stops the music).
// C: seg_2F1A:091A (MUS_091A)
func (g *Game) playSong(n int) {
	if n == g.song {
		return
	}
	if g.song != 0 {
		sound.StopSong()
		g.song = 0
	}
	if n == 0 || g.SoundOn == 0 {
		return
	}
	_ = sound.Play(n)
	g.song = n
}

// playAreaSong plays the song that suits where the party is.
// C: seg_2F1A:09A8 (MUS_09A8)
func (g *Game) playAreaSong() {
	if g.SoundOn == 0 {
		return
	}
	const gargoyleLand = 5
	var n int
	switch g.MapZ {
	case gargoyleLand:
		n = sound.SongGargoyle
	case 0:
		n = [4]int{sound.SongStones, sound.SongUltima, sound.SongForest, sound.SongBrit}[g.rand(0, 3)]
	default:
		n = sound.SongDungeon
	}
	if t := g.object(int(g.Party[g.Active])).Type(); (t == ObjShip || t == ObjSkiff) && !sound.Settings.NoShipMusic {
		n = sound.SongHornpipe
	}
	if g.InCombat != 0 && !sound.Settings.NoCombatMusic {
		n = sound.SongEngage
	}
	g.playSong(n)
}

// addSoundSource remembers an object that sounds, keeping the nearest ones.
// C: seg_2F1A:043B (MUS_043B)
func (g *Game) addSoundSource(t ObjectType, distance int) {
	switch t {
	case ObjClock, ObjFountain, ObjBell, ObjFireplace, ObjStalagmite, ObjFire, ObjWaterWheel,
		ObjLogSaw, ObjMillstone, ObjCookFire, ObjFireField, ObjProtectionField, ObjVolcano,
		ObjMusician2:
	default:
		return
	}
	i := 0
	for i < g.soundCount && g.soundDistance[i] <= distance {
		i++
	}
	if i >= soundSources {
		return
	}
	for k := min(g.soundCount, soundSources-1); k > i; k-- {
		g.soundTypes[k], g.soundDistance[k] = g.soundTypes[k-1], g.soundDistance[k-1]
	}
	if g.soundCount < soundSources {
		g.soundCount++
	}
	g.soundTypes[i], g.soundDistance[i] = t, distance
}

// crackle sounds random clicks: of steps periods, one in odds sounds a tone from tone().
func crackle(steps, odds int, tone func() int, rand func(a, b int) int) {
	for range steps {
		if rand(0, odds) == 0 {
			sound.Tone(tone())
		} else {
			sound.NoTone()
		}
		speaker(sound.Delay(8, 1))
	}
	sound.NoTone()
}

// The ambient sounds go to the speaker even with the sound off, as OSILIB has no switch.

// playSoundSources sounds the nearest sounding objects, rings the clocks and the bells and
// waits a timer tick.
// C: seg_2F1A:0525 (MUS_0525)
func (g *Game) playSoundSources() {
	const (
		chimeHz, chimeSteps, chimeWidth, chimeDelta = 3116, 2000, 20000, -10
		tickHz, tacHz, tickLength                   = 3000, 2000, 3
		fireOdds, fireSteps                         = 7, 5
		fireLowHz, fireHighHz                       = 2000, 15000
		fieldOdds, fieldSteps                       = 15, 80
		fieldLowHz, fieldHighHz                     = 200, 1500
		chimePeriod, tickPeriod                     = 3, 15
		tacPhase                                    = 8
	)
	count := 0
	if g.SpellFx[silenceSpell] == 0 {
		for i := range g.soundCount {
			d := g.soundDistance[i]
			switch g.soundTypes[i] {
			case ObjClock:
				switch {
				case g.musicClock != 0 && g.frameCounter&chimePeriod == 0:
					speaker(sound.PlayWavedNote(chimeHz, 1, chimeSteps, chimeWidth, chimeDelta))
					count++
				case g.frameCounter&tickPeriod == 0:
					speaker(sound.PlayNote(tickHz, tickLength-d>>2))
					count++
				case g.frameCounter&tickPeriod == tacPhase:
					speaker(sound.PlayNote(tacHz, tickLength-d>>2))
					count++
				}
			case ObjBell:
				if g.bellRings != 0 && g.frameCounter&tickPeriod == 0 {
					speaker(sound.PlayWavedNote(chimeHz, 1, chimeSteps, chimeWidth, chimeDelta))
					count++
				}
			case ObjFireplace, ObjFire, ObjCookFire, ObjFireField:
				if g.rand(0, 3) == 0 {
					count++
					crackle(fireSteps, d+fireOdds, func() int { return g.rand(fireLowHz, fireHighHz) }, g.rand)
				}
			case ObjProtectionField:
				if g.rand(0, 1) == 0 {
					count++
					crackle(fieldSteps, fieldOdds, func() int { return g.rand(fieldLowHz, fieldHighHz-d<<8) }, g.rand)
				}
			case ObjFountain:
				const fountainStep, fountainLength, fountainHz = 10, 30, 25000
				speaker(sound.PlayNoise(fountainStep, fountainLength-d<<2, fountainHz-d<<11))
				count++
			case ObjWaterWheel:
				const wheelStep, wheelLength, wheelHz = 20, 60, 10000
				speaker(sound.PlayNoise(wheelStep, wheelLength-d<<2, wheelHz-d<<10))
				count++
			case ObjMusician2:
				melody := melodies[g.melody]
				if note := melody[g.melodyNote]; note != 0 && !musicAdapter {
					g.sfx(sound.EffectPluck, scaleNotes[note-1])
					count++
				}
				if g.melodyNote++; melody[g.melodyNote] == melodyEnd {
					g.melodyNote = 0
				}
			}
			if count >= maxSounds {
				break
			}
		}
	}
	if g.musicClock != 0 && g.frameCounter&chimePeriod == 0 {
		g.musicClock--
	}
	if g.bellRings != 0 && g.frameCounter&tickPeriod == 0 {
		if g.bellRings--; g.bellRings == 0 {
			g.setTileAnimation(int(g.BaseTile[ObjBell]), animStopped)
		}
	}
	g.osiDelay(1)
}

// setTileAnimation starts or stops the animation of a tile.
// C: seg_0A33:0049 (SetTileAnimation)
func (g *Game) setTileAnimation(tile int, state byte) {
	for i := range g.animCount {
		if int(g.animTile[i]) == tile {
			g.animState[i] = state
			return
		}
	}
}

// quietForTalk silences the music for a conversation, or lowers it or lets it play on as
// the TalkMusic setting asks.
func (g *Game) quietForTalk() {
	switch sound.Settings.TalkMusic {
	case sound.TalkMusicKeep:
	case sound.TalkMusicLower:
		sound.LowerMusic(true)
	default:
		g.playSong(0)
	}
}

// The voice libraries of the characters without their own (Nuvie's numbering).
const (
	guardVoice = 228
	wispVoice  = 229
)

// speechMark reads the digits of a "~P" voice mark (an FM Towns script's, pc after the P)
// and has npc n say that sample.
func (g *Game) speechMark(n int) {
	sample := 0
	for c := g.scriptByte(g.pc); c >= '0' && c <= '9'; c = g.scriptByte(g.pc) {
		sample = sample*10 + int(c-'0')
		g.pc++
	}
	switch g.object(n).Type() {
	case ObjGuard:
		n = guardVoice
	case ObjWisp:
		n = wispVoice
	}
	sound.Speak(n, sample)
}
