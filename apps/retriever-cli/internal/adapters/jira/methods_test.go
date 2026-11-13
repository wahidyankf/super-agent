package jira

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestTheEndpointTableHoldsExactlyTheFiveRows pins the closed table to the five requests.
func TestTheEndpointTableHoldsExactlyTheFiveRows(t *testing.T) {
	want := []string{
		"GET /rest/api/3/myself",
		"GET /rest/api/3/project/search",
		"POST /rest/api/3/search/jql",
		"GET /rest/api/3/issue/{key}",
		"GET /rest/api/3/issue/{key}/comment",
	}
	var got []string
	for _, ep := range allowed {
		got = append(got, ep.method+" "+ep.path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("table = %v, want %v", got, want)
	}
}

// TestTheTableHoldsOneReadingPostAndNoWriteMethod proves the only POST is the JQL search, which reads, and no PUT, PATCH,
// or DELETE is in the table.
func TestTheTableHoldsOneReadingPostAndNoWriteMethod(t *testing.T) {
	posts := 0
	for _, ep := range allowed {
		switch ep.method {
		case http.MethodGet:
		case http.MethodPost:
			posts++
			if ep.path != "/rest/api/3/search/jql" {
				t.Fatalf("POST to %s; only the JQL search may use POST", ep.path)
			}
		default:
			t.Fatalf("method %s is in the table; Jira is read only", ep.method)
		}
	}
	if posts != 1 {
		t.Fatalf("%d POST rows, want exactly 1", posts)
	}
}

// TestTheRemovedLegacySearchPathIsAbsent proves the old /rest/api/3/search path, removed by Atlassian, is not a row.
func TestTheRemovedLegacySearchPathIsAbsent(t *testing.T) {
	for _, ep := range allowed {
		if ep.path == "/rest/api/3/search" {
			t.Fatal("the removed /rest/api/3/search path is in the table")
		}
	}
}

// TestPermittedRefusesEveryRowOutsideTheTable proves a request the table does not name is not permitted, and every row
// of the table is.
func TestPermittedRefusesEveryRowOutsideTheTable(t *testing.T) {
	outside := []endpoint{
		{http.MethodGet, "/rest/api/3/search"},
		{http.MethodPut, "/rest/api/3/issue/{key}"},
		{http.MethodPatch, "/rest/api/3/issue/{key}"},
		{http.MethodDelete, "/rest/api/3/issue/{key}"},
		{http.MethodPost, "/rest/api/3/issue/{key}/comment"},
		{http.MethodPost, "/rest/api/3/myself"},
	}
	for _, ep := range outside {
		if permitted(ep) {
			t.Fatalf("permitted(%s %s) is true; the table does not hold it", ep.method, ep.path)
		}
	}
	for _, ep := range allowed {
		if !permitted(ep) {
			t.Fatalf("permitted(%s %s) is false for a table row", ep.method, ep.path)
		}
	}
}

// TestTheRequestBuilderRefusesARowOutsideTheTable proves a request cannot be built from a row the table does not hold.
func TestTheRequestBuilderRefusesARowOutsideTheTable(t *testing.T) {
	g := &Gateway{base: testBase, host: testHost}
	_, err := g.newRequest(context.Background(), endpoint{http.MethodPost, "/rest/api/3/search"}, "", nil, nil)
	if !errors.Is(err, errMethodNotAllowed) {
		t.Fatalf("newRequest outside the table = %v, want errMethodNotAllowed", err)
	}
}

// TestTheRequestBuilderSubstitutesOnlyAKeyThatPassesThePattern proves an issue path takes a checked key and no other text.
func TestTheRequestBuilderSubstitutesOnlyAKeyThatPassesThePattern(t *testing.T) {
	g := &Gateway{base: testBase, host: testHost}
	req, err := g.newRequest(context.Background(), epIssue, "EXAMPLE-12", nil, nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	if req.URL.Path != "/rest/api/3/issue/EXAMPLE-12" || req.Method != http.MethodGet {
		t.Fatalf("request %s %s, want GET /rest/api/3/issue/EXAMPLE-12", req.Method, req.URL.Path)
	}
	for _, bad := range []string{"example-12", "EXAMPLE-0", "EXAMPLE", "EXAMPLE-12/../../x", "EXAMPLE-12?x=1", ""} {
		if _, err := g.newRequest(context.Background(), epIssue, bad, nil, nil); !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Fatalf("key %q: err = %v, want ErrInvalidDocumentID before any request", bad, err)
		}
	}
}

// TestTheRequestBuilderEncodesTheQueryAndSetsTheHostHeader proves query values are encoded, and the Host header is the
// endpoint host.
func TestTheRequestBuilderEncodesTheQueryAndSetsTheHostHeader(t *testing.T) {
	g := &Gateway{base: testBase, host: testHost}
	query := url.Values{}
	query.Set("query", "a&b=c")
	req, err := g.newRequest(context.Background(), epProjectSearch, "", query, nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	if req.URL.RawQuery != "query=a%26b%3Dc" {
		t.Fatalf("raw query = %q, want the encoded value", req.URL.RawQuery)
	}
	if req.Host != testHost {
		t.Fatalf("Host = %q, want %q", req.Host, testHost)
	}
}

// TestTheRequestBuilderCarriesTheBodyAsJSON proves a POST body is sent with its JSON content type.
func TestTheRequestBuilderCarriesTheBodyAsJSON(t *testing.T) {
	g := &Gateway{base: testBase, host: testHost}
	req, err := g.newRequest(context.Background(), epJQLSearch, "", nil, []byte(`{"jql":"project = X"}`))
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	body, _ := io.ReadAll(req.Body)
	if req.Method != http.MethodPost || string(body) != `{"jql":"project = X"}` {
		t.Fatalf("POST body = %q (method %s)", body, req.Method)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
}
