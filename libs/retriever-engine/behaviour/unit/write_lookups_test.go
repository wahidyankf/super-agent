package unit_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// The lookups of these bindings turn a name into an ID with one read before a write. Each read is a list of categories.

// lookupFindDoc is a read of the categories that declares a name parameter, so the name travels to the source.
const lookupFindDoc = `  - id: example-wiki.categories.find
    summary: Find categories by name.
    access: read
    method: GET
    path: /categories
    params:
      - name: name
        type: Text
        in: query
        required: false
    response:
      items: /items
      fields:
        id:
          pointer: /id
        name:
          pointer: /name
    examples:
      - request:
          name: Engineering
        response:
          status: 200
          body:
            items: [{id: 7, name: Engineering}]
        expect:
          id: 7
`

// lookupListDoc is a read of every category that declares no parameter, so the lookup matches by the name field alone.
const lookupListDoc = `  - id: example-wiki.categories.list
    summary: List categories.
    access: read
    method: GET
    path: /categories
    response:
      items: /items
      fields:
        id:
          pointer: /id
        name:
          pointer: /name
    examples:
      - request: {}
        response:
          status: 200
          body:
            items: [{id: 7, name: Engineering}]
        expect:
          id: 7
`

// lookupTeamDoc is a read of the categories that declares a team parameter beside the name parameter.
const lookupTeamDoc = `  - id: example-wiki.categories.team
    summary: Find categories by name for one team.
    access: read
    method: GET
    path: /categories
    params:
      - name: name
        type: Text
        in: query
        required: false
      - name: team
        type: Text
        in: query
        required: false
    response:
      items: /items
      fields:
        id:
          pointer: /id
        name:
          pointer: /name
    examples:
      - request:
          name: Engineering
          team: ops
        response:
          status: 200
          body:
            items: [{id: 7, name: Engineering}]
        expect:
          id: 7
`

// lookupPagedDoc is a read of the categories by a page-all rule, which a lookup refuses.
const lookupPagedDoc = `  - id: example-wiki.categories.paged
    summary: List categories page by page.
    access: read
    method: GET
    path: /categories
    pagination:
      style: page-all
      max_pages: 5
      page_size: 2
    response:
      items: /items
      fields:
        id:
          pointer: /id
        name:
          pointer: /name
    examples:
      - request: {}
        response:
          status: 200
          body:
            items: [{id: 7, name: Engineering}]
        expect:
          id: 7
`

// lookupWriteDoc is the assign write of the example wiki. Its category parameter is looked up by name through the read, and
// its team parameter is optional. The required argument decides whether the category must be given.
func lookupWriteDoc(read, required string) string {
	return `  - id: example-wiki.docs.assign
    summary: Assign a document to a category by its name.
    access: write
    method: POST
    path: /docs/assign
    params:
      - name: category
        type: Text
        in: body
        required: ` + required + `
        resolve:
          operation: ` + read + `
          field: name
      - name: team
        type: Text
        in: body
        required: false
    body:
      category: "{{category}}"
      team: "{{team}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          category: Engineering
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`
}

// lookupReply returns a reply of the lookup read with the given categories, as the JSON of its items.
func lookupReply(items string) rebindReply {
	return rebindJSON(http.StatusOK, `{"items":[`+items+`]}`)
}

// lookupAssign runs one assign write of the example wiki with the given arguments.
func lookupAssign(t *testing.T, eng *engine.Engine, args map[string]any) (domain.Record, error) {
	t.Helper()
	return eng.Write(context.Background(), "example-wiki.docs.assign", domain.NewArgs(args))
}

// lookupMethods returns the methods of the requests that the transport received, in order.
func lookupMethods(tr *rebindTransport) []string {
	var out []string
	for _, r := range tr.seen() {
		out = append(out, r.method)
	}
	return out
}

