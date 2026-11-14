package domain

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

// MaxReasonLength is the most characters a reason may hold. The server refuses a longer reason of archive and delete
// calls (inventory, Part 2: str[..2000]), so Retriever refuses it before any request.
const MaxReasonLength = 2000

// ErrInvalidReason reports a reason that is longer than MaxReasonLength characters or holds a control character.
var ErrInvalidReason = errors.New("invalid reason: use at most 2000 characters with no control character")

// ValidateReason checks a reason of a change, such as an archive or a delete. An empty reason is allowed, because the
// server does not require one.
func ValidateReason(s string) error {
	if utf8.RuneCountInString(s) > MaxReasonLength {
		return ErrInvalidReason
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return ErrInvalidReason
		}
	}
	return nil
}
