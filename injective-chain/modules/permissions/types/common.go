package types

import (
	"encoding/binary"
)

// Uint32ToLittleEndian - marshals uint32 to a little endian byte slice so it can be sorted
func Uint32ToLittleEndian(i uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, i)
	return b
}

// LittleEndianToUint32 returns an uint32 from little endian encoded bytes. If encoding
// is empty, zero is returned.
func LittleEndianToUint32(bz []byte) uint32 {
	if len(bz) == 0 {
		return 0
	}

	return binary.LittleEndian.Uint32(bz)
}
