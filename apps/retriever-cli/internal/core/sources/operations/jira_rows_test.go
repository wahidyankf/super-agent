package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// jiraWant is the expected shape of one Jira row (ADR 010 section 2). Each row is a read: no write, admin, critical, or
// MCP tool.
type jiraWant struct {
	id         domain.OperationID
	cli        string
	positional string
	fields     []string
}

// jiraWants are the five Jira rows, in the order of the ADR table.
var jiraWants = []jiraWant{
	{domain.OpJiraWhoami, "jira whoami", "", []string{"displayName", "accountType", "active"}},
	{domain.OpJiraProjectsList, "jira projects list", "", []string{"key", "name", "type", "style"}},
	{domain.OpJiraIssuesSearch, "jira issues search", "jql", []string{"key", "summary", "status", "type", "priority", "assignee", "updated", "projectKey", "truncated"}},
	{domain.OpJiraIssuesGet, "jira issues get", "issueKey", []string{"id", "title", "url", "markdown", "collectionId"}},
	{domain.OpJiraCommentsList, "jira comments list", "issueKey", []string{"id", "author", "created", "updated", "markdown"}},
}

// TestJiraRowsAreTheFiveReadRows checks the catalog filter for the Jira kind: exactly the five rows of ADR 010, in order,
// and no other row belongs to it.
func TestJiraRowsAreTheFiveReadRows(t *testing.T) {
	jira := RowsOf(domain.KindJira)
	if len(jira) != len(jiraWants) {
		t.Fatalf("RowsOf(jira) holds %d rows, want %d", len(jira), len(jiraWants))
	}
	for i, want := range jiraWants {
		if jira[i].ID != want.id {
			t.Errorf("RowsOf(jira)[%d] = %s, want %s", i, jira[i].ID, want.id)
		}
	}
}

// TestJiraRowsHaveTheADRShape checks each Jira row: its path, its effect and access, its positional parameter, its result
// fields, and that it has no MCP tool and no critical or target read.
func TestJiraRowsHaveTheADRShape(t *testing.T) {
	for _, want := range jiraWants {
		row, ok := Lookup(want.id)
		if !ok {
			t.Fatalf("Lookup(%s) found no row", want.id)
		}
		if row.CLI != want.cli {
			t.Errorf("%s CLI = %q, want %q", want.id, row.CLI, want.cli)
		}
		if row.Effect != domain.EffectRead || row.Access != domain.AccessRead {
			t.Errorf("%s is %s/%s, want read/read", want.id, row.Effect, row.Access)
		}
		if row.Tool != "" {
			t.Errorf("%s has MCP tool %q, want none (ADR 010 section 4)", want.id, row.Tool)
		}
		if row.Critical != nil || row.Target != nil {
			t.Errorf("%s has a critical or target read, want none", want.id)
		}
		if row.NeedsYes() || row.NotIdempotent || row.SameOnRepeat || row.RefuseSelf {
			t.Errorf("%s has a change flag, want none", want.id)
		}
		if !slices.Equal(row.Fields, want.fields) {
			t.Errorf("%s Fields = %v, want %v", want.id, row.Fields, want.fields)
		}
		if got := row.Positional(); got != want.positional {
			t.Errorf("%s positional = %q, want %q", want.id, got, want.positional)
		}
		if k, ok := row.Kind(); !ok || k != domain.KindJira {
			t.Errorf("%s kind = %v, %v; want jira", want.id, k, ok)
		}
	}
}

// TestJiraRowParametersMatchTheADR checks the parameters of the four rows that take any: the kind and bounds of each
// (ADR 010 section 2).
func TestJiraRowParametersMatchTheADR(t *testing.T) {
	cases := []struct {
		id   domain.OperationID
		name string
		kind Kind
		req  bool
		def  any
	}{
		{domain.OpJiraIssuesSearch, "jql", KindQuery, true, nil},
		{domain.OpJiraIssuesSearch, "limit", KindLimit, false, 50},
		{domain.OpJiraIssuesGet, "issueKey", KindIssueKey, true, nil},
		{domain.OpJiraCommentsList, "issueKey", KindIssueKey, true, nil},
		{domain.OpJiraCommentsList, "limit", KindLimit, false, 50},
		{domain.OpJiraCommentsList, "offset", KindOffset, false, 0},
		{domain.OpJiraProjectsList, "query", KindQuery, false, nil},
		{domain.OpJiraProjectsList, "limit", KindInt, false, 25},
		{domain.OpJiraProjectsList, "offset", KindOffset, false, 0},
	}
	for _, tc := range cases {
		row, _ := Lookup(tc.id)
		p, ok := row.Param(tc.name)
		if !ok {
			t.Errorf("%s has no parameter %s", tc.id, tc.name)
			continue
		}
		if p.Kind != tc.kind || p.Required != tc.req {
			t.Errorf("%s.%s is %s required=%v, want %s required=%v", tc.id, tc.name, p.Kind, p.Required, tc.kind, tc.req)
		}
		if tc.def != nil && p.Default != tc.def {
			t.Errorf("%s.%s default = %v, want %v", tc.id, tc.name, p.Default, tc.def)
		}
	}
	projects, _ := Lookup(domain.OpJiraProjectsList)
	limit, _ := projects.Param("limit")
	if limit.Min != 1 || limit.Max != 50 {
		t.Errorf("projects limit bounds = %d to %d, want 1 to 50", limit.Min, limit.Max)
	}
}

