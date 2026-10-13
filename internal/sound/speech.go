package sound

import (
	"encoding/binary"
	"fmt"
	"io/fs"

	"github.com/memmaker/u6-go/internal/data"
)

// The FM Towns version's voices: SPEECH/CHARnnn.SAM, one library per character of LZW
// packed 8-bit sign and magnitude samples, as Nuvie's ConverseSpeech plays them.

const (
	offsetBytes   = 4
	offsetMask    = 1<<24 - 1 // the low three bytes of an entry; the top one is a flag
	sampleSign    = 128
	sampleScale   = 1.0 / 128
	speechSamples = 4                                      // Nuvie stretches every 4 source samples
	speechOutputs = 11                                     // into 11 at 44.1 kHz
	speechStep    = float64(speechSamples) / speechOutputs // source samples per output sample
)

// speech is the voice queued for the output.
var speech []float32

// Speak queues sample number sample (from 1, as the scripts count) of character
// character's library, if Settings.Speech is on and the file is there.
func Speak(character, sample int) {
	mu.Lock()
	defer mu.Unlock()
	if !Settings.Speech || files == nil {
		return
	}
	pcm, err := loadSpeech(files, fmt.Sprintf("CHAR%d.SAM", character), sample-1)
	if err == nil {
		speech = append(speech, pcm...)
	}
}

// loadSpeech reads item index of the library name (in the folder or its SPEECH folder) as
// output samples.
func loadSpeech(fsys fs.FS, name string, index int) ([]float32, error) {
	lib, err := fs.ReadFile(fsys, name)
	if err != nil {
		lib, err = fs.ReadFile(fsys, "SPEECH/"+name)
	}
	if err != nil {
		return nil, err
	}
	entry := func(i int) int { return int(binary.LittleEndian.Uint32(lib[i*offsetBytes:])) & offsetMask }
	if len(lib) < offsetBytes {
		return nil, fs.ErrNotExist
	}
	count := entry(0) / offsetBytes
	if index < 0 || index >= count || count*offsetBytes > len(lib) {
		return nil, fs.ErrNotExist
	}
	start, end := entry(index), len(lib)
	for i := index + 1; i < count; i++ {
		if e := entry(i); e != 0 {
			end = e
			break
		}
	}
	if start == 0 || start >= end || end > len(lib) {
		return nil, fs.ErrNotExist
	}
	raw, err := data.Decompress(lib[start:end])
	if err != nil {
		return nil, err
	}
	return resampleSpeech(raw), nil
}

// resampleSpeech converts the sign and magnitude bytes to output samples, interpolating.
func resampleSpeech(raw []byte) []float32 {
	value := func(b byte) float64 {
		if b >= sampleSign {
			return -float64(b-sampleSign) * sampleScale
		}
		return float64(b) * sampleScale
	}
	var out []float32
	for at := 0.0; int(at)+1 < len(raw); at += speechStep {
		i, f := int(at), at-float64(int(at))
		out = append(out, float32(value(raw[i])*(1-f)+value(raw[i+1])*f))
	}
	return out
}
