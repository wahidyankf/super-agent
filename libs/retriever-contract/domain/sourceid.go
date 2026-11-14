package domain

import "regexp"

// sourceIDPattern is the slug shape of a source ID. It has no colon, space, or newline, so an ID is safe in a
// front-matter line and in a folder name.
var sourceIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// maxSourceIDLength is the longest source ID in bytes. Every accepted byte is ASCII, so bytes are characters.
const maxSourceIDLength = 32

// SourceID is the stable, unique name of one configured source, such as "outline". Build it with NewSourceID.
type SourceID struct {
	value string
}

// NewSourceID validates raw. Uppercase letters are rejected, not lowered.
func NewSourceID(raw string) (SourceID, error) {
	if len(raw) < 1 || len(raw) > maxSourceIDLength || !sourceIDPattern.MatchString(raw) {
		return SourceID{}, ErrInvalidSourceID
	}
	return SourceID{value: raw}, nil
}

// String returns the validated source ID.
func (id SourceID) String() string {
	return id.value
}