// TestSupportsJiraRunsOnlyItsFiveRows checks the closed list of the Jira kind: each Jira row is supported by the Jira
// kind, and no Outline or Kiwi row is, and no other kind runs a Jira row.
func TestSupportsJiraRunsOnlyItsFiveRows(t *testing.T) {
	for _, row := range Rows() {
		want := slices.ContainsFunc(jiraWants, func(w jiraWant) bool { return w.id == row.ID })
		if got := Supports(domain.KindJira, row); got != want {
			t.Errorf("Supports(jira, %s) = %v, want %v", row.ID, got, want)
		}
	}
	for _, want := range jiraWants {
		row, _ := Lookup(want.id)
		if Supports(domain.KindOutline, row) || Supports(domain.KindKiwi, row) {
			t.Errorf("an Outline or Kiwi kind runs %s", want.id)
		}
	}
	if Supports(domain.SourceKind{}, jiraWants[0].rowOrFail(t)) {
		t.Error("a zero kind runs a Jira row")
	}
}

// rowOrFail returns the catalog row of the Jira want, and fails the test when the catalog has none.
func (w jiraWant) rowOrFail(t *testing.T) Row {
	t.Helper()
	row, ok := Lookup(w.id)
	if !ok {
		t.Fatalf("Lookup(%s) found no row", w.id)
	}
	return row
}

// TestJiraRowsRefuseNonIssueKeyForIssueKeyParameter checks the KindIssueKey validator: a strict upper-case issue key of
// ADR 010 section 2 passes, and any other text is refused with ErrInvalidDocumentID before a request.
func TestJiraRowsRefuseNonIssueKeyForIssueKeyParameter(t *testing.T) {
	row, _ := Lookup(domain.OpJiraIssuesGet)
	for _, good := range []string{"EXAMPLE-12", "OPS-3", "AB-1", "EXAMPLE-999999999"} {
		if _, err := Bind(row, map[string]any{"issueKey": good}); err != nil {
			t.Errorf("Bind(issueKey %q) error = %v, want none", good, err)
		}
	}
	bad := []any{
		"example-12", "EXAMPLE-0", "EXAMPLE-012", "EXAMPLE12", "EXAMPLE-1/x", "project-EXAMPLE",
		"A-1", "EXAMPLE-1000000000", "ABCDEFGHIJKLMNOPQRSTU-1", "EXAMPLE-12 ", 12, nil,
	}
	for _, v := range bad {
		_, err := Bind(row, map[string]any{"issueKey": v})
		if v == nil {
			if !errors.Is(err, domain.ErrMissingArgument) {
				t.Errorf("Bind(issueKey nil) error = %v, want ErrMissingArgument", err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Errorf("Bind(issueKey %v) error = %v, want ErrInvalidDocumentID", v, err)
		}
	}
}

// TestValidateChecksAnIssueKeyAgainTheSameWay checks that the use case's second check (Validate) applies the same
// KindIssueKey rule, so a value that reaches the Jira adapter has passed it.
func TestValidateChecksAnIssueKeyAgainTheSameWay(t *testing.T) {
	row, _ := Lookup(domain.OpJiraCommentsList)
	args := domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"})
	if err := Validate(row, args); err != nil {
		t.Fatalf("Validate(EXAMPLE-12) error = %v", err)
	}
	args = domain.NewArgs(map[string]any{"issueKey": "example-12"})
	if err := Validate(row, args); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("Validate(example-12) error = %v, want ErrInvalidDocumentID", err)
	}
}
