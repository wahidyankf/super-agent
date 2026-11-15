package spec

import (
	"strings"
	"testing"
)

// builtIDsDoc is a read of one issue with the four built IDs: the project key, the document ID, the collection ID,
// and the browse URL, each built from the issue key at /key. It names example hosts only.
const builtIDsDoc = `version: 1
kind: example-wiki
summary: Reads one issue with built IDs.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.get
    summary: Read one issue.
    access: read
    method: GET
    path: /issues/{issueKey}
    params:
      - name: issueKey
        type: IssueKey
        in: path
        required: true
    response:
      fields:
        projectKey:
          project_key: /key
        id:
          issue_id: /key
        collectionId:
          collection_id: /key
        url:
          browse_url: /key
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
        expect:
          projectKey: EXAMPLE
          id: issue-EXAMPLE-12
          collectionId: project-EXAMPLE
          url: https://docs.example.test/browse/EXAMPLE-12
`

// TestBuiltIdsValidate checks that a reply field may build its value from an issue key: the project part, the
// document ID, the collection ID, and the browse URL.
func TestBuiltIdsValidate(t *testing.T) {
	if r := Validate([]byte(builtIDsDoc)); !r.Valid() {
		t.Fatalf("a field with a built ID is refused: %v", r.Err())
	}
}

// TestBuiltIdTwoSourcesAreRefused checks that a field reads one key only: a pointer and a built key together are
// refused.
func TestBuiltIdTwoSourcesAreRefused(t *testing.T) {
	doc := mutateAll(builtIDsDoc, "          project_key: /key", "          project_key: /key\n          pointer: /key")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a field with a pointer and a built key is valid")
	}
	if !hasFinding(r, "/operations/0/response/fields/projectKey") {
		t.Fatalf("no finding names the field:\n%s", allText(r))
	}
}

// TestBuiltIdTwoBuiltKeysAreRefused checks that a field builds one ID only: two built keys together are refused.
func TestBuiltIdTwoBuiltKeysAreRefused(t *testing.T) {
	doc := mutateAll(builtIDsDoc, "          project_key: /key", "          project_key: /key\n          issue_id: /key")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a field with two built keys is valid")
	}
	if !hasFinding(r, "/operations/0/response/fields/projectKey") {
		t.Fatalf("no finding names the field:\n%s", allText(r))
	}
}

// TestBuiltIdWithAScalarKeywordIsRefused checks that a built field takes no other scalar keyword: no fallback
// pointer, no format, and no converter.
func TestBuiltIdWithAScalarKeywordIsRefused(t *testing.T) {
	for _, extra := range []string{"          fallback_pointer: /name", "          format: id", "          convert: jira-title"} {
		doc := mutateAll(builtIDsDoc, "          project_key: /key", "          project_key: /key\n"+extra)
		r := Validate([]byte(doc))
		if r.Valid() {
			t.Fatalf("a built field with %q is valid", extra)
		}
		if !hasFinding(r, "/operations/0/response/fields/projectKey") {
			t.Fatalf("no finding names the field for %q:\n%s", extra, allText(r))
		}
	}
}

// mergedBuiltIDsDoc is a merged read with a built ID in one list. It names example hosts only.
const mergedBuiltIDsDoc = `version: 1
kind: example-wiki
summary: Merges issues and documents with a built ID.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.mention
    summary: Mention issues and documents.
    access: read
    method: GET
    path: /mention
    response:
      merge:
        label_field: kind
        lists:
          - items: /issues
            label: issue
            fields:
              projectKey:
                project_key: /key
          - items: /docs
            label: document
            fields:
              id:
                pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
            docs:
              - id: doc-1
        expect:
          kind: issue
          projectKey: EXAMPLE
`

// TestReviewListsBuiltIdsOfMergedLists checks that the review names the built ID of each list of a merge under the
// same operation.
func TestReviewListsBuiltIdsOfMergedLists(t *testing.T) {
	review := Review(requireValid(t, mergedBuiltIDsDoc).Spec)
	want := "1. example-wiki.issues.mention: projectKey as project key of /key"
	if !strings.Contains(review, want) {
		t.Fatalf("the review has no line %q:\n%s", want, review)
	}
}

// TestReviewListsBuiltIds checks the review block of the reads with built IDs: each names the field, the ID it
// builds, and the pointer it builds it from. A spec with no built ID writes no block, so the golden review keeps
// its text.
func TestReviewListsBuiltIds(t *testing.T) {
	review := Review(requireValid(t, builtIDsDoc).Spec)
	for _, line := range []string{
		"Built IDs: 1",
		"1. example-wiki.issues.get: collectionId as collection ID of /key; id as issue ID of /key; projectKey as project key of /key; url as browse URL of /key",
	} {
		if !strings.Contains(review, line) {
			t.Fatalf("the review has no line %q:\n%s", line, review)
		}
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Built IDs") {
		t.Fatal("the review of a spec with no built ID names built IDs")
	}
}
