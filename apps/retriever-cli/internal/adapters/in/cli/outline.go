package cli

import (
	"context"
	"fmt"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// outlineWiring returns the source service the source commands need. A setup error from the composition root is returned
// as it is, and a service the root did not supply refuses the command.
func (rt *runtime) outlineWiring() (sources.Service, error) {
	if rt.setupErr != nil {
		return sources.Service{}, setupError{err: rt.setupErr}
	}
	if rt.service.Gateways == nil {
		return sources.Service{}, errNotConfigured
	}
	return rt.service, nil
}

// sourceArg returns the --source value, or nil when the flag is omitted, which applies the default rule.
func sourceArg(inv *invocation) *string {
	if raw, ok := inv.values["--source"]; ok {
		return &raw
	}
	return nil
}

// exportDocument writes one document of the kind's source as a Markdown file in the --out folder. The sink the composition
// root gives is the only place that writes, so the command has no other side effect. Text mode prints the location the
// sink reports. JSON mode prints one document with the schema version and that location.
func exportDocument(inv *invocation, kind domain.SourceKind) error {
	rt := inv.rt
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	if rt.sink == nil {
		return withKind(kind, errNotConfigured)
	}
	location, err := sources.ExportDocument(context.Background(), service, rt.sink(inv.values["--out"]), kind,
		sourceArg(inv), inv.positional[0])
	if err != nil {
		return withKind(kind, err)
	}
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.out, map[string]any{"schemaVersion": schemaVersion, "location": location})
	}
	_, err = fmt.Fprintf(rt.out, "Wrote %s\n", location)
	return err
}
