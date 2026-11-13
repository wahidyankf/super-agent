package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Confirmation is what the caller says about one destructive or admin call (ADR 007 sections 3.4 and 3.5).
type Confirmation struct {
	// DryRun returns the plan, with the target read from the wiki, and sends no change. It needs neither Yes nor Confirm,
	// so a person can read the target and type it.
	DryRun bool
	// Yes is the --yes of the CLI. A call that is destructive or admin needs it. The MCP surface sets it, because an MCP
	// tool has no --yes: its controls are the ceiling, the source's access, and dry_run (ADR 007 section 3.3).
	Yes bool
	// Confirm is the --confirm value of a critical call. Nil means that none was given.
	Confirm *string
}

// resolvedTarget is the target of a change as the wiki answered it. Value is the canonical value that --confirm must
// equal, and Name, Count, and HasCount are what a dry run shows beside it. FullPage is set when a list target's page was
// full, so the trash may hold more items than Count (finding m7).
type resolvedTarget struct {
	Value    string
	Name     string
	Count    int
	HasCount bool
	FullPage bool
}

// Administer runs one destructive or admin operation of the catalog against the selected source (ADR 007 sections 3.1,
// 3.4, and 3.5). The order is fixed: a read is refused; the arguments are bound; a source is required and selected; the
// level and the kind are checked. A dry run makes its self-check and reads the target, then returns the plan. Otherwise
// --yes is required, and only then comes the self-check of a change to the key's own user, then a critical call must give
// the canonical target value that Retriever read, and last the admin port is called once. A refused, invalid, or dry-run call
// makes no change to the wiki, and a call without --yes makes no read at all.
func Administer(ctx context.Context, svc Service, op domain.OperationID, source *string, raw map[string]any, conf Confirmation) (domain.Record, error) {
	row, kind, err := rowOf(op)
	if err != nil {
		return domain.Record{}, err
	}
	if row.Effect == domain.EffectRead {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	args, err := operations.Bind(row, raw)
	if err != nil {
		return domain.Record{}, err
	}
	if source == nil {
		return domain.Record{}, domain.ErrSourceRequired
	}
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return domain.Record{}, err
	}
	if err := operations.Refuse(row, src); err != nil {
		return domain.Record{}, err
	}
	if conf.DryRun {
		if err := selfCheck(ctx, svc, row, src, args); err != nil {
			return domain.Record{}, err
		}
		target, err := readTarget(ctx, svc, row, src, args)
		if err != nil {
			return domain.Record{}, err
		}
		return dryRunAdminPlan(row, src, args, target), nil
	}
	if row.NeedsYes() && !conf.Yes {
		return domain.Record{}, domain.ErrYesRequired
	}
	if err := selfCheck(ctx, svc, row, src, args); err != nil {
		return domain.Record{}, err
	}
	if row.IsCritical(args.Values()) {
		if err := confirmCritical(ctx, svc, row, src, args, conf); err != nil {
			return domain.Record{}, err
		}
	}
	admin, err := svc.Gateways.Admin(src)
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := admin.Administer(ctx, op, args)
	if err != nil {
		return domain.Record{}, err
	}
	if !matchesRow(row, reply) {
		// The wiki answered, so the change may have run, and its reply cannot be read (ADR 007 section 3.4): this is the
		// same outcome as a 2xx reply that is not a readable success, so the caller never retries it.
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return reply, nil
}

// selfCheck refuses a change to the key's own user when the row marks it (ADR 007 section 3.5, rule 6). A row that does not
// mark it makes no read.
func selfCheck(ctx context.Context, svc Service, row operations.Row, src domain.Source, args domain.Args) error {
	if !row.RefuseSelf {
		return nil
	}
	return refuseOwnUser(ctx, svc, src, args)
}

// confirmCritical checks the --confirm of a critical call against the canonical value of its target (ADR 007 section 3.5).
// It reads the target, then compares the --confirm with the value that Retriever read or named. A critical row that names no
// target, no host, and no text has no canonical value, so the call stops before any request, with ErrUnsupportedOperation,
// instead of dereferencing a missing target. The catalog test that pins every critical row keeps that case unreachable.
func confirmCritical(ctx context.Context, svc Service, row operations.Row, src domain.Source, args domain.Args, conf Confirmation) error {
	target, err := readTarget(ctx, svc, row, src, args)
	if err != nil {
		return err
	}
	if target == nil {
		return domain.ErrUnsupportedOperation
	}
	if conf.Confirm == nil {
		return domain.ConfirmError{Kind: row.Critical.Kind}
	}
	if *conf.Confirm != target.Value {
		return domain.ConfirmError{Kind: row.Critical.Kind, Mismatch: true}
	}
	return nil
}

// readTarget reads the target of a change through the read port, so a dry run can name it and a critical call can
// compare its --confirm with it. A row with no target returns nil, unless it names a URL host as its canonical value
// (ADR 007 section 3.5): that value is then the host of the argument, and nothing is read, because the object does not
// exist yet. The target's canonical value is the reply field the row names, or the source ID when the row names none. A
// URL host, when the row names one, replaces it. A target that does not resolve stops the call.
func readTarget(ctx context.Context, svc Service, row operations.Row, src domain.Source, args domain.Args) (*resolvedTarget, error) {
	value := criticalValue(row, args)
	if row.Target == nil {
		if value == "" {
			return nil, nil
		}
		return &resolvedTarget{Value: value}, nil
	}
	reader, err := svc.Gateways.Reader(src)
	if err != nil {
		return nil, err
	}
	read, ok := operations.Lookup(row.Target.Read)
	if !ok {
		return nil, domain.ErrUnsupportedOperation
	}
	raw := map[string]any{}
	if name := row.Positional(); name != "" {
		if _, declared := read.Param(name); declared {
			raw[name] = args.Values()[name]
		}
	}
	if _, declared := read.Param("limit"); declared && read.IsList() {
		// A list target is counted on its largest page, so the count reaches as far as the server allows (finding m7).
		raw["limit"] = domain.MaxPageLimit
	}
	bound, err := operations.Bind(read, raw)
	if err != nil {
		return nil, err
	}
	rec, err := reader.Read(ctx, row.Target.Read, bound)
	if err != nil {
		return nil, err
	}
	target := &resolvedTarget{Value: src.ID().String(), Name: targetField(rec, row.Target.Name)}
	if row.Target.Field != "" {
		target.Value = targetField(rec, row.Target.Field)
		if target.Value == "" {
			return nil, domain.ErrUnexpectedResponse
		}
	}
	if value != "" {
		target.Value = value
	}
	if read.IsList() {
		target.Count = len(rec.Items())
		target.HasCount = true
		target.FullPage = target.Count >= domain.MaxPageLimit
	}
	return target, nil
}

// criticalValue returns the canonical value that a critical call names from its own arguments (ADR 007 section 3.5): the
// host of the URL argument the row names, or the exact text of the argument it names, such as the name of a new OAuth
// client. It returns "" when the row names neither, so the target is read instead. Bind has already checked the URL kind,
// so a bound URL always has a host.
func criticalValue(row operations.Row, args domain.Args) string {
	if row.Critical == nil {
		return ""
	}
	if row.Critical.Host != "" {
		return operations.HostOf(args.String(row.Critical.Host))
	}
	if row.Critical.Text != "" {
		return args.String(row.Critical.Text)
	}
	return ""
}

// dryRunAdminPlan returns the plan of a dry run of an admin call: the write plan, with the target that was read and, for
// a list read, the number of its items. A row with no target adds no target field.
func dryRunAdminPlan(row operations.Row, src domain.Source, args domain.Args, target *resolvedTarget) domain.Record {
	if target == nil {
		return dryRunPlan(row, src, args)
	}
	extra := []domain.Field{{Name: "target", Value: target.Value}}
	if row.Target != nil && row.Target.Name != "" {
		extra = append(extra, domain.Field{Name: "targetName", Value: target.Name})
	}
	if target.HasCount {
		extra = append(extra, domain.Field{Name: "targetCount", Value: targetCountValue(target)})
	}
	return dryRunPlan(row, src, args, extra...)
}

// targetCountValue returns the count a dry run shows for a list target: the exact count, or "at least N" when the page of N
// is full, because the list may hold more items (finding m7).
func targetCountValue(target *resolvedTarget) any {
	if target.FullPage {
		return fmt.Sprintf("at least %d", target.Count)
	}
	return target.Count
}

// refuseOwnUser refuses a change to the user of the API key (ADR 007 section 3.5, rule 6). It reads the key's user with
// auth.whoami and compares the userId of the call with that user's ID, without a difference of case. A reply without the
// user's ID is refused too, so the call never goes ahead on a guess. The refusal is ErrSelfTarget and makes no change.
func refuseOwnUser(ctx context.Context, svc Service, src domain.Source, args domain.Args) error {
	self, err := ownUserID(ctx, svc, src)
	if err != nil {
		return err
	}
	if strings.EqualFold(args.String("userId"), self) {
		return domain.ErrSelfTarget
	}
	return nil
}

// ownUserID returns the ID of the user of the API key, which auth.whoami names under user.
func ownUserID(ctx context.Context, svc Service, src domain.Source) (string, error) {
	reader, err := svc.Gateways.Reader(src)
	if err != nil {
		return "", err
	}
	// auth.whoami takes no argument, so its arguments are empty and need no binding.
	rec, err := reader.Read(ctx, domain.OpAuthWhoami, domain.NewArgs(map[string]any{}))
	if err != nil {
		return "", err
	}
	self := rec.Rec("user").Str("id")
	if self == "" {
		return "", domain.ErrUnexpectedResponse
	}
	return self, nil
}

// targetField returns the text at a field path of a reply, such as id or team.name, or "" when the path holds no text.
// A dot steps into a nested object of the reply.
func targetField(rec domain.Record, path string) string {
	name, rest, nested := strings.Cut(path, ".")
	if !nested {
		return rec.Str(name)
	}
	return targetField(rec.Rec(name), rest)
}
