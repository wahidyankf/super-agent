package kiwi

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestParseProductIDAcceptsOnlyCanonicalNumbers checks the collection ID form: product-<n>, n positive, at most nine
// digits, no leading zero.
func TestParseProductIDAcceptsOnlyCanonicalNumbers(t *testing.T) {
	valid := map[string]int{"product-1": 1, "product-123456789": 123456789}
	for raw, want := range valid {
		got, err := parseProductID(raw)
		if err != nil || got != want {
			t.Errorf("parseProductID(%q) = (%d, %v), want (%d, nil)", raw, got, err, want)
		}
	}
	for _, raw := range []string{"product-0", "product-01", "product-1234567890", "case-1", "product.1",
		"product-", "PRODUCT-1", "product--1", "product-1 ", "product-+1", ""} {
		if _, err := parseProductID(raw); !errors.Is(err, domain.ErrInvalidCollectionID) {
			t.Errorf("parseProductID(%q) error = %v, want ErrInvalidCollectionID", raw, err)
		}
	}
}

// TestParseDocumentIDNamesThePlanOrTheCase checks that plan-<n> and case-<n> parse, and that any other form is refused.
func TestParseDocumentIDNamesThePlanOrTheCase(t *testing.T) {
	if kind, n, err := parseDocumentID("plan-12"); err != nil || kind != docPlan || n != 12 {
		t.Fatalf("parseDocumentID(plan-12) = (%v, %d, %v), want (plan, 12, nil)", kind, n, err)
	}
	if kind, n, err := parseDocumentID("case-40"); err != nil || kind != docCase || n != 40 {
		t.Fatalf("parseDocumentID(case-40) = (%v, %d, %v), want (case, 40, nil)", kind, n, err)
	}
	for _, raw := range []string{"product-1", "plan-0", "case-12x", "plan-", "doc-1", "Plan-1", "case-1234567890"} {
		if _, _, err := parseDocumentID(raw); !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Errorf("parseDocumentID(%q) error = %v, want ErrInvalidDocumentID", raw, err)
		}
	}
}

// TestParsePlanAndCaseRefuseTheWrongPrefix checks that a position names one kind: a case ID in a plan position is an
// invalid document ID.
func TestParsePlanAndCaseRefuseTheWrongPrefix(t *testing.T) {
	if _, err := parsePlanID("case-1"); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("parsePlanID(case-1) error = %v, want ErrInvalidDocumentID", err)
	}
	if _, err := parseCaseID("plan-1"); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("parseCaseID(plan-1) error = %v, want ErrInvalidDocumentID", err)
	}
}

// TestIDsFormTheirCanonicalText checks the formatters, which are the inverse of the parsers.
func TestIDsFormTheirCanonicalText(t *testing.T) {
	if got := productID(7); got != "product-7" {
		t.Fatalf("productID(7) = %q", got)
	}
	if got := planID(12); got != "plan-12" {
		t.Fatalf("planID(12) = %q", got)
	}
	if got := caseID(40); got != "case-40" {
		t.Fatalf("caseID(40) = %q", got)
	}
}
