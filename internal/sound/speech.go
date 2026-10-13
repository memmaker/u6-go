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
// character's library, if the file is there.
func Speak(character, sample int) {
	mu.Lock()
	defer mu.Unlock()
	if files == nil {
		return
	}
	pcm, err := loadSpeech(files, fmt.Sprintf("CHAR%d.SAM", character), sample-1)
	if err == nil {
		speech = append(speech, pcm...)
	}
}

// Speaking says whether a voice is queued.
func Speaking() bool {
	mu.Lock()
	defer mu.Unlock()
	return len(speech) > 0
}

// loadSpeech reads item index of the library name (in CD/SPEECH, SPEECH or the folder) as
// output samples.
func loadSpeech(fsys fs.FS, name string, index int) ([]float32, error) {
	lib, err := readFirst(fsys, "CD/SPEECH/"+name, "SPEECH/"+name, name)
	if err != nil {
		return nil, err
	}
	item, err := libraryItem(lib, index)
	if err != nil {
		return nil, err
	}
	raw, err := data.Decompress(item)
	if err != nil {
		return nil, err
	}
	return resample(raw, speechStep), nil
}

// libraryItem is item index of a library with 4-byte offsets (whose top byte is a flag).
func libraryItem(lib []byte, index int) ([]byte, error) {
	if len(lib) < offsetBytes {
		return nil, fs.ErrNotExist
	}
	entry := func(i int) int { return int(binary.LittleEndian.Uint32(lib[i*offsetBytes:])) & offsetMask }
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
	return lib[start:end], nil
}

// resample converts sign and magnitude bytes to output samples, step source samples per
// output sample, interpolating.
func resample(raw []byte, step float64) []float32 {
	value := func(b byte) float64 {
		if b >= sampleSign {
			return -float64(b-sampleSign) * sampleScale
		}
		return float64(b) * sampleScale
	}
	var out []float32
	for at := 0.0; int(at)+1 < len(raw); at += step {
		i, f := int(at), at-float64(int(at))
		out = append(out, float32(value(raw[i])*(1-f)+value(raw[i+1])*f))
	}
	return out
}

// readFirst reads the first of names that exists.
func readFirst(fsys fs.FS, names ...string) (contents []byte, err error) {
	for _, name := range names {
		if contents, err = fs.ReadFile(fsys, name); err == nil {
			return contents, nil
		}
	}
	return nil, err
}
