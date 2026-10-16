package id

import (
	"crypto/sha256"
	"encoding/binary"
)

const derivedCharset = "0123456789abcdefghijklmnopqrstuvwxyz"

// DerivedID is the ID a copied row takes from what it was copied from: always the same for the same source, so a copy
// that runs again inserts nothing twice. It has GenID's shape (a prefix and 12 base-36 characters).
func DerivedID(prefix IDPrefix, source string) string {
	sum := sha256.Sum256([]byte(source))
	out := make([]byte, IDLength12)
	for i := range out {
		v := binary.BigEndian.Uint16(sum[i*2 : i*2+2])
		out[i] = derivedCharset[int(v)%len(derivedCharset)]
	}
	return string(prefix) + "_" + string(out)
}
