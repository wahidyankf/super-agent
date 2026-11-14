package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// binder adds one hand-written tool to a server. It is built from a use case by bind.
type binder func(server *sdk.Server, meta *sdk.Tool, c caller)

// tool is one hand-written row: its name, what it does, whether it calls a
// wiki, and how it is bound to its use case. Only a tool that is not an operation of the catalog is written here; every
// operation of the catalog becomes a tool in operation_tools.go.
type tool struct {
	name        string
	description string
	effect      domain.Effect
	opensWorld  bool
	bind        binder
}

// handWrittenTools is the table of the tools the catalog does not generate. list_sources reads the catalog itself, not
// a wiki, so it is not an operation.
var handWrittenTools = []tool{
	{
		name:        "list_sources",
		description: "Lists the configured wiki sources with their kind and tags. Filter by kind and by tags; a source must have every tag given. The endpoint and the credential name are never shown.",
		effect:      domain.EffectRead,
		opensWorld:  false,
		bind:        bind(listSources),
	},
}

// caller carries what every handler needs: the source service and the logger.
type caller struct {
	svc sources.Service
	log *slog.Logger
}

// register adds every hand-written tool of the table. A table that names one tool twice is an error. It returns the names
// it added.
func register(server *sdk.Server, svc sources.Service, log *slog.Logger, table []tool) ([]string, error) {
	c := caller{svc: svc, log: log}
	seen := map[string]bool{}
	var names []string
	for _, t := range table {
		if seen[t.name] {
			return nil, fmt.Errorf("tool %s is listed twice", t.name)
		}
		seen[t.name] = true
		meta := &sdk.Tool{
			Name:        t.name,
			Description: describe(t.description),
			Annotations: annotationsOf(t.effect, t.opensWorld),
		}
		t.bind(server, meta, c)
		names = append(names, t.name)
	}
	return names, nil
}

// RegisteredTools returns the names of every tool a server built over svc offers for the named resources (nil for every
// resource), the hand-written ones first. A sync test compares it with the catalog and the command tree.
func RegisteredTools(svc sources.Service, resources []string) ([]string, error) {
	return registeredTools(svc, resources, handWrittenTools)
}

// registeredTools is RegisteredTools over the given hand-written table, so a test can offer a table that the production
// one never is, such as one that names a tool twice.
func registeredTools(svc sources.Service, resources []string, table []tool) ([]string, error) {
	server := sdk.NewServer(&sdk.Implementation{Name: "retriever"}, nil)
	names, err := register(server, svc, slog.New(slog.DiscardHandler), table)
	if err != nil {
		return nil, err
	}
	return append(names, registerOperations(server, svc, slog.New(slog.DiscardHandler), newResourceSet(resources))...), nil
}

// describe ends a tool description with the untrusted-data sentence.
func describe(text string) string {
	return text + " " + untrustedSentence
}

// annotationsOf derives the hints of a read tool from its effect, and of a hand-written tool. The hints of a generated
// write row come from rowAnnotations, which reads the row's own flags. Every pointer is set, because the SDK defaults a
// nil DestructiveHint to true, and a read tool must not be announced as a writer. The hints are advice from the server:
// the control is the use case, not the annotation.
func annotationsOf(e domain.Effect, opensWorld bool) *sdk.ToolAnnotations {
	world := opensWorld
	switch e {
	case domain.EffectRead:
		return &sdk.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: ptr(false),
			OpenWorldHint:   &world,
		}
	default:
		// Only the read effect is annotated here. Any other effect gets the most cautious hints, so an effect that is
		// not yet annotated is never announced as read-only.
		return &sdk.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: &world}
	}
}

// ptr returns a pointer to a copy of v, for the optional hints.
func ptr[T any](v T) *T {
	return &v
}

// bind adds one hand-written tool built from a use case. The use case returns the answer, the resolved source ID for the
// log line ("" when none was resolved), and an error.
func bind[In, Out any](use func(context.Context, sources.Service, In) (Out, string, error)) binder {
	return func(server *sdk.Server, meta *sdk.Tool, c caller) {
		sdk.AddTool(server, meta, handler(c, meta.Name, use))
	}
}

// handler turns a use case into an SDK handler. It times the call, writes one log line, and returns a failure as an
// error whose text is the public text only. The SDK sends that text to the model as a tool error. A panic in the use
// case becomes the unexpected error, so no panic value reaches the model, the log, or the process exit.
func handler[In, Out any](c caller, name string, use func(context.Context, sources.Service, In) (Out, string, error)) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		start := time.Now()
		out, source, panicType, err := guarded(ctx, c.svc, in, use)
		if panicType != "" {
			var zero Out
			c.record(name, source, outcomeUnexpected, start, "panic_type", panicType)
			return nil, zero, errUnexpected
		}
		if err != nil {
			var zero Out
			public, class := publicError(err)
			c.record(name, source, class, start)
			return nil, zero, public
		}
		c.record(name, source, outcomeOK, start)
		return nil, out, nil
	}
}

// guarded runs one use case and recovers its panic. A recovered panic is reported by the type of its value only,
// because the value can carry anything, a key included. Its text is never kept.
func guarded[In, Out any](ctx context.Context, svc sources.Service, in In, use func(context.Context, sources.Service, In) (Out, string, error)) (out Out, source, panicType string, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero Out
			out, source, panicType, err = zero, "", fmt.Sprintf("%T", r), nil
		}
	}()
	out, source, err = use(ctx, svc, in)
	return out, source, "", err
}

