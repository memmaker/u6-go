package sound

import (
	"math"
	"testing"
)

func TestSoftFilterPassesBass(t *testing.T) {
	var s [2]float64
	y := 0.0
	for range outputRate / 10 {
		y = soften(1, &s)
	}
	if math.Abs(y-1) > 1e-6 {
		t.Fatalf("steady level after the filter: %v, want 1", y)
	}
}

func TestResampleSpeech(t *testing.T) {
	out := resampleSpeech([]byte{64, 64 + sampleSign})
	if out[0] != 0.5 || len(out) != 3 || out[2] >= 0 {
		t.Fatalf("resampled %v", out)
	}
}
