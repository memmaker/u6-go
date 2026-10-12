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
