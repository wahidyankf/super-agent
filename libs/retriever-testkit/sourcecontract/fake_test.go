package sourcecontract_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/sourcecontract"
)

// TestFakeGatewayPassesTheSourceContract runs the shared contract suite against the
// in-memory fake, the unit-level implementation of ports.SourceGateway.
func TestFakeGatewayPassesTheSourceContract(t *testing.T) {
	sourcecontract.Run(t, func(_ *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
		return sourcecontract.NewFake(w)
	})
}

// standardFake returns the fake over the standard dataset with no fault.
func standardFake() sourcecontract.Gateway {
	return sourcecontract.NewFake(sourcecontract.Wiring{Dataset: sourcecontract.StandardDataset()})
}

// TestFakeReadRefusesAnOperationItDoesNotServe checks that the fake answers an operation outside its dataset with
// ErrUnsupportedOperation. The dataset holds no account, so auth.whoami is not served.
func TestFakeReadRefusesAnOperationItDoesNotServe(t *testing.T) {
	_, err := standardFake().Read(context.Background(), domain.OpAuthWhoami, domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Read(auth.whoami) error = %v, want ErrUnsupportedOperation", err)
	}
}

// TestFakeReadReportsAnUnknownCollectionForItsDocuments checks that collections.documents fails as not found for a
// collection the dataset does not hold.
func TestFakeReadReportsAnUnknownCollectionForItsDocuments(t *testing.T) {
	_, err := standardFake().Read(context.Background(), domain.OpCollectionsDocuments,
		domain.NewArgs(map[string]any{"collectionId": "col-missing"}))
	if !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("Read(collections.documents) error = %v, want ErrCollectionNotFound", err)
	}
}

// TestFakeReadFiltersDocumentsByCollection checks that documents.list with a collection filter returns only the
// documents of that collection, and that the filter leaves the other collections out.
func TestFakeReadFiltersDocumentsByCollection(t *testing.T) {
	cases := []struct {
		name       string
		collection string
		wantIDs    []string
	}{
		{name: "a collection with one document", collection: "col-03", wantIDs: []string{"doc-6"}},
		{name: "a collection with no document", collection: "col-empty", wantIDs: nil},
		{name: "no filter", collection: "", wantIDs: []string{"doc-1", "doc-2", "doc-3", "doc-4", "doc-5", "doc-6"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := standardFake().Read(context.Background(), domain.OpDocumentsList,
				domain.NewArgs(map[string]any{"collectionId": tc.collection, "offset": 0, "limit": 25}))
			if err != nil {
				t.Fatalf("Read(documents.list): %v", err)
			}
			items := rec.Items()
			if len(items) != len(tc.wantIDs) {
				t.Fatalf("documents.list returned %d documents, want %d", len(items), len(tc.wantIDs))
			}
			for i, want := range tc.wantIDs {
				if got := items[i].Str("id"); got != want {
					t.Fatalf("document %d = %q, want %q", i, got, want)
				}
			}
		})
	}
}