// outcomeOK and outcomeUnexpected are the outcome classes of a successful call and of an error the adapter does not
// know, in the log line.
const (
	outcomeOK         = "ok"
	outcomeUnexpected = "unexpected"
)

// record writes the one log line of a tool call: the tool, the resolved source when there is one, the outcome class,
// the duration in milliseconds, and any extra pairs. It never holds an argument, a result, or a message from an
// upstream service.
func (c caller) record(name, source, outcome string, start time.Time, extra ...any) {
	attrs := []any{"tool", name}
	if source != "" {
		attrs = append(attrs, "source", source)
	}
	attrs = append(attrs, "outcome", outcome, "duration_ms", time.Since(start).Milliseconds())
	attrs = append(attrs, extra...)
	c.log.Info("tool call", attrs...)
}

// --- list_sources ---------------------------------------------------------------

type listSourcesIn struct {
	Kind *string  `json:"kind,omitempty" jsonschema:"the source kind to keep, such as outline"`
	Tags []string `json:"tags,omitempty" jsonschema:"tags a source must all have to be listed"`
}

type listSourcesOut struct {
	Sources   []sourceOut `json:"sources"`
	Count     int         `json:"count"`
	Truncated bool        `json:"truncated"`
}

type sourceOut struct {
	ID   string   `json:"id"`
	Kind string   `json:"kind"`
	Tags []string `json:"tags"`
}

// tagsOrEmpty returns the tags of a source, or an empty list when it has none, so the answer shows [] and never null.
func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

// listSources returns the catalog rows that match the filter. It reads the catalog only.
func listSources(_ context.Context, svc sources.Service, in listSourcesIn) (listSourcesOut, string, error) {
	filter, err := domain.NewSourceFilter(in.Kind, in.Tags...)
	if err != nil {
		return listSourcesOut{}, "", err
	}
	summaries := sources.ListSources(svc.Catalog, filter)
	rows := make([]sourceOut, 0, len(summaries))
	for _, r := range summaries {
		rows = append(rows, sourceOut{ID: r.ID, Kind: r.Kind, Tags: tagsOrEmpty(r.Tags)})
	}
	kept, truncated := fitPlainRows(rows, func(kept []sourceOut, tr bool) any {
		return listSourcesOut{Sources: kept, Count: len(kept), Truncated: tr}
	})
	return listSourcesOut{Sources: kept, Count: len(kept), Truncated: truncated}, "", nil
}

// errors the adapter adds to the public set. Their texts are fixed and carry no value.
var (
	errNegativeOffset = errors.New("offset must not be negative")
	errBadOffset      = errors.New("offset must be a whole number of bytes")
	errOffsetPastEnd  = errors.New("offset is past the end of the document")
	errUnexpected     = errors.New("unexpected error")
	errBadSource      = errors.New("the source must be text")
)

// missingKeyText replaces the variable name of a missing key. The name of a credential is never shown to a model.
const missingKeyText = "the API key for this source is not set"

// invalidIdentityText replaces the variable name of a malformed e-mail address, for the same reason as missingKeyText.
const invalidIdentityText = "the e-mail address for this source is not valid"

// publicError maps a use-case error onto the text the model may see and one fixed outcome class for the log line.
// Only fixed texts and the names of the source IDs the catalog validated reach the model. A value that comes from
// the upstream service never does.
func publicError(err error) (error, string) {
	var missing domain.MissingKeyError
	if errors.As(err, &missing) {
		return errors.New(missingKeyText), "missing key"
	}
	var invalidIdentity domain.InvalidIdentityError
	if errors.As(err, &invalidIdentity) {
		// The variable name is never shown to a model, as for a missing key.
		return errors.New(invalidIdentityText), "invalid identity"
	}
	var unknown domain.UnknownSourceError
	if errors.As(err, &unknown) {
		return fmt.Errorf("unknown source %q", unknown.ID), "unknown source"
	}
	var ambiguous domain.AmbiguousSourceError
	if errors.As(err, &ambiguous) {
		text := fmt.Sprintf("more than one %s source exists (%s); pass the source argument",
			ambiguous.Kind, strings.Join(ambiguous.IDs, ", "))
		return errors.New(text), "ambiguous source"
	}
	var unavailable domain.OperationUnavailableError
	if errors.As(err, &unavailable) {
		return errors.New(unavailable.Error()), "operation unavailable"
	}
	var noSource domain.NoSourceOfKindError
	if errors.As(err, &noSource) {
		return errors.New(noSource.Error()), "no source of kind"
	}
	if errors.Is(err, domain.ErrNoSourceOfKind) {
		// A raw sentinel names no kind, so it gets the default kind's text.
		return errors.New("no outline source is configured"), "no source of kind"
	}
	for _, typed := range []error{errNegativeOffset, errBadOffset, errOffsetPastEnd} {
		if errors.Is(err, typed) {
			return typed, typed.Error()
		}
	}
	var kindErr domain.KindError
	if errors.As(err, &kindErr) {
		if text, ok := kindErr.PublicText(); ok {
			return errors.New(text), text
		}
	}
	for _, typed := range domain.PublicErrors() {
		if errors.Is(err, typed) {
			return typed, typed.Error()
		}
	}
	return errUnexpected, outcomeUnexpected
}
