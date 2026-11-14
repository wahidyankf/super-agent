package mcp

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// postReadRow returns the catalog row of a read over POST from a user spec, before any pin is known. Its routing is a read,
// and only its hint flag differs between the two tests below.
func postReadRow(writeHints bool) operations.Row {
	return operations.Row{
		ID:          "example-wiki.docs.find",
		Tool:        "example_wiki_docs_find",
		Summary:     "Find documents in a collection.",
		Description: "Find documents in a collection.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		WriteHints:  writeHints,
	}
}

// TestUnpinnedUserPostReadHintsAreNotReadOnly checks the hint rule of ADR 002, E1. A POST read from a user spec whose source
// does not pin the spec announces the hints of a write, so no client runs it as read-only. The same read, once its source
// pins the spec, announces read-only hints. The routing stays a read in both cases.
func TestUnpinnedUserPostReadHintsAreNotReadOnly(t *testing.T) {
	unpinned := postReadRow(true)
	hints := toolFor(unpinned).Annotations
	if hints.ReadOnlyHint {
		t.Fatal("an unpinned POST read is announced as read-only")
	}
	if hints.DestructiveHint == nil || *hints.DestructiveHint {
		t.Fatal("an unpinned POST read is not announced as non-destructive, as a create is")
	}
	if isWrite(unpinned) {
		t.Fatal("an unpinned POST read was routed as a write")
	}

	pinned := toolFor(postReadRow(false)).Annotations
	if !pinned.ReadOnlyHint || pinned.DestructiveHint == nil || *pinned.DestructiveHint {
		t.Fatalf("a pinned POST read lost its read-only hints: %+v", pinned)
	}
}

// TestGetReadHintsAreUnchanged checks that a GET read keeps the read-only hints whatever its write flag says, because the
// flag is set only on a POST read.
func TestGetReadHintsAreUnchanged(t *testing.T) {
	get := operations.Row{
		ID:          "example-wiki.docs.get",
		Tool:        "example_wiki_docs_get",
		Summary:     "Read one document.",
		Description: "Read one document.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
	}
	hints := toolFor(get).Annotations
	if !hints.ReadOnlyHint || !hints.IdempotentHint {
		t.Fatalf("a GET read lost its read-only hints: %+v", hints)
	}
}
