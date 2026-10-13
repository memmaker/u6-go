package sound

import "math"

// Options are the port's optional enhancements, after ScummVM's Nuvie; the zero value is
// the original game but for the lowered conversation music. The command loads them from settings.json in the game folder.
type Options struct {
	// TalkMusic is what the music does in conversations: "" or "lower" plays it quieter,
	// "stop" stops it (as the original), "keep" plays it on.
	TalkMusic string `json:"talk_music"`
	// ShowStealing asks before taking an item that belongs to someone.
	ShowStealing bool `json:"show_stealing"`
	// FreeBalloonMovement steers the balloon where the player goes instead of with the wind.
	FreeBalloonMovement bool `json:"free_balloon_movement"`
	// DisplayedWindDir keeps the date and wind shown while aboard a ship, raft or balloon.
	DisplayedWindDir bool `json:"displayed_wind_dir"`
	// SkipIntro starts at the main menu without the boot intro.
	SkipIntro bool `json:"skip_intro"`
	// SmoothLighting shades the dark in gradients instead of dithered tiles.
	SmoothLighting bool `json:"smooth_lighting"`
	// MapTileLighting lets glowing map tiles (lava, hearths) light their surroundings, as
	// lamps and fires already do.
	MapTileLighting bool `json:"map_tile_lighting"`
	// SquarePixels shows square pixels instead of the CRT's 4:3 picture.
	SquarePixels bool `json:"square_pixels"`
}

// Settings are the options in force; set them before Start.
var Settings Options

// The TalkMusic choices besides lowering.
const (
	TalkMusicStop = "stop"
	TalkMusicKeep = "keep"
)

// loweredMusicGain is the music's level while lowered for a conversation.
const loweredMusicGain = 0.3

var musicLowered bool

// LowerMusic plays the music quieter (true) or at its level again.
func LowerMusic(on bool) {
	mu.Lock()
	musicLowered = on
	mu.Unlock()
}

// The speaker cone filter: a resonant two-pole low-pass at the output rate.
const (
	softCutoffHz  = 4000
	softResonance = 1.2 // the filter's Q: a little above a flat 0.707, a soft peak at the cutoff
)

// softFilter is its biquad (b0 b1 b2 a1 a2), set up by init.
var softFilter [5]float64

func init() {
	w := 2 * math.Pi * softCutoffHz / outputRate
	alpha := math.Sin(w) / (2 * softResonance)
	a0 := 1 + alpha
	b1 := (1 - math.Cos(w)) / a0
	softFilter = [5]float64{b1 / 2, b1, b1 / 2, -2 * math.Cos(w) / a0, (1 - alpha) / a0}
}

// soften runs one speaker sample through softFilter with state s (transposed direct form II).
func soften(x float64, s *[2]float64) float64 {
	c := softFilter
	y := c[0]*x + s[0]
	s[0] = c[1]*x - c[3]*y + s[1]
	s[1] = c[2]*x - c[4]*y
	return y
}
