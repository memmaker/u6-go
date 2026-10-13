package data

import "testing"

// Castle British's objects are the 21st block of LZOBJBLK and region (2,2).
func TestPackedRegion(t *testing.T) {
	const castleBlock, castleRegion = 20, 2*8 + 2
	if r := PackedRegion(castleBlock); r != castleRegion {
		t.Fatalf("block %d: region %d, want %d", castleBlock, r, castleRegion)
	}
	seen := map[int]bool{}
	for b := range 64 {
		seen[PackedRegion(b)] = true
	}
	if len(seen) != 64 {
		t.Fatalf("%d distinct regions, want 64", len(seen))
	}
}