// TestLookups_MatchedEntrySentWithItsID binds a write named by one matching entry to a write that carries the ID of that
// entry and not the name.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A write named by one matching entry is sent with the ID of that entry
func TestLookups_MatchedEntrySentWithItsID(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.find", "true")+lookupFindDoc)
	tr := rebindReplies(
		lookupReply(`{"id":7,"name":"Engineering"},{"id":8,"name":"Marketing"}`),
		rebindJSON(http.StatusCreated, `{"id":"doc-9"}`),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := lookupAssign(t, eng, map[string]any{"category": "Engineering"}); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	body := writesBody(t, tr)
	if got := fmt.Sprint(body["category"]); got != "7" {
		t.Fatalf("the write carries %v, want the ID 7 and not the name", body["category"])
	}
}

// TestLookups_NoSingleMatchUnresolvedNoWrite binds a name that matches two entries, or none, to the unresolved-name error,
// with no write sent.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A name that matches <matches> fails with the unresolved-name error and no write is sent
func TestLookups_NoSingleMatchUnresolvedNoWrite(t *testing.T) {
	rows := []struct {
		matches, items string
	}{
		{matches: "two entries", items: `{"id":7,"name":"Engineering"},{"id":8,"name":"Engineering"}`},
		{matches: "no entry", items: `{"id":7,"name":"Marketing"}`},
	}
	for _, row := range rows {
		t.Run(row.matches, func(t *testing.T) {
			s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.find", "true")+lookupFindDoc)
			tr := rebindReplies(lookupReply(row.items), rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := lookupAssign(t, eng, map[string]any{"category": "Engineering"})
			if !errors.Is(err, domain.ErrUnresolvedName) {
				t.Fatalf("the error is %v, want the unresolved-name error", err)
			}
			for _, m := range lookupMethods(tr) {
				if m != http.MethodGet {
					t.Fatalf("a %s was sent, want only the lookup read", m)
				}
			}
		})
	}
}

// TestLookups_FailedLookupReadMapsAsReadAndSendsNoWrite binds a lookup read that fails to the error of a failed read, and no
// write request.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A lookup whose single read fails maps its error as a read does and sends no write
func TestLookups_FailedLookupReadMapsAsReadAndSendsNoWrite(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.find", "true")+lookupFindDoc)
	tr := rebindReplies(timingRetryAfter(""), rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := lookupAssign(t, eng, map[string]any{"category": "Engineering"})
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("the error is %v, want the error of the failed read", err)
	}
	for _, m := range lookupMethods(tr) {
		if m == http.MethodPost {
			t.Fatalf("the write was sent after a failed lookup")
		}
	}
}

// TestLookups_ReadWithoutNameParamMatchesByField binds a lookup read that declares no name parameter to a read of every entry,
// with the one entry found by its field.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A lookup whose read declares no name parameter reads every entry and finds the one that matches by its field
func TestLookups_ReadWithoutNameParamMatchesByField(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.list", "true")+lookupListDoc)
	tr := rebindReplies(
		lookupReply(`{"id":7,"name":"Engineering"},{"id":8,"name":"Marketing"}`),
		rebindJSON(http.StatusCreated, `{"id":"doc-9"}`),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := lookupAssign(t, eng, map[string]any{"category": "Marketing"}); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 || reqs[0].url.Query().Has("name") {
		t.Fatalf("the lookup read sends a name, want no name in the read")
	}
	if got := fmt.Sprint(writesBody(t, tr)["category"]); got != "8" {
		t.Fatalf("the write carries %s, want the entry that matches by its field, 8", got)
	}
}

// TestLookups_TextIDSentAsItCame binds a lookup entry with a text ID to the write carrying that text as it came.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A lookup whose matched entry carries a text ID sends that text as it came
func TestLookups_TextIDSentAsItCame(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.list", "true")+lookupListDoc)
	tr := rebindReplies(
		lookupReply(`{"id":"cat-7","name":"Engineering"}`),
		rebindJSON(http.StatusCreated, `{"id":"doc-9"}`),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := lookupAssign(t, eng, map[string]any{"category": "Engineering"}); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	if got := writesBody(t, tr)["category"]; got != "cat-7" {
		t.Fatalf("the write carries %v, want the text ID as it came", got)
	}
}

// TestLookups_UnsetLookupParamRunsNoRead binds an optional lookup parameter that the caller leaves unset to no lookup read and
// no key in the body.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A lookup parameter the caller leaves unset runs no read and its key is left out of the body
func TestLookups_UnsetLookupParamRunsNoRead(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.find", "false")+lookupFindDoc)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := lookupAssign(t, eng, map[string]any{}); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	if got := strings.Join(lookupMethods(tr), ","); got != "POST" {
		t.Fatalf("the requests are %s, want only the write", got)
	}
	if _, present := writesBody(t, tr)["category"]; present {
		t.Fatalf("the body holds a category key, want none")
	}
}

// TestLookups_LookupReadGetsCallerArgumentOfSameName binds a lookup read that declares a parameter the caller sets to the
// caller's argument, with the write keeping its own argument too.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A lookup read passes the caller's argument of the same name when it declares that parameter
func TestLookups_LookupReadGetsCallerArgumentOfSameName(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.team", "true")+lookupTeamDoc)
	tr := rebindReplies(
		lookupReply(`{"id":7,"name":"Engineering"}`),
		rebindJSON(http.StatusCreated, `{"id":"doc-9"}`),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := lookupAssign(t, eng, map[string]any{"category": "Engineering", "team": "ops"}); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 || reqs[0].url.Query().Get("team") != "ops" {
		t.Fatalf("the lookup read carries team %q, want the caller's argument", reqs[0].url.Query().Get("team"))
	}
	body := writesBody(t, tr)
	if body["team"] != "ops" || fmt.Sprint(body["category"]) != "7" {
		t.Fatalf("the write body holds team %v and category %v, want its own team argument and the ID", body["team"], body["category"])
	}
}

// TestLookups_PagedLookupRefusedBeforeRequest binds a write whose parameter is looked up through a paged read to a refusal
// with no request sent.
// @scenario specs/libs/retriever-engine/behaviours/write-lookups.feature: A write whose lookup is over a paged read is refused before any request
func TestLookups_PagedLookupRefusedBeforeRequest(t *testing.T) {
	s := rebindSpec(t, lookupWriteDoc("example-wiki.categories.paged", "true")+lookupPagedDoc)
	tr := rebindReplies(lookupReply(`{"id":7,"name":"Engineering"}`), rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := lookupAssign(t, eng, map[string]any{"category": "Engineering"})
	if err == nil || len(tr.seen()) != 0 {
		t.Fatalf("the write is served with %d requests, want a refusal and no request", len(tr.seen()))
	}
}
