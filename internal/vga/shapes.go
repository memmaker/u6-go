package vga

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"

	"github.com/memmaker/u6-go/internal/host"
)

// Shape is a picture from a shape library (.SHP files, U6MCGA.PTR): runs of pixels placed
// around a hot spot. It reaches Left pixels left of the hot spot, Right pixels right of it,
// Up above and Down below.
type Shape struct {
	Right, Left, Up, Down int
	Runs                  []ShapeRun
}

// ShapeRun is a row of pixels starting X, Y pixels from the hot spot.
type ShapeRun struct {
	X, Y   int
	Pixels []byte
}

// Width and Height are the size of the shape's box.
func (s *Shape) Width() int  { return s.Left + s.Right + 1 }
func (s *Shape) Height() int { return s.Up + s.Down + 1 }

// Surface is a picture shapes are drawn on, clipped to the inclusive rectangle Left..Right,
// Top..Bottom (tSurface in MOUSE.H).
type Surface struct {
	Pixels                   []byte
	Width                    int
	Right, Left, Top, Bottom int
}

// ScreenSurface is the whole screen.
func (d *Driver) ScreenSurface() Surface {
	return Surface{Pixels: d.Screen[:], Width: ScreenWidth, Right: ScreenWidth - 1, Bottom: ScreenHeight - 1}
}

// BackBufferSurface is the whole back buffer.
func (d *Driver) BackBufferSurface() Surface {
	return Surface{Pixels: d.BackBuffer[:], Width: BackBufferWidth,
		Right: BackBufferWidth - 1, Bottom: BackBufferHeight - 1}
}

func (s *Surface) contains(x, y int) bool {
	return x >= s.Left && x <= s.Right && y >= s.Top && y <= s.Bottom
}

// DrawShape draws a shape on a surface with its hot spot at (x, y), clipped to the surface.
//
// GR_30 (driver offset 48). The mouse code calls it to draw the pointer.
// C: U6MCGA.DRV:30A4
func (d *Driver) DrawShape(s Surface, x, y int, shape *Shape) { drawShape(s, x, y, shape) }

// DrawShape draws a shape on the surface with its hot spot at (x, y), clipped to the surface.
// Every pixel of the shape's runs is drawn: shapes have no transparent colour.
func (s Surface) DrawShape(x, y int, shape *Shape) { drawShape(s, x, y, shape) }

func drawShape(s Surface, x, y int, shape *Shape) {
	for _, run := range shape.Runs {
		for i, c := range run.Pixels {
			if px, py := x+run.X+i, y+run.Y; s.contains(px, py) {
				s.Pixels[py*s.Width+px] = c
			}
		}
	}
}

// shapeBox calls f for every point of the shape's box at (x, y) inside the surface, with its
// index in a saved copy of the box.
func shapeBox(s Surface, x, y int, shape *Shape, f func(surfaceAt, savedAt int)) {
	for row := range shape.Height() {
		for col := range shape.Width() {
			if px, py := x-shape.Left+col, y-shape.Up+row; s.contains(px, py) {
				f(py*s.Width+px, row*shape.Width()+col)
			}
		}
	}
}

// SaveUnderShape copies the part of the surface a shape drawn at (x, y) would cover into saved,
// a box of shape.Width() x shape.Height() pixels.
//
// GR_0C (driver offset 12). The mouse code calls it before drawing the pointer.
// C: U6MCGA.DRV:3316
func (d *Driver) SaveUnderShape(s Surface, saved []byte, x, y int, shape *Shape) {
	shapeBox(s, x, y, shape, func(surfaceAt, savedAt int) { saved[savedAt] = s.Pixels[surfaceAt] })
}

// RestoreUnderShape puts back what SaveUnderShape saved.
//
// GR_0F (driver offset 15). The mouse code calls it to remove the pointer.
// C: U6MCGA.DRV:347B
func (d *Driver) RestoreUnderShape(s Surface, saved []byte, x, y int, shape *Shape) {
	shapeBox(s, x, y, shape, func(surfaceAt, savedAt int) { s.Pixels[surfaceAt] = saved[savedAt] })
}

var errBadShape = errors.New("bad shape")

