package spec

import "testing"

// issueKeyDoc is a read with an IssueKey path parameter. It names example hosts only.
const issueKeyDoc = `version: 1
kind: example-wiki
summary: Reads one issue by its key.
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
        key:
          pointer: /key
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
        expect:
          key: EXAMPLE-12
`

// TestIssueKeyParamTypeValidates checks that a spec may name the IssueKey parameter kind on a path parameter. The
// schema gains IssueKey in the parameter type enum and in the path-parameter allow-list, so the document loads, and
// the engine binds the value through the existing KindIssueKey validator of the catalog.
func TestIssueKeyParamTypeValidates(t *testing.T) {
	r := Validate([]byte(issueKeyDoc))
	if !r.Valid() {
		t.Fatalf("a spec with an IssueKey path parameter is refused: %v", r.Err())
	}
}
