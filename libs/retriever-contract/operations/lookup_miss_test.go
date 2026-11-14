package operations_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestLookupReportsAnUnknownOperation checks the miss branch: an ID that no row names returns no row and false.
func TestLookupReportsAnUnknownOperation(t *testing.T) {
	row, found := operations.Lookup(domain.OperationID("no.such.operation"))
	if found {
		t.Fatalf("Lookup of an unknown ID found row %+v", row)
	}
	if row.ID != "" {
		t.Errorf("Lookup of an unknown ID returned a non-zero row %+v", row)
	}
}
