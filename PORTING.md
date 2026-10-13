# Porting contract

Source of truth: `~/Projects/u6-re/src` (GAME.EXE, byte-exact), `src-u` (U.EXE), `src-end`
(END.EXE), and the original data in `~/Projects/u6-re/gog/inst` (tests take `U6DATA`).
Read `/Users/felix/notes/c-to-go.md` first; its rules #0–#6 are binding:
behaviour parity, no far pointers or memory emulation, no magic numbers or hex, readable
names, idiomatic Go, obvious bugs may be fixed (mark them `// fix:`).

## Packages

| package | replaces | notes |
|---|---|---|
| `internal/host` | DOS, BIOS, timer IRQ | lockstep Ebiten host: `Yield`, `Wait`, `KeyQueue`, mouse |
| `internal/gameimport` | — | `-import` (shared with other ports, do not edit) |
| `internal/data` | OSI_FILE, INFLATE | `Decompress` (LZW) and file loaders |
| `internal/vga` | U6MCGA.DRV (the `GR_xx` entries) | 320x200 screen + back buffer, palette, tiles, font, pixmaps, pointer; produces `host.Frame` |
| `internal/sound` | U6ADLIB.DRV, OSISND, PLAYNOTE | `.M` music on an OPL2 emulator, PC speaker effects |
| `internal/game` | GAME.EXE | one package, files named by topic (not by segment) |
| `internal/intro` | U.EXE | title, intro, character creation |
| `internal/ending` | END.EXE | the ending |
| `cmd/u6` | ULTIMA6.EXE | launcher flow: intro → game → ending |

## Conventions

- C globals become fields of a typed state struct (or package variables for the driver
  state), never byte arrays addressed by offset. Saved files are read and written with
  explicit field-by-field (de)serialisers that keep the original file layout.
- Every blocking wait (`while(!kbhit())`, page flips, delays) calls `host.Yield`/`host.Wait`.
- Every function keeps a comment `// C: seg_XXXX:YYYY (C_XXXX)` naming its original, so
  parity can be audited; that comment is the only place hex labels may appear.
- Constants get names (`const tileSize = 16`), with the original value written in decimal.
- One small test per non-trivial parser or algorithm, against the real data files.

## In-game UI (ui-redesign)

The world view fills the screen; the panels are layered over it as scrolls
(`internal/game/layout.go`, screenshots in `docs/ui/`). Hot keys:

- Ctrl+D shows or hides the date and wind scroll.
- Ctrl+L shows or hides the message log. Prompts show it while they wait.
- The screen takes the window's shape: in fullscreen and in a browser window of any shape
  the view widens (or heightens) to fill it, up to 736 pixels, at the same uniform scale and
  CRT pixel shape; the panels keep to their edges (`internal/game/screen.go`,
  `docs/ui/widescreen_*.png`).
- Ctrl+F picks the party formation: Default (the original's), Follow (a single line) or
  Triangle (the Avatar in front, widening rows behind). It lasts for the session.
- Touching the bottom edge with the mouse slides in the command icons.
- In conversations, the arrow keys pick a topic (the mouse wheel and Up/Down scroll a long list) and Enter asks it; typing still works.
- Ctrl+O opens the game menu (after Nuvie's): Save, Load, Quick save, Quick load, Load
  latest, Settings (the toggles of `settings` in `internal/game/gamemenu.go`; other options
  append theirs there), Quit. F11 quick saves without asking (a copy of SAVEGAME in
  QUICKSAV), F12 quick loads it, Shift+F12 loads the latest save (`docs/ui/gamemenu.png`).
- Shift+F1..F9 opens or closes a party member's inventory window; several can be open and
  each moves by dragging its name row (`internal/game/windows.go`, `docs/ui/windows.png`).
  A click in a window turns the status area to that member first.
- Things are dragged with the mouse, each drag running the command that does the same:
  map → an inventory is Get (by the active member), inventory → map is Drop (the active
  member's things), onto another member's window is Move, onto a container in the grid is
  Move into it, onto the doll is Ready, a worn thing onto the grid is Unready.
