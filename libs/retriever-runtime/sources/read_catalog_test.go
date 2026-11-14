package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestReadRefusesAnOperationOutsideTheCatalog checks that an operation the catalog does not hold is refused before any
// source is asked or any argument is bound, so no request is made for it.
func TestReadRefusesAnOperationOutsideTheCatalog(t *testing.T) {
	reader := &fakeReader{reply: docObject()}
	svc, res := readService(t, reader)

	_, err := Read(context.Background(), svc, domain.OperationID("apiKeys.create"), nil, nil)

	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("error = %v, want ErrUnsupportedOperation", err)
	}
	if len(res.asked) != 0 || len(reader.ops) != 0 {
		t.Fatalf("a refused operation asked a source (%v) or reached the reader (%v)", res.asked, reader.ops)
	}
}
