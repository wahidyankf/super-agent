package jiraseam

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestProjectCollectionIDMapsOnlyStrictProjectKeys checks the project key pattern and the collection ID it forms.
func TestProjectCollectionIDMapsOnlyStrictProjectKeys(t *testing.T) {
	ok := []string{"EXAMPLE", "AB", "A_1", "ABCDEFGHIJKLMNOPQRST"} // 2 to 20 characters
	for _, key := range ok {
		id, err := ProjectCollectionID(key)
		if err != nil {
			t.Errorf("ProjectCollectionID(%q) = %v, want nil", key, err)
			continue
		}
		if id.String() != "project-"+key {
			t.Errorf("ProjectCollectionID(%q) = %q, want project-%s", key, id.String(), key)
		}
	}
	bad := []string{"", "A", "example", "EXA EMP", "EXA-EMP", "1ABC", "ABCDEFGHIJKLMNOPQRSTU", "EXA/EMP", "EXA\\n"}
	for _, key := range bad {
		if _, err := ProjectCollectionID(key); !errors.Is(err, domain.ErrInvalidCollectionID) {
			t.Errorf("ProjectCollectionID(%q) = %v, want ErrInvalidCollectionID", key, err)
		}
	}
}

// TestProjectKeyReadsOnlyProjectIDs checks that a collection ID is read back to its key, and that any other shape is refused.
func TestProjectKeyReadsOnlyProjectIDs(t *testing.T) {
	id, err := domain.NewCollectionID("project-EXAMPLE")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	key, err := ProjectKey(id)
	if err != nil || key != "EXAMPLE" {
		t.Fatalf("ProjectKey = %q, %v; want EXAMPLE, nil", key, err)
	}
	for _, raw := range []string{"example", "project-example", "project-A", "issue-EXAMPLE-1", "project-", "EXAMPLE"} {
		bad, err := domain.NewCollectionID(raw)
		if err != nil {
			t.Fatalf("NewCollectionID(%q): %v", raw, err)
		}
		if _, err := ProjectKey(bad); !errors.Is(err, domain.ErrInvalidCollectionID) {
			t.Errorf("ProjectKey(%q) = %v, want ErrInvalidCollectionID", raw, err)
		}
	}
}

// TestParseIssueKeyAcceptsOnlyStrictUpperCaseKeys checks the issue key pattern, which is refused rather than corrected.
func TestParseIssueKeyAcceptsOnlyStrictUpperCaseKeys(t *testing.T) {
	ok := []string{"EXAMPLE-1", "EXAMPLE-12", "AB-100", "A_B-999999999", "ABCDEFGHIJKLMNOPQRST-1"}
	for _, key := range ok {
		got, err := ParseIssueKey(key)
		if err != nil || got != key {
			t.Errorf("ParseIssueKey(%q) = %q, %v; want the key, nil", key, got, err)
		}
	}
	bad := []string{
		"",
		"example-12",
		"EXAMPLE-0",
		"EXAMPLE-012",
		"EXAMPLE-",
		"EXAMPLE-1000000000",
		"EXAMPLE12",
		"A-1",
		"EXAMPLE-1 ",
		" EXAMPLE-1",
		"EXAMPLE-1\\n",
		"EXAMPLE-1/../x",
		"EXAMPLE-1?x=1",
		"EXAMPLE-1&x",
		"project-EXAMPLE",
		"ABCDEFGHIJKLMNOPQRSTU-1",
	}
	for _, key := range bad {
		if _, err := ParseIssueKey(key); !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Errorf("ParseIssueKey(%q) = %v, want ErrInvalidDocumentID", key, err)
		}
	}
}

// TestIssueDocumentIDRoundTrip checks that an issue key becomes a document ID and reads back to the same key.
func TestIssueDocumentIDRoundTrip(t *testing.T) {
	id, err := IssueDocumentID("EXAMPLE-12")
	if err != nil {
		t.Fatalf("IssueDocumentID: %v", err)
	}
	if id.String() != "issue-EXAMPLE-12" {
		t.Fatalf("IssueDocumentID = %q, want issue-EXAMPLE-12", id.String())
	}
	key, err := IssueKey(id)
	if err != nil || key != "EXAMPLE-12" {
		t.Fatalf("IssueKey = %q, %v; want EXAMPLE-12, nil", key, err)
	}
	if _, err := IssueDocumentID("example-12"); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("IssueDocumentID(lower case) = %v, want ErrInvalidDocumentID", err)
	}
}

// TestIssueKeyReadsOnlyIssueIDs checks that a project ID in a document position is refused.
func TestIssueKeyReadsOnlyIssueIDs(t *testing.T) {
	for _, raw := range []string{"project-EXAMPLE", "issue-example-12", "issue-EXAMPLE-0", "issue-", "EXAMPLE-12"} {
		id, err := domain.NewDocumentID(raw)
		if err != nil {
			t.Fatalf("NewDocumentID(%q): %v", raw, err)
		}
		if _, err := IssueKey(id); !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Errorf("IssueKey(%q) = %v, want ErrInvalidDocumentID", raw, err)
		}
	}
}

// TestCheckBaseURLRefusesABadBase checks the connection base rules: an unparsable base, a non-HTTPS scheme, and a
// base with no host or with user information are refused, and an HTTPS base with a host passes.
func TestCheckBaseURLRefusesABadBase(t *testing.T) {
	if err := CheckBaseURL("https://[::1"); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("CheckBaseURL error = %v, want errInvalidBaseURL", err)
	}
	if err := CheckBaseURL("http://127.0.0.1:9443"); !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("CheckBaseURL error = %v, want ErrHTTPSRequired", err)
	}
	if err := CheckBaseURL("https://user@127.0.0.1:9443"); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("CheckBaseURL error = %v, want errInvalidBaseURL", err)
	}
	if err := CheckBaseURL("https://jira.example.test"); err != nil {
		t.Fatalf("CheckBaseURL = %v, want nil", err)
	}
}

// TestRefuseRedirectsKeepsTheLastResponse checks that the seam client stops at the first redirect.
func TestRefuseRedirectsKeepsTheLastResponse(t *testing.T) {
	if err := refuseRedirects(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("refuseRedirects = %v, want ErrUseLastResponse", err)
	}
}

// TestKindErrorNamesJiraOnce checks that a plain failure is named for Jira exactly once, and that a nil, a
// cancellation, and an already-named failure pass through unchanged.
func TestKindErrorNamesJiraOnce(t *testing.T) {
	if kindError(nil) != nil {
		t.Fatal("kindError(nil) is not nil")
	}
	if err := kindError(context.Canceled); !errors.Is(err, context.Canceled) || err != context.Canceled {
		t.Fatalf("kindError(context.Canceled) = %v, want the cancellation", err)
	}
	once := kindError(domain.ErrKeyRejected)
	if !errors.Is(once, domain.ErrKeyRejected) {
		t.Fatalf("kindError does not wrap the sentinel: %v", once)
	}
	if twice := kindError(once); twice.Error() != once.Error() {
		t.Fatalf("kindError wrapped twice: %q against %q", twice, once)
	}
}
