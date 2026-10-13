package vga

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/memmaker/u6-go/internal/data"
	"github.com/memmaker/u6-go/internal/host"
)

func dataDir(t *testing.T) string {
	dir := os.Getenv("U6DATA")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Projects/u6-re/gog/inst")
	}
	if _, err := os.Stat(filepath.Join(dir, "MAPTILES.VGA")); err != nil {
		t.Skip("no game data in", dir)
	}
	return dir
}

// TestSineMatchesDriver checks the computed sine table against the one in U6MCGA.DRV.
func TestSineMatchesDriver(t *testing.T) {
	packed, err := os.ReadFile(filepath.Join(dataDir(t), "U6MCGA.DRV"))
	if err != nil {
		t.Fatal(err)
	}
	driver, err := data.Decompress(packed)
	if err != nil {
		t.Fatal(err)
	}
	const sineTableOffset = 9956
	for i, want := range driver[sineTableOffset : sineTableOffset+angleSteps] {
		if sine[i] != int(int8(want)) {
			t.Fatalf("sine[%d] = %d, driver has %d", i, sine[i], int8(want))
		}
	}
}

func TestRender(t *testing.T) {
	files := os.DirFS(dataDir(t))
	d := New(files)
	if err := d.LoadTiles(); err != nil {
		t.Fatal(err)
	}
	if err := d.LoadPointers(); err != nil {
		t.Fatal(err)
	}
	font, err := LoadFont(files)
	if err != nil {
		t.Fatal(err)
	}
	d.Font = font
	palette, err := LoadPalette(files)
	if err != nil {
		t.Fatal(err)
	}
	d.Palette = *palette
	paper, err := LoadPixmap(files, "PAPER.BMP")
	if err != nil {
		t.Fatal(err)
	}
	if paper.Width != ScreenWidth || paper.Height != ScreenHeight {
		t.Fatalf("paper is %dx%d", paper.Width, paper.Height)
	}

	d.DrawPixmap(paper, 0, 0)
	// the map view: grass, water, coasts, then objects (transparent and pixel block tiles)
	const viewTiles = 11 // the original's map view
	for i := range viewTiles * viewTiles {
		x, y := i%viewTiles*tileSize, i/viewTiles*tileSize
		d.DrawTileOffscreen(i*3, x, y) // map tiles and the first object tiles
	}
	d.AnimateCoasts()
	d.DrawTileOffscreen(residentTiles+400, 40, 40)
	d.DrawTileOffscreen(residentTiles+400+1<<silhouetteShift, 72, 40)
	d.MakeRotatedSprite(8, residentTiles+400, angleSteps/8, 127, 127)
	d.DrawSprite(8, 112, 48)
	d.CopyBackBufferToScreen(8, 8, 167, 167, 8, 8)
	for i := range 9 {
		d.DrawTile(400+i, 8+i*tileSize, 176)
	}
	for i, ch := range []byte("Ultima VI") {
		d.DrawChar(ch, 176+i*charSize, 8)
	}
	d.SetColor(12)
	d.FillRect(176, 20, 300, 22)
	d.MovePointer(250, 100)
	d.ShowPointer(true)

	frame := d.Frame()
	img := image.NewRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	level := func(v byte) uint8 { return uint8(int(v) * 255 / 63) }
	rgb := func(c byte) color.RGBA {
		p := frame.Palette[int(c)*3:]
		return color.RGBA{level(p[0]), level(p[1]), level(p[2]), 255}
	}
	for i, c := range frame.Pixels {
		img.Set(i%ScreenWidth, i/ScreenWidth, rgb(c))
	}
	for i, c := range frame.Pointer {
		if c != 0 {
			img.Set(frame.PointerX-frame.PointerHotX+i%frame.PointerWidth,
				frame.PointerY-frame.PointerHotY+i/frame.PointerWidth, rgb(c))
		}
	}
	out := filepath.Join(t.TempDir(), "render.png")
	if dir := os.Getenv("VGA_PNG_DIR"); dir != "" {
		out = filepath.Join(dir, "render.png")
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	t.Log("wrote", out)
}

// TestFadeInLargestScreen fades in the largest screen the port shows: every pixel arrives.
func TestFadeInLargestScreen(t *testing.T) {
	d := New(nil)
	for i := range d.BackBuffer {
		d.BackBuffer[i] = 1
	}
	const w, h = host.MaxScreenWidth, host.MaxScreenHeight
	view := make([]byte, w*h)
	d.FadeIn(view, w, 0, 0, 0, 0, w-1, h-1)
	for i, c := range view {
		if c != 1 {
			t.Fatalf("pixel %d not faded in", i)
		}
	}
}
