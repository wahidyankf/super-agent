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
// A write names its source, because a change always says where it goes, and it accepts --dry-run, which prints the plan
// and makes no request. A write has no --yes: none of stage b's writes is destructive.

// dryRunFlag is the --dry-run switch every write command accepts. A dry run runs the checks, and prints what would change.
var dryRunFlag = flag{name: "--dry-run", help: "print the change that would be made and make no request"}

// yesFlag is the --yes switch of a destructive or admin command. Without it such a command refuses
// before any request. There is no interactive prompt.
var yesFlag = flag{name: "--yes", help: "run a change that deletes, removes, or grants access; without it the command refuses"}

// confirmFlagName is the --confirm option of a critical command. Its value is the canonical target,
// which the dry run prints. It has no other source: no prompt, no standard input, no environment variable.
const confirmFlagName = "--confirm"

// writeSourceFlag is the --source option of a write command. A change always names its source, so the flag is required.
var writeSourceFlag = flag{name: "--source", meta: "<id>", help: "configured source ID; required, a change names its source"}

// runWrite runs one write row against the named source. With --dry-run it prints the plan and makes no request. Otherwise
// it prints the reply of the wiki: the projected record in JSON mode, or one line naming the change and the record in text
// mode.
func runWrite(inv *invocation, row operations.Row) error {
	rt := inv.rt
	kind, _ := row.Kind()
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	jsonMode := rt.format == domain.FormatJSON
	dryRun := inv.switches[dryRunFlag.name]
	rec, err := sources.Write(context.Background(), service, row.ID, sourceArg(inv), rawArgs(inv, row), dryRun)
	if err != nil {
		return withKind(kind, err)
	}
	switch {
	case jsonMode:
		return writeJSON(rt.stdout, jsonDocument(row, rec))
	case dryRun:
		writeDryRun(rt.stdout, rec)
	default:
		writeChanged(rt.stdout, row, rec)
	}
	return nil
}

// writeDryRun prints the plan of a dry run: the operation, the source, the target when the change names one, and each
// argument, then the line that says nothing changed. A text argument shows its byte count, which the use case put in the
// plan. The target is the value --confirm must equal for a critical call, so a person can read it.
func writeDryRun(w io.Writer, plan domain.Record) {
	fmt.Fprintf(w, "dry run of %s on source %s\n", plan.Str("operation"), plan.Str("source"))
	labels := []struct{ field, label string }{
		{"target", "target"}, {"targetName", "target name"}, {"targetCount", "items in the target"},
	}
	for _, l := range labels {
		if v, ok := plan.Get(l.field); ok {
			fmt.Fprintf(w, "%s: %s\n", l.label, scalarText(v))
		}
	}
	writeFields(w, "", plan.Rec("arguments"))
	fmt.Fprintln(w, "nothing was changed")
}

// runAdmin runs one destructive or admin row against the named source. The --yes
// switch and the --confirm value are passed to the use case, which refuses a call that lacks them before any request. With
// --dry-run it prints the plan, the target included, and makes no change.
func runAdmin(inv *invocation, row operations.Row) error {
	rt := inv.rt
	kind, _ := row.Kind()
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	jsonMode := rt.format == domain.FormatJSON
	conf := sources.Confirmation{DryRun: inv.switches[dryRunFlag.name], Yes: inv.switches[yesFlag.name]}
	if value, ok := inv.values[confirmFlagName]; ok {
		conf.Confirm = &value
	}
	rec, err := sources.Administer(context.Background(), service, row.ID, sourceArg(inv), rawArgs(inv, row), conf)
	if err != nil {
		return withKind(kind, err)
	}
	switch {
	case jsonMode:
		return writeJSON(rt.stdout, jsonDocument(row, rec))
	case conf.DryRun:
		writeDryRun(rt.stdout, rec)
	default:
		writeChanged(rt.stdout, row, rec)
	}
	return nil
}

// writeChanged prints the record the wiki answered for a write, after one line that names the change.
func writeChanged(w io.Writer, row operations.Row, rec domain.Record) {
	fmt.Fprintf(w, "changed: %s\n", row.CLI)
	writeText(w, row, rec)
}
