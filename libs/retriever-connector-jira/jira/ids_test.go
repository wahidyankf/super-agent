package jira

import (
	"errors"
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
	bad := []string{"", "A", "example", "EXA EMP", "EXA-EMP", "1ABC", "ABCDEFGHIJKLMNOPQRSTU", "EXA/EMP", "EXA\n"}
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
		"EXAMPLE-1\n",
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
