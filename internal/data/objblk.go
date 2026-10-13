package data

// PackedRegion returns the surface region (y*8+x) of the b-th block in LZOBJBLK. The file
// keeps the 8x8 regions in 2x2 groups, each group in reading order, so the bits of b are
// x0 y0 x1 x2 y1 y2 from the lowest.
func PackedRegion(b int) int {
	x := b&1 | b>>2&3<<1
	y := b>>1&1 | b>>4&3<<1
	return y*8 + x
}
