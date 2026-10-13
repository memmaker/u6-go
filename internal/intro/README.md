# internal/intro — U.EXE

A port of Ultima VI's `U.EXE` (source: `~/Projects/u6-re/src-u`): the boot intro (Origin
logo, the television, the storm, the moongate), the main menu, the introduction (the
gargoyles' altar), the acknowledgments and the character creation (name, sex, portrait,
the woods, the gypsy's questions).

Entry point: `intro.Run(fsys, saveDir) (Result, error)`. `StartGame` means the player
created a character or chose *Journey Onward* (U.EXE ran `GAME.EXE`); `Quit` means ctrl-Q.

## Files written for GAME.EXE

Only the character creation writes, into `saveDir/SAVEGAME`:

- If `SAVEGAME/OBJLIST` is missing, it first creates the saved game the way the installer
  did: `LZOBJBLK` unpacked into `OBJBLKAA`..`OBJBLKHH` (64 surface blocks, each a 16-bit
  count and 8-byte records, named `OBJBLK` + `'A'+i%8` + `'A'+i/8`), `LZDNGBLK` into
  `OBJBLKAI`..`OBJBLKEI` (5 dungeon blocks) followed by the 7283 bytes of `OBJLIST`.
  `MAP` is not written (GAME.EXE's own set up does that).
- `SAVEGAME/OBJLIST` (7283 bytes), as `SaveObjlist` wrote it. If byte 7282 is not 0 a
  game is in progress: nothing is written and passage 998 is shown (the creation then
  returns to the menu). Otherwise:

| offset | value |
|-------:|-------|
| 2305 | strength |
| 2561 | dexterity |
| 2817 | intelligence |
| 3074, 3075 | experience 370, low and high byte |
| 3585 | hit points 90 |
| 3840..3853 | the name, 14 raw bytes of the 16-byte buffer (the NUL and what followed it) |
| 4082 | level 3 |
| 5106 | magic: intelligence * 2 (as a byte) |
| 7161 | karma: 75, +3 per Honesty answer; kept from one creation to the next while U.EXE runs |
| 7281 | 1 female, 0 male |
| 7282 | portrait + 1 (portrait: 0..5 female, 6..11 male) |

## Deviations

- VGA only (no CGA/EGA/Tandy paths); disk prompts (`DISK_confirm`) are gone.
- *Transfer a Character* (from Ultima IV/V disks) is not ported; the menu comes back.
- The pointer is drawn by the host over the screen (`INTRO.PTR` shape 0); the original's
  save-under/redraw (`mouse_save`, `mouse_draw`, `ShowPage`) is not needed.
- Timing: the machine-speed calibration is fixed at a fast machine (`randomBetween(a, b)`
  returns `a`), frames are paced by host ticks (one wait for the retrace is one tick), and
  the timer callbacks (colour cycling, starfield) run on the host's 18.2 Hz timer without
  their wait for the retrace.
- The television static and the gypsy's balance took random bits from the bytes of the
  program's own code at a random offset; the port uses a pseudo random stream seeded by
  that offset.
- `rand` is Turbo C's, seeded from the clock.
