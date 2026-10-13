package sound

import (
	"io/fs"
	"math/rand/v2"

	"github.com/memmaker/u6-go/internal/data"
)

// The FM Towns version's sampled sound effects, played instead of the speaker's when the CD's
// SOUNDS1.DAT (an LZW packed library) and SOUNDS2.DAT (a plain one) are under CD/, as
// Nuvie's TownsSfxManager plays them: 8-bit sign and magnitude samples at 14.7 kHz.

const (
	townsRate         = 14700
	townsStep         = float64(townsRate) / outputRate
	townsSounds1Count = 12 // the samples of SOUNDS1.DAT; SOUNDS2.DAT's follow them
)

// The FM Towns samples (Nuvie's numbers).
const (
	TownsBlocked         = 0
	TownsSwing           = 2
	TownsRubberDuck      = 3
	TownsHit             = 4
	TownsFire            = 6 // and the next two: the fire crackles in turn at random
	townsFireSamples     = 3
	TownsMissile         = 9
	TownsBrokenGlass     = 12
	TownsBell            = 13
	TownsExplosion       = 16
	TownsFountain        = 46
	TownsProtectionField = 47
	TownsWaterWheel      = 48
)

// townsEffects are the samples played for the speaker effects that have one.
var townsEffects = map[int]int{
	EffectNote:       TownsBlocked, // only the blocked move plays it
	EffectNoise:      TownsHit,
	EffectSwing:      TownsSwing,
	EffectMissile:    TownsMissile,
	EffectProjectile: TownsMissile,
	EffectWhoosh:     TownsBrokenGlass, // a mirror or a thrown object breaks
	EffectSqueak:     TownsRubberDuck,
	EffectImpact:     TownsExplosion,
}

var (
	townsLibraries [2][]byte // SOUNDS1.DAT unpacked and SOUNDS2.DAT, read on first use
	townsLoaded    bool
	samples        []float32 // the sample playing
)

// townsLibrary reads the two libraries once; nil without the CD's files.
func townsLibrary() [2][]byte {
	if !townsLoaded && files != nil {
		townsLoaded = true
		if packed, err := fs.ReadFile(files, "CD/SOUNDS1.DAT"); err == nil {
			townsLibraries[0], _ = data.Decompress(packed)
		}
		townsLibraries[1], _ = fs.ReadFile(files, "CD/SOUNDS2.DAT")
	}
	return townsLibraries
}

// TownsSounds says whether the FM Towns samples are there.
func TownsSounds() bool {
	mu.Lock()
	defer mu.Unlock()
	libs := townsLibrary()
	return libs[0] != nil && libs[1] != nil
}

// PlaySample starts FM Towns sample n unless one is still playing; false without the CD's
// files. It does not wait: like Nuvie, the game runs on while it sounds.
func PlaySample(n int) bool {
	mu.Lock()
	defer mu.Unlock()
	libs := townsLibrary()
	if libs[0] == nil || libs[1] == nil {
		return false
	}
	if len(samples) > 0 {
		return true
	}
	if n >= TownsFire && n < TownsFire+townsFireSamples {
		n = TownsFire + rand.IntN(townsFireSamples)
	}
	lib, index := libs[0], n
	if n >= townsSounds1Count {
		lib, index = libs[1], n-townsSounds1Count
	}
	if raw, err := libraryItem(lib, index); err == nil {
		samples = resample(raw, townsStep)
	}
	return true
}
