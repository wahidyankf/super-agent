package kiwi

import (
	"fmt"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The Kiwi IDs: product-<n> is a collection, plan-<n> a test plan, and case-<n> a test case. The
// number is a positive integer of at most nine digits, with no leading zero, so the text is one canonical form.
const (
	productPrefix = "product-"
	planPrefix    = "plan-"
	casePrefix    = "case-"
	maxIDDigits   = 9
)

// docKind is the kind of a Kiwi document: a test plan or a test case.
type docKind int

const (
	docPlan docKind = iota + 1
	docCase
)

// parseNumber returns the number after prefix. The text must be the prefix and one canonical positive number.
func parseNumber(raw, prefix string) (int, bool) {
	if !strings.HasPrefix(raw, prefix) {
		return 0, false
	}
	digits := raw[len(prefix):]
	if digits == "" || len(digits) > maxIDDigits || digits[0] == '0' {
		return 0, false
	}
	// The digits are one to nine ASCII digits with no leading zero, so the value is at least one and fits an int: no
	// overflow or zero check is needed after the loop.
	n := 0
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// parseProductID returns the number of a product-<n> collection ID, or ErrInvalidCollectionID.
func parseProductID(raw string) (int, error) {
	n, ok := parseNumber(raw, productPrefix)
	if !ok {
		return 0, domain.ErrInvalidCollectionID
	}
	return n, nil
}

// parsePlanID returns the number of a plan-<n> document ID, or ErrInvalidDocumentID.
func parsePlanID(raw string) (int, error) {
	n, ok := parseNumber(raw, planPrefix)
	if !ok {
		return 0, domain.ErrInvalidDocumentID
	}
	return n, nil
}

// parseCaseID returns the number of a case-<n> document ID, or ErrInvalidDocumentID.
func parseCaseID(raw string) (int, error) {
	n, ok := parseNumber(raw, casePrefix)
	if !ok {
		return 0, domain.ErrInvalidDocumentID
	}
	return n, nil
}

// parseDocumentID returns the kind and the number of a plan or a case ID. Any other text is ErrInvalidDocumentID.
func parseDocumentID(raw string) (docKind, int, error) {
	if n, ok := parseNumber(raw, planPrefix); ok {
		return docPlan, n, nil
	}
	if n, ok := parseNumber(raw, casePrefix); ok {
		return docCase, n, nil
	}
	return 0, 0, domain.ErrInvalidDocumentID
}

// productID, planID, and caseID form the canonical text of an ID from its number.
func productID(n int) string { return fmt.Sprintf("%s%d", productPrefix, n) }
func planID(n int) string    { return fmt.Sprintf("%s%d", planPrefix, n) }
func caseID(n int) string    { return fmt.Sprintf("%s%d", casePrefix, n) }
