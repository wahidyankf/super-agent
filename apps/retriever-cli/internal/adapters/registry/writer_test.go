package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// writingGateway is a gateway that can also write. The embedded interface supplies the reads, which these tests never call.
type writingGateway struct {
	ports.SourceGateway
	writes int
}

// Write counts the write and answers an empty record.
func (g *writingGateway) Write(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	g.writes++
	return domain.Record{}, nil
}

// readOnlyGateway is a gateway with no Write method, so it is not a ports.SourceWriter.
type readOnlyGateway struct {
	ports.SourceGateway
}

// writeTestSource builds an outline source from valid literals. It makes no network call.
func writeTestSource(t *testing.T) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatal(err)
	}
	kind, err := domain.NewSourceKind("outline")
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://outline-stub.invalid")
	if err != nil {
		t.Fatal(err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(sid, kind, set, endpoint, credential)
}

// writeTestKind returns the kind that writeTestSource names, so a test can key a builder map by it.
func writeTestKind(t *testing.T) domain.SourceKind {
	t.Helper()
	kind, err := domain.NewSourceKind("outline")
	if err != nil {
		t.Fatal(err)
	}
	return kind
}

// TestWriterReturnsTheGatewayThatWrites checks that Writer hands back the same cached gateway For returns, so a write
// shares the request lock and the pacing rule of the reads.
func TestWriterReturnsTheGatewayThatWrites(t *testing.T) {
	gw := &writingGateway{}
	r := New(map[domain.SourceKind]Builder{
		writeTestKind(t): func(domain.Source) (ports.SourceGateway, error) { return gw, nil },
	})
	source := writeTestSource(t)

	writer, err := r.Writer(source)
	if err != nil {
		t.Fatalf("Writer = error %v, want a writer", err)
	}
	if _, err := writer.Write(context.Background(), domain.OpDocumentsRestore, domain.Args{}); err != nil {
		t.Fatalf("Write = error %v, want none", err)
	}
	if gw.writes != 1 {
		t.Fatalf("the gateway wrote %d times, want 1", gw.writes)
	}
	again, err := r.For(source)
	if err != nil {
		t.Fatalf("For = error %v, want the cached gateway", err)
	}
	if any(again) != any(writer) {
		t.Fatal("Writer does not return the gateway that For caches")
	}
}

// TestWriterRefusesAGatewayThatCannotWrite checks that a gateway with no write method is unsupported, and not nil.
func TestWriterRefusesAGatewayThatCannotWrite(t *testing.T) {
	r := New(map[domain.SourceKind]Builder{
		writeTestKind(t): func(domain.Source) (ports.SourceGateway, error) { return &readOnlyGateway{}, nil },
	})
	writer, err := r.Writer(writeTestSource(t))
	if !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Writer = (%v, %v), want ErrUnsupportedSource", writer, err)
	}
	if writer != nil {
		t.Fatal("Writer returned a writer for a gateway that cannot write")
	}
}

// TestWriterReportsABuildFailure checks that a gateway that cannot be built gives its error, so the key error reaches
// the caller and nothing is written.
func TestWriterReportsABuildFailure(t *testing.T) {
	want := domain.ErrUnsupportedSource
	r := New(map[domain.SourceKind]Builder{
		writeTestKind(t): func(domain.Source) (ports.SourceGateway, error) { return nil, want },
	})
	if _, err := r.Writer(writeTestSource(t)); !errors.Is(err, want) {
		t.Fatalf("Writer = error %v, want %v", err, want)
	}
}

// TestWriterIsUnsupportedForAKindWithoutABuilder checks that a source of a kind with no builder cannot be written.
func TestWriterIsUnsupportedForAKindWithoutABuilder(t *testing.T) {
	r := New(map[domain.SourceKind]Builder{})
	if _, err := r.Writer(writeTestSource(t)); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Writer = error %v, want ErrUnsupportedSource", err)
	}
}
