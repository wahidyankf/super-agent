package kiwistub

import (
	"crypto/sha256"
	"encoding/binary"
)

// detReader is a deterministic io.Reader for test key material. Its bytes are the SHA-256 of a counter, so a test that
// makes a certificate gets the same key bytes on each run and draws no real entropy.
type detReader struct{ n uint64 }

// Read fills p with the next bytes of the counter stream.
func (r *detReader) Read(p []byte) (int, error) {
	for i := 0; i < len(p); {
		var block [8]byte
		binary.BigEndian.PutUint64(block[:], r.n)
		r.n++
		sum := sha256.Sum256(block[:])
		i += copy(p[i:], sum[:])
	}
	return len(p), nil
}
