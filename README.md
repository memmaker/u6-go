# u6-go

Ultima VI: The False Prophet (Origin, 1990), the DOS VGA version, ported to Go and
[Ebiten](https://ebitengine.org): the intro and character creation (U.EXE), the game
(GAME.EXE) and the ending (END.EXE). It runs on the original game's data files; no game
data is in this repository.

## Playing

Download a build for Windows, macOS or Linux from the
[releases](https://github.com/memmaker/u6-go/releases), or build it yourself:

```sh
go run ./cmd/u6 -import /path/to/your/copy -data game   # once
go run ./cmd/u6 -data game
```

`-import SRC` finds Ultima VI in whatever you hand it: a plain folder, loose files, zip
files, CD-ROM images (ISO, BIN/CUE) or the GOG installer (`setup_*.exe` with its `.bin`
slices). It copies the game files into `-data` and exits.

## Sound

Music plays on an emulated AdLib. On macOS and Linux it can play on munt's MT-32 emulation
instead: install munt's libmt32emu and put the MT-32 ROMs in `roms/` (in the working
directory or next to the binary), or pass `-roms DIR`. `-import-cd` takes the FM Towns CD's voices, English conversations and sampled
sound effects from a disc image or folder.

## In the browser

The WebAssembly build (`GOOS=js GOARCH=wasm go build ./cmd/u6`) runs in
[goplay](https://ruzzoli.de/goplay/) on the player's own files.

## More

PORTING.md describes how the original program maps to the Go packages. The bundled
interface images in `internal/game/nuvie` come from the Nuvie project (GPL, see its
COPYING). Licensed under the GPL v3 (LICENSE).

## Releases

Pushing a `v*` tag builds Windows, macOS (universal) and Linux binaries and publishes them
as a GitHub release (`.github/workflows/release.yml`).
