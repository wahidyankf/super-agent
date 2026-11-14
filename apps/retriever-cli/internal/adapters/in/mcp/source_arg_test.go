package mcp

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// TestASourceOfAWrongTypeIsRefusedBeforeAnyRequest checks that a source argument that is not text is refused. It was
// once read as if the caller had left it out, so the call ran against the default source.
func TestASourceOfAWrongTypeIsRefusedBeforeAnyRequest(t *testing.T) {
	row, ok := operations.Lookup(domain.OpCollectionsList)
	if !ok {
		t.Fatal("collections.list is not in the catalog")
	}
	for _, raw := range []string{`{"source":5}`, `{"source":true}`, `{"source":["outline"]}`, `{"source":null}`} {
		gw := &fakeGateway{}
		svc := serviceWith(t, gw, mustSource(t, "outline"))
		req := &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Arguments: json.RawMessage(raw)}}
		_, _, _, public := runOperation(context.Background(), svc, row, req)
		if public == nil {
			t.Fatalf("arguments %s: the call succeeded, want a refusal", raw)
		}
		if gw.calls != 0 {
			t.Fatalf("arguments %s: the gateway was called %d times, want none", raw, gw.calls)
		}
	}
}
