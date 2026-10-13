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
- [ ] Port GAME.EXE modules (world map, objects, UI/console, commands, combat, AI, magic, conversation, save/load)
  - [x] phase 1: init, map, view, scroll, movement, main loop (121/476 fns)
  - [ ] phase 2: commands, objects, inventory/portrait views, music hookup
  - [ ] phase 3: combat, AI, magic
  - [ ] phase 4: conversation, remaining TODOs
- [x] Mouse + keyboard input
- [x] Sound/music (AdLib driver port, internal/sound)
- [ ] Port U.EXE: title/intro/character creation
- [ ] Port END.EXE: ending
- [ ] Launcher flow (ULTIMA6 -> U -> GAME -> END) inside one Go program
- [ ] Runs from original data files natively
- [ ] Parity checks against DOSBox-X screenshots

## C. Publish
- [ ] -import SRC (folders, zip, ISO/BIN/CUE, GOG installers, loose files) + COVER.PNG
- [ ] wasm build works
- [ ] Registered in goplay (games.js, deploy.sh, README table)
- [ ] Deployed live and checked at https://ruzzoli.de/goplay/
