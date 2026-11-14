package mcp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// TestAnUndeclaredInputNameIsRefusedBeforeAnyCall covers finding M3. A tool takes only the snake_case names it lists, the
// source, and the MCP-only switches. A camelCase catalog name, or a second spelling of one parameter, is refused as an
// invalid argument before any binding, so the edit-mode guard of update_document cannot be passed by naming editMode
// beside edit_mode. No request is made.
func TestAnUndeclaredInputNameIsRefusedBeforeAnyCall(t *testing.T) {
	cases := map[string]struct {
		tool string
		args map[string]any
	}{
		"a camelCase catalog name": {"update_document", map[string]any{
			"source": "outline", "document_id": "doc-1", "text": "Plan", "editMode": "append",
		}},
		"both spellings of edit mode": {"update_document", map[string]any{
			"source": "outline", "document_id": "doc-1", "text": "Plan", "edit_mode": "append", "editMode": "replace",
		}},
		"a camelCase id beside the snake_case revision": {"update_document", map[string]any{
			"source": "outline", "documentId": "doc-1", "text": "Plan", "last_revision": 5,
		}},
		"a read tool given a switch it does not list": {"get_document", map[string]any{
			"source": "outline", "document_id": "doc-1", "dry_run": true,
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			gw := &fakeGateway{}
			svc := serviceWith(t, gw, mustSource(t, "outline"))
			var logs bytes.Buffer
			cs := connect(t, svc, &logs)
			text, isErr := call(t, cs, tc.tool, tc.args)
			if !isErr || !strings.Contains(text, "use the input names the tool lists") {
				t.Fatalf("answer = (%v, %q), want an invalid-arguments error", isErr, text)
			}
			if gw.calls != 0 {
				t.Fatalf("the gateway was called %d times, want none", gw.calls)
			}
		})
	}
}

// updateReply returns the Record the adapter answers an update with: every field the update row names, so the use case
// accepts it.
func updateReply(t *testing.T) domain.Record {
	t.Helper()
	row, ok := operations.Lookup(domain.OpDocumentsUpdate)
	if !ok {
		t.Fatal("documents.update is not in the catalog")
	}
	fields := make([]domain.Field, 0, len(row.Fields))
	for _, name := range row.Fields {
		fields = append(fields, domain.Field{Name: name, Value: "x"})
	}
	return domain.NewRecord(fields...)
}
