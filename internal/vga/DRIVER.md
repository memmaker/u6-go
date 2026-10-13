# U6MCGA.DRV entries

GAME.EXE calls the driver through `D_ECB8` with the entry offset (gr.h `GR_xx`). Driver
calls use the Pascal convention. Coordinates are pixels; "back buffer" is the 176x176
picture (game `ScratchBuf`, `tGr._1c`) in which the map view is composed at (8,8)..(167,167).

| GR | offset | Go | arguments | semantics |
|---|---|---|---|---|
| 00 | 0 | `New(fs)` | mode (ignored) | set mode 19, return tGr; the game's file callbacks become an `fs.FS` |
| 03 | 3 | `Shutdown()` | | restore the text mode (no-op) |
| 06 | 6 | `SetColor(c)` | colour | colour of PutPixel/FillRect |
| 09 | 9 | `PutPixel(x, y)` | | screen, or back buffer if `Offscreen`; XOR if `XorMode` |
| 0C | 12 | `SaveUnderShape(s, saved, x, y, shape)` | surface, buffer, hot spot, shape lib+index | save what a shape would cover (mouse code only) |
| 0F | 15 | `RestoreUnderShape(...)` | same | put it back (mouse code only) |
| 12 | 18 | `FillRect(x0, y0, x1, y1)` | inclusive corners, any order | like PutPixel |
| 15 | 21 | `ScrollRectVertically(x0, y0, x1, y1, dy)` | dy < 0: up | screen only; game uses dy = -8 (text scroll) |
| 18 | 24 | `DrawPixmap(p, x, y)` | tPixmap | opaque, screen |
| 1B | 27 | `Unused1B(a, b)` | | empty |
| 1E | 30 | `DrawChar(ch, x, y)` | | 8x8 from `Font`, `TextForeground` (tGr._0a) on `TextBackground` (_0b), screen |
| 21 | 33 | `Unused21()` | | empty |
| 24 | 36 | `InitPalette()` | | empty on VGA (the game loads U6PAL itself) |
| 27 | 39 | `AnimatePalette(step)` | | empty on VGA (the game cycles the DAC itself) |
| 2A | 42 | `LoadTiles()` | memory block | load MAPTILES/TILEINDX/MASKTYPE/OBJTILES/ANIMMASK; sets tGr._20 (non-null = ok) |
| 2D | 45 | `DrawTile(tile, x, y)` | | tile on the screen |
| 30 | 48 | `DrawShape(s, x, y, shape)` | surface, hot spot, shape lib+index | mouse pointer drawing |
| 33 | 51 | `MakeRotatedSprite(slot, tile, angle, px, py)` | angle 0..127, pivot in 1/16 px (game: 127,127) | rotated tile into sprite slot 0..15 |
| 36 | 54 | `DrawSprite(slot, x, y)` | centre | back buffer, at (x-8, y-8), 255 transparent |
| 39 | 57 | `SaveUnderSprite(slot, x, y)` | centre | copy back buffer 16x16 at (x-8, y-8) into slot |
| 3C | 60 | `Unused3C()` | | empty |
| 3F | 63 | `AnimateCoasts()` | | rebuild coast tiles 16..47 from water tiles + ANIMMASK |
| 42 | 66 | `DrawTileOffscreen(tile, x, y)` | tile bits 11-12: silhouette | tile on the back buffer; silhouette draws black pixels in colour 11/12/9 |
| 45 | 69 | `CopyBackBufferToScreen(x0, y0, x1, y1, toX, toY)` | inclusive | x rounded to pixel pairs |
| 48 | 72 | `FadeIn(x0, y0, x1, y1)` | inclusive | back buffer to the same place on screen, dissolving (LFSR order) |
| 4B | 75 | `SetTileFrame(tile, frame)` | | tile shows frame's picture (frame < 512 only) |
| 4E | 78 | `Vortex()` | | swirl of the map view (C_2FC1_1BCE) |

## Shared structure (tGr) fields

| field | Go |
|---|---|
| `_06` font | `Font` (`LoadFont`, U6.CH) |
| `_0a`, `_0b` | `TextForeground`, `TextBackground` |
| `_0c` | `XorMode` |
| `_0d`, `_0e` | not read by this driver (`_0e`, tile animation on/off, can be dropped) |
| `_0f` | `Offscreen` |
| `_10` disk prompt, `_24`..`_30` file callbacks | the `fs.FS` given to `New` |
| `_1c` back buffer | `BackBuffer` |
| `_20` tile memory | `LoadTiles` error |

## Files

The driver reads, through the game's callbacks, only: `maptiles.vga` (packed), `tileindx.vga`,
`masktype.vga` (packed: 2048 kinds, 1536 object tile groups, 192 group offsets),
`animmask.vga` (packed) and `objtiles.vga` (groups paged in on demand). ANIMDATA, TILEFLAG,
U6PAL, U6.CH, PAPER.BMP and U6MCGA.PTR are read by GAME.EXE; loaders for the ones that are
graphics are here: `LoadPalette`, `LoadFont`, `LoadPixmap`, `LoadShapes` / `LoadPointers`.

## Not driver entries

- Palette: the game sets the DAC (`Driver.Palette`) from U6PAL and animates it (PaletteAnimation).
- Mouse pointer: MOUSE.ASM draws U6MCGA.PTR shapes with GR_0C/30/0F. Here `SetPointer`,
  `MovePointer`, `ShowPointer` feed the host's pointer overlay in `Frame()` instead.
- `Present`: set it to `host.Yield` so FadeIn and Vortex animate.
