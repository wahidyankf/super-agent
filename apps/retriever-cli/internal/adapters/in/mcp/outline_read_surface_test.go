package mcp

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// outlineReadToolsGolden is the MCP surface of the Outline read tools, captured from the Go-backed build before any Outline
// read moves to the connector engine. Each tool keeps its name, description, input schema, and hints. A port must
// reproduce this text byte for byte, and a deliberate change to a tool updates this file in the same change.
//
//go:embed testdata/outline-read-tools.golden
var outlineReadToolsGolden []byte

// outlineReadToolNames returns the MCP names of the built-in Outline read rows that expose a tool, in catalog order.
func outlineReadToolNames() []string {
	var names []string
	for _, row := range operations.RowsOf(domain.KindOutline) {
		if row.Effect == domain.EffectRead && row.Tool != "" {
			names = append(names, row.Tool)
		}
	}
	return names
}

// outlineReadSurface lists the tools of the production server over an in-memory transport and returns the Outline read
// tools as indented JSON, with their names, descriptions, input schemas, and hints. Every Outline read row that exposes a
// tool must be listed, or it fails.
func outlineReadSurface(t *testing.T) []byte {
	t.Helper()
	var logs bytes.Buffer
	cs := connect(t, emptyService(t), &logs)
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := outlineReadToolNames()
	var reads []*sdk.Tool
	for _, tool := range res.Tools {
		if slices.Contains(want, tool.Name) {
			reads = append(reads, tool)
		}
	}
	if len(reads) != len(want) {
		t.Fatalf("the server lists %d Outline read tools, want %d", len(reads), len(want))
	}
	out, err := json.MarshalIndent(reads, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n')
}

// TestOutlineReadToolsMatchTheGoBackedGolden compares the Outline read tools to the golden, byte for byte.
func TestOutlineReadToolsMatchTheGoBackedGolden(t *testing.T) {
	if got := outlineReadSurface(t); !bytes.Equal(got, outlineReadToolsGolden) {
		t.Fatal("the Outline read tools differ from the golden; a change to a read tool must update the golden on purpose")
	}
}
