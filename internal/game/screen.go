package game

import (
	"image"

	"github.com/memmaker/u6-go/internal/host"
)

// The port's screen follows the window's shape: on a wider (or taller) window than 4:3
// the host asks for a wider (taller) screen, and the map view grows to fill it, at the
// same pixel size. Everything that measures the view (sight, the pointer, what counts as
// on screen) reads its size from here.

// fitView sizes the view to the screen the host asks for: the least odd number of tiles
// that covers it, so the party stays in the middle. It returns whether the size changed.
func (g *Game) fitView() bool {
	width, height := host.FitWidth, host.FitHeight
	if width == g.screenWidth && height == g.screenHeight {
		return false
	}
	oldCentreX, oldCentreY := g.mapCentre()
	first := g.viewTilesX == 0
	tiles := func(pixels int) int {
		n := (pixels + tileSize - 1) / tileSize
		return min(n|1, maxViewTiles)
	}
	g.screenWidth, g.screenHeight = width, height
	g.viewTilesX, g.viewTilesY = tiles(width), tiles(height)
	g.viewCenterX, g.viewCenterY = g.viewTilesX/2, g.viewTilesY/2
	g.world = make([]byte, width*height)
	g.shown = make([]byte, width*height)
	centreX, centreY := g.mapCentre()
	if first {
		g.pointerCentreX, g.pointerCentreY = centreX, centreY
	} else {
		g.pointerCentreX += centreX - oldCentreX
		g.pointerCentreY += centreY - oldCentreY
		g.refreshArea()
	}
	return true
}

// viewCrop is how many pixels of the back buffer's view are left out at the left and top
// of the screen.
func (g *Game) viewCrop() (int, int) {
	return (g.viewTilesX*tileSize - g.screenWidth) / 2, (g.viewTilesY*tileSize - g.screenHeight) / 2
}

// viewPixels is the part of the back buffer's view that is shown.
func (g *Game) viewPixels() image.Rectangle {
	cropX, cropY := g.viewCrop()
	return image.Rect(cropX, cropY, cropX+g.screenWidth, cropY+g.screenHeight)
}
