package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// The write commands of the outline group are generated from the same catalog rows as the reads.
// A write names its source, because a change always says where it goes.

// writeSourceFlag is the --source option of a write command. A change always names its source, so the flag is required.
var writeSourceFlag = flag{name: "--source", meta: "<id>", help: "configured source ID; required, a change names its source"}

// runWrite runs one write row against the named source. It prints the reply of the wiki: the projected record in JSON
// mode, or one line naming the change and the record in text mode.
func runWrite(inv *invocation, row operations.Row) error {
	rt := inv.rt
	kind, _ := row.Kind()
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	rec, err := sources.Write(context.Background(), service, row.ID, sourceArg(inv), rawArgs(inv, row))
	if err != nil {
		return withKind(kind, err)
	}
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.stdout, jsonDocument(row, rec))
	}
	writeChanged(rt.stdout, row, rec)
	return nil
}

// writeChanged prints the record the wiki answered for a write, after one line that names the change.
func writeChanged(w io.Writer, row operations.Row, rec domain.Record) {
	fmt.Fprintf(w, "changed: %s\n", row.CLI)
	writeText(w, row, rec)
}
