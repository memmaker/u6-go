package game

import (
	"image"
	"math"

	"github.com/memmaker/u6-go/internal/sound"
)

// The port's lighting options, after Nuvie: smooth lighting shades the dark in gradients
// between the tiles' light levels instead of dithered shade tiles, and map tile lighting
// lets glowing map tiles light their surroundings like lamps.

// lightMapTiles gives each area place the light of its map tile, unless an object there
// already gives more.
func (g *Game) lightMapTiles() {
	for y := lightFirst; y < lightLast; y++ {
		for x := lightFirst; x < lightLast; x++ {
			at := y*areaSize + x
			flags := &g.areaFlags[at]
			if light := g.tileLight(int(g.AreaTiles[at])); light != 0 && int(*flags&areaLightMask) < light {
				*flags = *flags&^areaLightMask | byte(light)
			}
		}
	}
}

// smoothShade returns how bright each screen pixel shows with smooth lighting, or nil
// when it is off, the world is not shown or all of the view is lit. The light of each view
// tile is at its middle and blends to its neighbours'; panels and scrolls (and whatever was
// drawn over the world) show at full light.
func (g *Game) smoothShade(panels []panel, scrolls []image.Rectangle) []byte {
	if !sound.Settings.SmoothLighting || !g.worldShown || g.xray {
		return nil
	}
	lightAt := func(i, j int) float64 {
		i = max(0, min(i, g.viewTilesX-1))
		j = max(0, min(j, g.viewTilesY-1))
		switch g.tileView[j][i] {
		case tileHidden: // black anyway: do not darken the places seen next to it
			return 1
		case tileDarkness:
			return 0
		}
		return float64(min(g.areaLight[(j+g.ViewY)*areaSize+i+g.ViewX], fullLight)) / fullLight
	}
	dark := false
	for j := range g.viewTilesY {
		for i := range g.viewTilesX {
			dark = dark || lightAt(i, j) < 1
		}
	}
	if !dark {
		return nil
	}
	shade := make([]byte, g.screenWidth*g.screenHeight)
	cropX, cropY := g.viewCrop()
	for y := range g.screenHeight {
		// the place between tile middles, in tiles
		fy := float64(y+cropY)/tileSize - 0.5
		j := int(math.Floor(fy))
		dy := fy - float64(j)
		for x := range g.screenWidth {
			fx := float64(x+cropX)/tileSize - 0.5
			i := int(math.Floor(fx))
			dx := fx - float64(i)
			top := lightAt(i, j)*(1-dx) + lightAt(i+1, j)*dx
			bottom := lightAt(i, j+1)*(1-dx) + lightAt(i+1, j+1)*dx
			shade[y*g.screenWidth+x] = byte((top*(1-dy) + bottom*dy) * math.MaxUint8)
		}
	}
	lit := func(r image.Rectangle) {
		r = r.Intersect(image.Rect(0, 0, g.screenWidth, g.screenHeight))
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				shade[y*g.screenWidth+x] = math.MaxUint8
			}
		}
	}
	for _, p := range panels {
		lit(p.onScreen())
	}
	for _, s := range scrolls {
		lit(s)
	}
	for i := range shade {
		if g.shown[i] != g.world[i] {
			shade[i] = math.MaxUint8
		}
	}
	return shade
}
