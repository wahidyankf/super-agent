package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/sourcecontract"
)

// recorder is a sourcecontract.Reporter that keeps each mismatch, so a check can fail without failing this test.
type recorder struct {
	reports []string
}

func (r *recorder) Helper() {}

func (r *recorder) Fatalf(format string, args ...any) {
	r.reports = append(r.reports, fmt.Sprintf(format, args...))
}

// referenceGateway builds the reference fake for a Wiring.
func referenceGateway(w sourcecontract.Wiring) ports.SourceGateway {
	return sourcecontract.NewFake(w)
}

// defectiveGateway is the reference fake with one defect: its collection list drops the last collection.
type defectiveGateway struct {
	sourcecontract.Gateway
}

func (g defectiveGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	cs, err := g.Gateway.ListCollections(ctx)
	if err != nil || len(cs) == 0 {
		return cs, err
	}
	return cs[:len(cs)-1], nil
}

func defectiveReferenceGateway(w sourcecontract.Wiring) ports.SourceGateway {
	return defectiveGateway{Gateway: sourcecontract.NewFake(w)}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: The reference fake gateway passes the whole source contract suite
func TestReferenceFakePassesWholeSourceContractSuite(t *testing.T) {
	sourcecontract.Run(t, func(_ *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
		return referenceGateway(w)
	})
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: A read of an operation that the fake does not serve is refused
func TestReadOfUnservedOperationIsRefused(t *testing.T) {
	gw := sourcecontract.NewFake(sourcecontract.Wiring{Dataset: sourcecontract.StandardDataset()})
	rec, err := gw.Read(context.Background(), domain.OperationID("users.list"), domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("error = %v, want unsupported operation", err)
	}
	if rec.IsList() || len(rec.Fields()) > 0 {
		t.Fatalf("the refused read returned a reply: %+v", rec)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: The documents of an unknown collection report collection not found
func TestDocumentsOfUnknownCollectionReportCollectionNotFound(t *testing.T) {
	gw := sourcecontract.NewFake(sourcecontract.Wiring{Dataset: sourcecontract.StandardDataset()})
	args := domain.NewArgs(map[string]any{"collectionId": "col-missing"})
	_, err := gw.Read(context.Background(), domain.OpCollectionsDocuments, args)
	if !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("error = %v, want collection not found", err)
	}
	if err.Error() != "collection not found" {
		t.Fatalf("error text = %q, want %q", err.Error(), "collection not found")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: The documents read filters by collection
func TestDocumentsReadFiltersByCollection(t *testing.T) {
	gw := sourcecontract.NewFake(sourcecontract.Wiring{Dataset: sourcecontract.StandardDataset()})
	args := domain.NewArgs(map[string]any{"collectionId": "col-tree", "offset": 0, "limit": 50})
	rec, err := gw.Read(context.Background(), domain.OpDocumentsList, args)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	items := rec.Items()
	if len(items) != 5 {
		t.Fatalf("the read returned %d documents, want the 5 of col-tree", len(items))
	}
	for _, item := range items {
		if got := item.Str("collectionId"); got != "col-tree" {
			t.Fatalf("a document of collection %q was returned", got)
		}
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: A check that a gateway fails reports its defect once
func TestCheckThatGatewayFailsReportsItsDefectOnce(t *testing.T) {
	profile := sourcecontract.OutlineProfile()
	rec := &recorder{}
	for _, name := range sourcecontract.CheckNames(profile) {
		if !sourcecontract.RunCheck(rec, profile, name, defectiveReferenceGateway) {
			t.Fatalf("the contract has no check named %q", name)
		}
	}
	if len(rec.reports) != 1 {
		t.Fatalf("the defect was reported %d times: %v", len(rec.reports), rec.reports)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: Every check of the suite passes the fake gateway when run on its own
func TestEveryCheckPassesFakeGatewayWhenRunOnItsOwn(t *testing.T) {
	profile := sourcecontract.OutlineProfile()
	names := sourcecontract.CheckNames(profile)
	if len(names) == 0 {
		t.Fatal("the contract has no checks")
	}
	for _, name := range names {
		rec := &recorder{}
		if !sourcecontract.RunCheck(rec, profile, name, referenceGateway) {
			t.Fatalf("the contract has no check named %q", name)
		}
		if len(rec.reports) != 0 {
			t.Fatalf("check %q failed on the fake: %v", name, rec.reports)
		}
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/source-contract-suite.feature: A nested record is shown as plain values
func TestNestedRecordIsShownAsPlainValues(t *testing.T) {
	rec := domain.NewRecord(
		domain.Field{Name: "outer", Value: domain.NewRecord(domain.Field{Name: "inner", Value: "x"})},
		domain.Field{Name: "list", Value: domain.NewList([]domain.Record{
			domain.NewRecord(domain.Field{Name: "id", Value: "a"}),
		})},
	)
	got, err := json.Marshal(sourcecontract.PlainValue(rec))
	if err != nil {
		t.Fatalf("encode the plain value: %v", err)
	}
	want := `{"list":[{"id":"a"}],"outer":{"inner":"x"}}`
	if string(got) != want {
		t.Fatalf("plain value = %s, want %s", got, want)
	}
}
