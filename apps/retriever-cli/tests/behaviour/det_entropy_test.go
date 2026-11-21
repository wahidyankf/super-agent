package behaviour

import (
	"crypto/sha256"
	"encoding/binary"
)

// detEntropy is a deterministic io.Reader for the key material of the stub certificates. Its bytes are the SHA-256 of a
// counter, so a scenario that starts a stub reads no real entropy, and the same run makes the same certificate.
type detEntropy struct{ n uint64 }

// Read fills p with the next bytes of the counter stream.
func (r *detEntropy) Read(p []byte) (int, error) {
	for i := 0; i < len(p); {
		var block [8]byte
		binary.BigEndian.PutUint64(block[:], r.n)
		r.n++
		sum := sha256.Sum256(block[:])
		i += copy(p[i:], sum[:])
	}
	return len(p), nil
}
