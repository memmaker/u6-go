package intro

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
)

// The saved game U.EXE leaves for GAME.EXE: SAVEGAME/OBJLIST with the new avatar.

const (
	saveGameDir     = "SAVEGAME"
	objlistName     = "OBJLIST"
	objlistSize     = 7283
	objblkPrefix    = "OBJBLK"
	surfaceBlocks   = 64
	dungeonBlocks   = 5
	blocksPerRow    = 8
	blockRecordSize = 8
	blockCountBytes = 2

	// OBJLIST offsets (the avatar is actor 1)
	strengthAt     = 2305
	dexterityAt    = 2561
	intelligenceAt = 2817
	experienceAt   = 3074
	hitPointsAt    = 3585
	nameAt         = 3840
	nameBytes      = 14
	levelAt        = 4082
	magicAt        = 5106
	karmaAt        = 7161
	femaleAt       = 7281
	portraitAt     = 7282
	maxByte        = 255
)

// errShortObjects is a broken LZOBJBLK or LZDNGBLK.
var errShortObjects = errors.New("intro: short initial object file")

// Permissions of the directories and files written.
const (
	dirMode  = 0o755
	fileMode = 0o644
)

// ensureSaveGame unpacks the initial objects into SAVEGAME when there is no OBJLIST, as
// the installer did on the original disks: the 64 surface blocks of LZOBJBLK, the 5 dungeon
// blocks of LZDNGBLK and the OBJLIST that follows them.
func (u *intro) ensureSaveGame() error {
	dir := filepath.Join(u.saveDir, saveGameDir)
	if _, err := os.Stat(filepath.Join(dir, objlistName)); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	split := func(raw []byte, first, count int) ([]byte, error) {
		for b := first; b < first+count; b++ {
			if len(raw) < blockCountBytes {
				return nil, errShortObjects
			}
			size := blockCountBytes + int(binary.LittleEndian.Uint16(raw))*blockRecordSize
			if len(raw) < size {
				return nil, errShortObjects
			}
			name := objblkPrefix + string(rune('A'+b%blocksPerRow)) + string(rune('A'+b/blocksPerRow))
			if err := os.WriteFile(filepath.Join(dir, name), raw[:size], fileMode); err != nil {
				return nil, err
			}
			raw = raw[size:]
		}
		return raw, nil
	}
	surface, err := u.readLZW("LZOBJBLK")
	if err != nil {
		return err
	}
	if _, err := split(surface, 0, surfaceBlocks); err != nil {
		return err
	}
	dungeons, err := u.readLZW("LZDNGBLK")
	if err != nil {
		return err
	}
	rest, err := split(dungeons, surfaceBlocks, dungeonBlocks)
	if err != nil {
		return err
	}
	if len(rest) < objlistSize {
		return errShortObjects
	}
	return os.WriteFile(filepath.Join(dir, objlistName), rest[:objlistSize], fileMode)
}

// saveObjlist writes the new avatar into SAVEGAME/OBJLIST, starting a new saved game when
// one is in progress.
// C: seg_0406:0006 (SaveObjlist)
func (u *intro) saveObjlist() error {
	path := filepath.Join(u.saveDir, saveGameDir, objlistName)
	if b, err := os.ReadFile(path); err == nil && len(b) > portraitAt && b[portraitAt] != 0 {
		// a game in progress: the new avatar starts a new game, where the original asked
		// for the installer to make a new character disk
		if err := os.RemoveAll(filepath.Join(u.saveDir, saveGameDir)); err != nil {
			return err
		}
	}
	if err := u.ensureSaveGame(); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) < objlistSize {
		return errShortObjects
	}
	c := &u.char
	b[strengthAt] = byte(c.strength)
	b[dexterityAt] = byte(c.dexterity)
	b[intelligenceAt] = byte(c.intelligence)
	b[magicAt] = byte(c.intelligence << 1)
	b[femaleAt] = 0
	if c.sex[0] == 'F' {
		b[femaleAt] = 1
	}
	b[karmaAt] = byte(c.karma)
	copy(b[nameAt:nameAt+nameBytes], c.name[:nameBytes])
	binary.LittleEndian.PutUint16(b[experienceAt:], uint16(c.experience))
	c.hitPoints = min(c.hitPoints, maxByte)
	b[hitPointsAt] = byte(c.hitPoints)
	b[levelAt] = byte(c.level)
	b[portraitAt] = byte(c.portrait + 1)
	return os.WriteFile(path, b[:objlistSize], fileMode)
}