// LoadShapes reads a packed shape library: after a 4-byte size, a table of 4-byte offsets of
// the shapes from the library's start (the first offset ends the table).
func LoadShapes(files fs.FS, name string) ([]*Shape, error) {
	lib, err := readPacked(files, name)
	if err != nil {
		return nil, err
	}
	const (
		offsetBytes = 4
		tableStart  = 4
	)
	if len(lib) < tableStart+offsetBytes {
		return nil, fmt.Errorf("%s: %w", name, errBadShape)
	}
	count := (int(binary.LittleEndian.Uint32(lib[tableStart:])) - tableStart) / offsetBytes
	shapes := make([]*Shape, count)
	for i := range shapes {
		at := tableStart + i*offsetBytes
		if at+offsetBytes > len(lib) {
			return nil, fmt.Errorf("%s: %w", name, errBadShape)
		}
		if shapes[i], err = decodeShape(lib, int(binary.LittleEndian.Uint32(lib[at:]))); err != nil {
			return nil, fmt.Errorf("%s shape %d: %w", name, i, err)
		}
	}
	return shapes, nil
}

// decodeShape reads a shape: four 16-bit extents (right, left, up, down), then runs. A run is
// a 16-bit header (0 ends the shape; bit 0 set: packed; the rest: pixel count), a 16-bit x and
// y, and the pixels. Packed pixels come in pieces of a byte (bit 0 set: a repeat; the rest:
// length) followed by the repeated pixel or the pixels.
// C: U6MCGA.DRV:30A4
func decodeShape(lib []byte, at int) (*Shape, error) {
	pos := at
	word := func() (int, error) {
		if pos+2 > len(lib) {
			return 0, errBadShape
		}
		v := int(int16(binary.LittleEndian.Uint16(lib[pos:])))
		pos += 2
		return v, nil
	}
	shape := &Shape{}
	for _, extent := range []*int{&shape.Right, &shape.Left, &shape.Up, &shape.Down} {
		v, err := word()
		if err != nil {
			return nil, err
		}
		*extent = v
	}
	for {
		header, err := word()
		if err != nil {
			return nil, err
		}
		if header == 0 {
			return shape, nil
		}
		count := int(uint16(header)) >> 1
		var run ShapeRun
		if run.X, err = word(); err != nil {
			return nil, err
		}
		if run.Y, err = word(); err != nil {
			return nil, err
		}
		if header&1 == 0 {
			if pos+count > len(lib) {
				return nil, errBadShape
			}
			run.Pixels = lib[pos : pos+count]
			pos += count
		} else {
			for len(run.Pixels) < count {
				if pos >= len(lib) {
					return nil, errBadShape
				}
				piece := int(lib[pos])
				n := piece >> 1
				pos++
				if piece&1 != 0 {
					if pos >= len(lib) {
						return nil, errBadShape
					}
					for range n {
						run.Pixels = append(run.Pixels, lib[pos])
					}
					pos++
				} else {
					if pos+n > len(lib) {
						return nil, errBadShape
					}
					run.Pixels = append(run.Pixels, lib[pos:pos+n]...)
					pos += n
				}
			}
		}
		shape.Runs = append(shape.Runs, run)
	}
}

// pointerState is the mouse pointer. The original mouse code (MOUSE.ASM) draws it on the
// screen with SaveUnderShape, DrawShape and RestoreUnderShape; here the host draws it over
// the frame instead.
type pointerState struct {
	shapes []*Shape
	index  int
	x, y   int
	shown  bool
}

// LoadPointers reads the mouse pointer shapes from U6MCGA.PTR.
func (d *Driver) LoadPointers() error {
	shapes, err := LoadShapes(d.files, "U6MCGA.PTR")
	d.pointer.shapes = shapes
	return err
}

// SetPointer selects the pointer shape (MOUSE_POINTER: 0 is the arrow, 1 the hourglass,
// from 2 the direction arrows).
func (d *Driver) SetPointer(index int) { d.pointer.index = index }

// MovePointer puts the pointer's hot spot at (x, y) on the screen.
func (d *Driver) MovePointer(x, y int) { d.pointer.x, d.pointer.y = x, y }

// ShowPointer shows or hides the pointer.
func (d *Driver) ShowPointer(shown bool) { d.pointer.shown = shown }

func (p *pointerState) addTo(frame *host.Frame) {
	if !p.shown || p.index < 0 || p.index >= len(p.shapes) {
		return
	}
	shape := p.shapes[p.index]
	box := Surface{Pixels: make([]byte, shape.Width()*shape.Height()), Width: shape.Width(),
		Right: shape.Width() - 1, Bottom: shape.Height() - 1}
	drawShape(box, shape.Left, shape.Up, shape)
	frame.Pointer = box.Pixels
	frame.PointerWidth, frame.PointerHeight = shape.Width(), shape.Height()
	frame.PointerHotX, frame.PointerHotY = shape.Left, shape.Up
	frame.PointerX, frame.PointerY = p.x, p.y
	frame.PointerShown = true
}
