package kiwi

import (
	"crypto/sha256"
	"encoding/binary"
)

// detReader is a deterministic io.Reader for test key material. Its bytes are the SHA-256 of a counter, so a test that
// makes a key gets the same bytes each run and reads no random source.
type detReader struct{ n uint64 }

// Read fills p with the next bytes of the counter stream.
func (r *detReader) Read(p []byte) (int, error) {
	for filled := 0; filled < len(p); {
		var counter [8]byte
		binary.BigEndian.PutUint64(counter[:], r.n)
		r.n++
		sum := sha256.Sum256(counter[:])
		filled += copy(p[filled:], sum[:])
	}
	return len(p), nil
}
