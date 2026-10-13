# Ultima VI: byte-exact decompilation + Go port — checklist

## A. Byte-exact executables (~/Projects/u6-re)
- [x] GAME.EXE byte-identical to shipped file (2026-10-13)
- [x] Notes: rules added to ~/notes/c-to-go.md
- [x] END.EXE byte-identical (2026-10-13, u6-re 06f25b3)
- [x] U.EXE byte-identical (2026-10-13, u6-re 49b8dd5)

## B. Go port (~/Projects/u6-go), rules in ~/notes/c-to-go.md
- [x] Repo skeleton: go.mod, cmd/u6, host (Ebiten lockstep), gameimport (-import)
- [x] Data layer: file loading, LZW decompression [x], tiles/palette/fonts [x]
- [x] Graphics: VGA driver semantics (U6MCGA.DRV) in Go
- [x] Game state with readable names (globals, objects, NPCs)
- [x] Port GAME.EXE modules (world map, objects, UI/console, commands, combat, AI, magic, conversation, save/load)
  - [x] phase 1: init, map, view, scroll, movement, main loop (121/476 fns)
  - [x] phase 2: commands, objects, inventory/portrait views, travel, eggs, music hookup (280/476 fns)
  - [x] phase 3: combat, AI, magic (NPC spell casting in combat included)
  - [x] conversation engine (talkdr)
  - [x] phase 4: remaining TODOs: no `TODO(port)` left; 439/476 fns (the other 37 are DOS disk/mouse/driver/cfg glue, the chunk cache and dead code; see internal/game/PORT_STATUS.md)
- [x] Mouse + keyboard input
- [x] Sound/music (AdLib driver port, internal/sound)
- [x] Port U.EXE: title/intro/character creation (internal/intro)
- [x] Port END.EXE: ending (internal/ending)
- [x] Launcher flow (ULTIMA6 -> U -> GAME -> END) inside one Go program
- [x] Runs from original data files natively and in the browser (intro → creation → game)
- [ ] Parity checks against DOSBox-X screenshots

## C. Publish
- [x] -import SRC (folders, zip, ISO/BIN/CUE, floppy images, GOG installers, loose files) + COVER.PNG
- [x] wasm build works
- [x] Registered in goplay (games.js, deploy.sh, README table)
- [x] Deployed live at https://ruzzoli.de/goplay/ (2026-10-13; redeploy after each phase)
