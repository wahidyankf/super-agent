package engine

import (
	"fmt"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// truncatedField is the output field that the truncated block computes on each item of a cursor read with the
// page-end rule.
const truncatedField = "truncated"

// cutOf returns the page size to keep for a read with the page-end rule: the value of the Limit or Int parameter
// that cut_to_limit names. A cut of zero or less keeps nothing meaningful, so it means no cut, and the walk stays
// bounded by the page cap.
func cutOf(o spec.Operation, args domain.Args) int {
	if o.Pagination == nil || o.Pagination.CutToLimit == "" {
		return 0
	}
	return args.Int(o.Pagination.CutToLimit)
}

// cutPage keeps the first cut items of a page that holds more, and reports whether it cut. A page within the cut
// passes whole.
func cutPage(items []domain.Record, cut int) ([]domain.Record, bool) {
	if cut > 0 && len(items) > cut {
		return items[:cut], true
	}
	return items, false
}

// lastOf returns the reply flag that decides last: the bool at the last pointer. A missing, null, or non-bool flag
// is an unexpected response naming the pointer path, because the flag beats the short-page rule and the walk cannot
// guess it.
func lastOf(root any, pointer string) (bool, error) {
	v, found := pointerValue(root, pointer)
	if !found || v == nil {
		return false, fmt.Errorf("last_pointer %s: %w", pointer, domain.ErrUnexpectedResponse)
	}
	last, isBool := v.(bool)
	if !isBool {
		return false, fmt.Errorf("last_pointer %s: %w", pointer, domain.ErrUnexpectedResponse)
	}
	return last, nil
}

// truncatedOf returns the per-item truncated flag of a cursor read with the truncated block: true exactly when the
// token is present and non-empty and last is false, or when the cut kept only the first items. A missing token
// counts as empty and a missing last counts as false, as the Go code reads them. A non-string token or a non-bool
// last is an unexpected response naming the pointer path.
func truncatedOf(root any, t spec.Truncated, cut bool) (bool, error) {
	token, err := tokenOf(root, t.Token)
	if err != nil {
		return false, err
	}
	last, err := lastOrFalse(root, t.Last)
	if err != nil {
		return false, err
	}
	return (token != "" && !last) || cut, nil
}

// tokenOf returns the next-page token at the pointer: the empty text when the reply holds no token, and an
// unexpected response naming the pointer path when the token is not a string.
func tokenOf(root any, pointer string) (string, error) {
	v, found := pointerValue(root, pointer)
	if !found || v == nil {
		return "", nil
	}
	token, isText := v.(string)
	if !isText {
		return "", fmt.Errorf("truncated token %s: %w", pointer, domain.ErrUnexpectedResponse)
	}
	return token, nil
}

// lastOrFalse returns the last flag at the pointer for the truncated block: false when the reply holds no flag, and
// an unexpected response naming the pointer path when the flag is not a bool.
func lastOrFalse(root any, pointer string) (bool, error) {
	v, found := pointerValue(root, pointer)
	if !found || v == nil {
		return false, nil
	}
	last, isBool := v.(bool)
	if !isBool {
		return false, fmt.Errorf("truncated last %s: %w", pointer, domain.ErrUnexpectedResponse)
	}
	return last, nil
}
