package sound

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

// midiStreams hashes the MIDI the original U6ROLAND.DRV sends in each song's first
// recordedTicks ticks, captured in DOSBox-X (MPU-401 intelligent mode) with the driver
// freshly loaded and the song's MIDI.DAT instruments set as MUS_0851 does: one line per
// message, "tick status data..." in two-digit hex, ticks counted from 1.
var midiStreams = map[string]string{
	"ULTIMA.M":   "83a3f67c7cbc7351213a5219a9361e6d64bd04ad87e5bdc5dfe4def000fc3b31",
	"BOOTUP.M":   "4d961d68ae9567742fb3423f93d49440a73c217927e91982fb887e328a732f8a",
	"INTRO.M":    "801e823a35a102b76ae4a7c20b9ae3fd7cf644fe7ae7978d958860efad8fa206",
	"CREATE.M":   "f967b25d3d0f2fb2bedfa4aa2d9d78eb2c948cc5a3231480aeff8905056f8e6d",
	"FOREST.M":   "91ba3592d23ad0a4c9561c78de81a8995e7a5baa4432d613e2e557005cb5b286",
	"HORNPIPE.M": "fc05139e6c7ad739058908279d1ba7f369e3653050349051232c2a6da5a947bd",
	"ENGAGE.M":   "74a0d10eaaa043eb03918819d51820650996f33c5a1daf83bbd183a11a98f7b3",
	"STONES.M":   "b0c287216c3671258f3b2eba57b20b8a55f7da0c0b77feb1664e85704d4fac96",
	"DUNGEON.M":  "ac8801c8ca603f29285551c1072713aeff20d507eb4e1d421e6e0ca5421a50ae",
	"BRIT.M":     "686f564268547f123489ab947a50dac3a5b4bf4b72f129b9fa15757fa6be66a5",
	"GARGOYLE.M": "cf84e1cd87522f20a01afe5aec015caa0ea95087071567d97fad7a4c2d4d199d",
	"END.M":      "74356c12d4f15a7b1d180877f81a7bfdd463bfc7f6496b009f218e4b4a0e0b70",
}

const recordedTicks = 2300

// TestRolandMatchesDriver plays every song through the MT-32 driver and compares its MIDI
// with the original driver's, message by message and tick by tick.
func TestRolandMatchesDriver(t *testing.T) {
	fsys := os.DirFS(dataFS(t))
	defer func(out func(uint32)) { midiOut = out }(midiOut)
	for n, name := range Songs[1:] {
		b, err := loadSong(fsys, name)
		if err != nil {
			t.Fatal(name, err)
		}
		r := newRoland()
		r.instruments.load(fsys, n+1)
		p := newPlayer(b, r)
		var log strings.Builder
		tick := 0
		midiOut = func(msg uint32) {
			size := 3
			if msg%256/16*16 == midiProgram {
				size = 2
			}
			fmt.Fprint(&log, tick)
			for range size {
				fmt.Fprintf(&log, " %02x", msg%256)
				msg /= 256
			}
			log.WriteString("\n")
		}
		for tick = 1; tick <= recordedTicks; tick++ {
			p.tick()
		}
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(log.String()))); got != midiStreams[name] {
			t.Errorf("%s: the MIDI differs from U6ROLAND.DRV's", name)
		}
	}
}
