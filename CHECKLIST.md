# Ultima VI: byte-exact decompilation + Go port — checklist

## A. Byte-exact executables (~/Projects/u6-re)
- [x] GAME.EXE byte-identical to shipped file (2026-10-13)
- [x] Notes: rules added to ~/notes/c-to-go.md
- [ ] END.EXE byte-identical (agent running)
- [ ] U.EXE byte-identical (agent running)

## B. Go port (~/Projects/u6-go), rules in ~/notes/c-to-go.md
- [ ] Repo skeleton: go.mod, cmd/u6, host (Ebiten lockstep), gameimport (-import)
- [ ] Data layer: file loading, LZW decompression, tiles/palette/fonts
- [ ] Graphics: VGA driver semantics (U6MCGA.DRV) in Go
- [ ] Game state with readable names (globals, objects, NPCs)
- [ ] Port GAME.EXE modules (world map, objects, UI/console, commands, combat, AI, magic, conversation, save/load)
- [ ] Mouse + keyboard input
- [ ] Sound/music (AdLib driver port)
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
