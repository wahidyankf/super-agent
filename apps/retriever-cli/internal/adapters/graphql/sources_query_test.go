package graphql

import (
	"testing"

	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// sendServed sends one query to a handler over svc and returns the decoded reply.
func sendServed(t *testing.T, svc usecases.Service, query string) reply {
	t.Helper()
	h, err := NewHandler(
		Deps{Service: svc, Logger: discardLogger()},
		WithListener(tcpListener("127.0.0.1", testPort)),
	)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return decode(t, sendTo(t, h, graphqlPOST(query)))
}

// sourceRows returns the rows of the sources field of a reply that has no errors.
func sourceRows(t *testing.T, r reply) []map[string]any {
	t.Helper()
	if len(r.Errors) != 0 {
		t.Fatalf("reply has errors %v, want none", r.messages())
	}
	raw, ok := r.Data["sources"].([]any)
	if !ok {
		t.Fatalf("data.sources is %T, want a list", r.Data["sources"])
	}
	rows := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a source row is %T, want an object", item)
		}
		rows = append(rows, row)
	}
	return rows
}

// TestSourcesQueryListsEachSourceWithKindAndTags checks each row of the sources query. A row carries only the three
// fields of the Source type that remain: id, kind, and tags.
func TestSourcesQueryListsEachSourceWithKindAndTags(t *testing.T) {
	rows := sourceRows(t, sendServed(t, twoOutlineSources(nil), `{ sources { id kind tags } }`))
	if len(rows) != 2 {
		t.Fatalf("sources returned %d rows, want 2", len(rows))
	}
	if rows[0]["id"] != "outline" || rows[0]["kind"] != "outline" {
		t.Errorf("first row = %v, want outline, outline", rows[0])
	}
	if rows[1]["id"] != "outline-engineering" {
		t.Errorf("second row = %v, want outline-engineering", rows[1])
	}
	if tags, _ := rows[1]["tags"].([]any); len(tags) != 2 {
		t.Errorf("second row tags = %v, want two tags", rows[1]["tags"])
	}
}

// TestSourcesQueryAppliesKindAndAllTags checks that the filter is an all-of filter over the given tags.
func TestSourcesQueryAppliesKindAndAllTags(t *testing.T) {
	rows := sourceRows(t, sendServed(t, twoOutlineSources(nil),
		`{ sources(kind: "outline", tags: ["wiki", "engineering"]) { id } }`))
	if len(rows) != 1 || rows[0]["id"] != "outline-engineering" {
		t.Fatalf("sources = %v, want only outline-engineering", rows)
	}
}

// TestSourcesQueryWithNoMatchReturnsAnEmptyList checks that a filter with no match is an empty list, not null, because
// the field is non-null.
func TestSourcesQueryWithNoMatchReturnsAnEmptyList(t *testing.T) {
	rows := sourceRows(t, sendServed(t, twoOutlineSources(nil),
		`{ sources(tags: ["company", "engineering"]) { id } }`))
	if len(rows) != 0 {
		t.Fatalf("sources = %v, want an empty list", rows)
	}
}

// TestSourcesQueryCannotReachTheEndpoint checks that asking for an endpoint is a validation error.
func TestSourcesQueryCannotReachTheEndpoint(t *testing.T) {
	r := sendServed(t, twoOutlineSources(nil), `{ sources { id endpoint } }`)
	assertErrorContains(t, r, "endpoint")
}

// TestSourcesQueryRefusesAnUnknownKind checks the safe text for a kind the code does not know.
func TestSourcesQueryRefusesAnUnknownKind(t *testing.T) {
	r := sendServed(t, twoOutlineSources(nil), `{ sources(kind: "notion") { id } }`)
	assertErrorContains(t, r, "unknown source kind (known: outline, kiwi, jira)")
}

// TestCollectionsWithoutASourceIsAmbiguousWithTwoSources checks that an omitted source with two outline sources
// stops before any request, and names the sources.
func TestCollectionsWithoutASourceIsAmbiguousWithTwoSources(t *testing.T) {
	gw := &countingGateway{}
	r := sendServed(t, twoOutlineSources(gw), `{ collections { id } }`)
	assertErrorContains(t, r, "more than one outline source exists (outline, outline-engineering); pass the source argument")
	if n := gw.calls.Load(); n != 0 {
		t.Fatalf("gateway calls = %d, want 0", n)
	}
}
