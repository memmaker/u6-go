package sound

import "math"

// Options are the port's optional enhancements, after ScummVM's Nuvie; the zero value is
// the original game. The command loads them from settings.json in the game folder.
type Options struct {
	// SoftSpeaker passes the PC speaker through a gentle resonant low-pass, taking the
	// edge off its square waves.
	SoftSpeaker bool `json:"soft_speaker"`
	// Speech plays the FM Towns voices (CHARnnn.SAM) where the conversation scripts mark
	// them; it needs the FM Towns CONVERSE.A/B, the DOS scripts have no marks.
	Speech bool `json:"speech"`
	// TalkMusic is what the music does in conversations: "" stops it (as the original),
	// "lower" plays it quieter, "keep" plays it on.
	TalkMusic string `json:"talk_music"`
	// NoCombatMusic keeps the area's song in combat instead of ENGAGE.M.
	NoCombatMusic bool `json:"no_combat_music"`
	// NoShipMusic keeps the area's song on ships instead of HORNPIPE.M.
	NoShipMusic bool `json:"no_ship_music"`
	// ShowStealing asks before taking an item that belongs to someone.
	ShowStealing bool `json:"show_stealing"`
	// FreeBalloonMovement steers the balloon where the player goes instead of with the wind.
	FreeBalloonMovement bool `json:"free_balloon_movement"`
	// DisplayedWindDir keeps the date and wind shown while aboard a ship, raft or balloon.
	DisplayedWindDir bool `json:"displayed_wind_dir"`
	// SkipIntro starts at the main menu without the boot intro.
	SkipIntro bool `json:"skip_intro"`
}

// Settings are the options in force; set them before Start.
var Settings Options

// The TalkMusic choices besides stopping.
const (
	TalkMusicLower = "lower"
	TalkMusicKeep  = "keep"
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

// The soft speaker filter: a resonant two-pole low-pass at the output rate.
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
