package gameimport

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
)

// listFAT lists a FAT12/FAT16 disk image (floppy .img/.ima files and the like). Not a FAT
// image: no files, no error.
func listFAT(r io.ReaderAt, size int64) ([]file, error) {
	var boot [512]byte
	if _, err := r.ReadAt(boot[:], 0); err != nil || boot[510] != 0x55 || boot[511] != 0xAA {
		return nil, nil
	}
	le := binary.LittleEndian
	sectorSize := int64(le.Uint16(boot[11:]))
	perCluster := int64(boot[13])
	reserved := int64(le.Uint16(boot[14:]))
	fats := int64(boot[16])
	rootEntries := int64(le.Uint16(boot[17:]))
	totalSectors := int64(le.Uint16(boot[19:]))
	if totalSectors == 0 {
		totalSectors = int64(le.Uint32(boot[32:]))
	}
	fatSectors := int64(le.Uint16(boot[22:]))
	if sectorSize < 128 || sectorSize > 4096 || perCluster == 0 || fats == 0 || fatSectors == 0 ||
		totalSectors*sectorSize > size+sectorSize {
		return nil, nil
	}
	fat := make([]byte, fatSectors*sectorSize)
	if _, err := r.ReadAt(fat, reserved*sectorSize); err != nil {
		return nil, nil
	}
	rootStart := (reserved + fats*fatSectors) * sectorSize
	rootBytes := rootEntries * 32
	dataStart := rootStart + (rootBytes+sectorSize-1)/sectorSize*sectorSize
	clusterBytes := perCluster * sectorSize
	clusters := (totalSectors*sectorSize-dataStart)/clusterBytes + 2
	fat12 := clusters < 4085

	next := func(c int64) int64 {
		if fat12 {
			v := int64(le.Uint16(fat[c*3/2:]))
			if c&1 == 1 {
				return v >> 4
			}
			return v & 0xFFF
		}
		return int64(le.Uint16(fat[c*2:]))
	}
	chain := func(c int64) []byte {
		var b bytes.Buffer
		for n := 0; c >= 2 && c < clusters && n < int(clusters); n++ {
			buf := make([]byte, clusterBytes)
			r.ReadAt(buf, dataStart+(c-2)*clusterBytes)
			b.Write(buf)
			c = next(c)
		}
		return b.Bytes()
	}

	var out []file
	var walk func(dir []byte, prefix string, depth int)
	walk = func(dir []byte, prefix string, depth int) {
		for at := 0; at+32 <= len(dir); at += 32 {
			e := dir[at : at+32]
			if e[0] == 0 {
				return
			}
			attr := e[11]
			if e[0] == 0xE5 || e[0] == '.' || attr&0x08 != 0 { // deleted, . and .., volume label/LFN
				continue
			}
			name := strings.TrimRight(string(e[:8]), " ")
			if ext := strings.TrimRight(string(e[8:11]), " "); ext != "" {
				name += "." + ext
			}
			first := int64(le.Uint16(e[26:]))
			if attr&0x10 != 0 {
				if depth < 16 {
					walk(chain(first), prefix+name+"/", depth+1)
				}
				continue
			}
			fileSize := int64(le.Uint32(e[28:]))
			out = append(out, file{prefix + name, fileSize, func() (io.ReaderAt, error) {
				b := chain(first)
				if int64(len(b)) > fileSize {
					b = b[:fileSize]
				}
				return bytes.NewReader(b), nil
			}})
		}
	}
	root := make([]byte, rootBytes)
	if _, err := r.ReadAt(root, rootStart); err != nil {
		return nil, nil
	}
	walk(root, "", 0)
	return out, nil
}
