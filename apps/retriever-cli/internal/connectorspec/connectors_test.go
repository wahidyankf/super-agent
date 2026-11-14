package connectorspec

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// docsGet is one read operation of the example kind, with a path slot and one example.
const docsGet = `  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
        expect:
          title: First document
`

// wikiHeader is the head of a valid spec of kind example-wiki, before its operations.
const wikiHeader = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
`

// specFile returns a spec file named name with data, and the digest its bytes have.
func specFile(name string, data string) specfolder.File {
	sum := sha256.Sum256([]byte(data))
	return specfolder.File{Name: name, Data: []byte(data), Digest: hex.EncodeToString(sum[:])}
}

// mustLoad loads the files and requires the load to succeed.
func mustLoad(t *testing.T, files ...specfolder.File) Set {
	t.Helper()
	set, err := Load(files)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	return set
}

// mustFail loads the files and requires exactly the error text want, which names a file and a path and no value.
func mustFail(t *testing.T, want string, files ...specfolder.File) error {
	t.Helper()
	_, err := Load(files)
	if err == nil {
		t.Fatalf("Load() error = nil, want %q", want)
	}
	if got := err.Error(); got != want {
		t.Fatalf("Load() error = %q, want %q", got, want)
	}
	return err
}

func TestAValidSpecRegistersItsKindAndItsRows(t *testing.T) {
	set := mustLoad(t, specFile("example-wiki.yml", wikiHeader+docsGet))
	if got := strings.Join(set.Kinds.Names(), ","); got != "example-wiki,jira,kiwi,outline" {
		t.Fatalf("kinds = %q, want the built-in kinds and example-wiki", got)
	}
	row, ok := set.Operations.Lookup("example-wiki.docs.get")
	if !ok {
		t.Fatal("the spec operation is not in the catalog")
	}
	kind, ok := set.Operations.KindOf(row)
	if !ok || kind.String() != "example-wiki" {
		t.Fatalf("KindOf() = %v, %v; want example-wiki", kind, ok)
	}
	if row.CLI != "example-wiki docs get" || row.Tool != "example_wiki_docs_get" {
		t.Fatalf("row CLI %q, Tool %q; want the default names", row.CLI, row.Tool)
	}
	if got := len(set.Operations.Rows()); got != len(operations.Rows())+1 {
		t.Fatalf("catalog rows = %d, want the built-in rows plus one", got)
	}
}

func TestAnEmptyFolderRegistersOnlyTheBuiltInKinds(t *testing.T) {
	set := mustLoad(t)
	if got := strings.Join(set.Kinds.Names(), ","); got != "jira,kiwi,outline" {
		t.Fatalf("kinds = %q, want the three built-in kinds", got)
	}
	if got := len(set.Operations.Rows()); got != len(operations.Rows()) {
		t.Fatalf("catalog rows = %d, want the built-in rows only", got)
	}
	if len(set.Specs) != 0 || len(set.Digests) != 0 {
		t.Fatalf("specs = %d, digests = %d; want none", len(set.Specs), len(set.Digests))
	}
}

// TestAnInvalidSpecStopsStartUpNamingItsFileAndPathNeverAValue is the proof of rule (d): the exit is status 2 with the
// stable code, and the text names the file and the path, and it holds no value of the spec.
func TestAnInvalidSpecStopsStartUpNamingItsFileAndPathNeverAValue(t *testing.T) {
	bad := strings.Replace(wikiHeader+docsGet, "summary: Reads documents of an example wiki.", "summary: Reads pasted-secret-91c2 {x}.", 1)
	err := mustFail(t, "connector spec file bad.yml: /summary: "+specRule(t, bad), specFile("bad.yml", bad))
	var specErr *Error
	if !errors.As(err, &specErr) {
		t.Fatalf("Load() error = %T, want *Error", err)
	}
	if got := specErr.ErrorCode(); got != "retriever.connector.spec_invalid" {
		t.Fatalf("ErrorCode() = %q, want retriever.connector.spec_invalid", got)
	}
	if strings.Contains(err.Error(), "pasted-secret-91c2") {
		t.Fatal("the error text holds a value of the spec")
	}
}

// specRule returns the rule the spec module gives the summary of bad, so a test asserts the exact text.
func specRule(t *testing.T, data string) string {
	t.Helper()
	res := spec.Validate([]byte(data))
	if res.Valid() {
		t.Fatal("the spec is valid, want a finding")
	}
	return res.Findings[0].Rule
}

func TestASpecKindThatCollidesWithABuiltInKindIsRefused(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "kind: example-wiki", "kind: outline", 1)
	data = strings.ReplaceAll(data, "example-wiki.", "outline.")
	mustFail(t, "connector spec file bad.yml: /kind: is the kind of a built-in source", specFile("bad.yml", data))
}

func TestASpecKindThatCollidesWithACommandWordIsRefused(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "kind: example-wiki", "kind: serve", 1)
	data = strings.ReplaceAll(data, "example-wiki.", "serve.")
	mustFail(t, "connector spec file bad.yml: /kind: collides with a built-in kind or a command word", specFile("bad.yml", data))
}

func TestTwoSpecsOfOneKindCollide(t *testing.T) {
	mustFail(t, "connector spec file b.yml: /kind: collides with another spec's kind",
		specFile("a.yml", wikiHeader+docsGet), specFile("b.yml", wikiHeader+docsGet))
}

func TestAnOperationWithCLIOffHasNoCommandButKeepsItsTool(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "    path: /docs/{id}\n", "    path: /docs/{id}\n    cli: false\n", 1)
	set := mustLoad(t, specFile("example-wiki.yml", data))
	for _, row := range set.CLIRows {
		if row.ID == "example-wiki.docs.get" {
			t.Fatal("a cli: false operation has a CLI command")
		}
	}
	row, _ := set.Operations.Lookup("example-wiki.docs.get")
	if row.Tool != "example_wiki_docs_get" {
		t.Fatalf("Tool = %q, want the MCP tool name", row.Tool)
	}
}

func TestAnOperationWithMCPOffHasNoToolButKeepsItsCommand(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "    path: /docs/{id}\n", "    path: /docs/{id}\n    mcp: false\n", 1)
	set := mustLoad(t, specFile("example-wiki.yml", data))
	row, _ := set.Operations.Lookup("example-wiki.docs.get")
	if row.Tool != "" {
		t.Fatalf("Tool = %q, want none for mcp: false", row.Tool)
	}
	if !hasRow(set.CLIRows, "example-wiki.docs.get") {
		t.Fatal("the CLI command of an mcp: false operation is missing")
	}
}

func TestCustomSurfaceNamesReplaceTheDefaults(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "    path: /docs/{id}\n", "    path: /docs/{id}\n    cli: fetch\n    mcp: wiki-fetch\n", 1)
	set := mustLoad(t, specFile("example-wiki.yml", data))
	row, _ := set.Operations.Lookup("example-wiki.docs.get")
	if row.CLI != "example-wiki fetch" || row.Tool != "wiki-fetch" {
		t.Fatalf("CLI %q, Tool %q; want the custom names", row.CLI, row.Tool)
	}
}

func TestEffectFollowsTheAccessLabelAndTheMethod(t *testing.T) {
	cases := []struct {
		access, method string
		want           domain.Effect
	}{
		{"read", "GET", domain.EffectRead},
		{"write", "POST", domain.EffectAdditive},
		{"write", "PUT", domain.EffectUpdate},
		{"write", "PATCH", domain.EffectUpdate},
		{"write", "DELETE", domain.EffectDestructive},
		{"admin", "POST", domain.EffectDestructive},
	}
	for _, c := range cases {
		if got := effectOf(spec.Operation{Access: c.access, Method: c.method}); got != c.want {
			t.Fatalf("effectOf(%s %s) = %v, want %v", c.access, c.method, got, c.want)
		}
	}
}

func TestBasicAuthMarksTheKindAsUsingAnIdentity(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY",
		"type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL", 1)
	set := mustLoad(t, specFile("example-wiki.yml", data))
	kind, err := set.Kinds.Kind("example-wiki")
	if err != nil || !kind.UsesIdentity() {
		t.Fatalf("Kind() = %v, %v; want a kind that uses an identity", kind, err)
	}
}

func TestAnOperationIDMustStartWithItsKind(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "id: example-wiki.docs.get", "id: example-other.docs.get", 1)
	mustFail(t, "connector spec file bad.yml: /operations/0/id: does not start with the kind of the spec", specFile("bad.yml", data))
}

func TestTwoOperationsOnOneCommandPathCollide(t *testing.T) {
	first := strings.Replace(docsGet, "    path: /docs/{id}\n", "    path: /docs/{id}\n    cli: fetch\n", 1)
	second := strings.Replace(docsGet, "example-wiki.docs.get", "example-wiki.docs.fetch", 1)
	second = strings.Replace(second, "    path: /docs/{id}\n", "    path: /docs/{id}\n    cli: fetch\n", 1)
	mustFail(t, "connector spec file bad.yml: /operations/1/cli: collides with another command",
		specFile("bad.yml", wikiHeader+first+second))
}

func TestTwoOperationsOnOneToolNameCollide(t *testing.T) {
	first := strings.Replace(docsGet, "    path: /docs/{id}\n", "    path: /docs/{id}\n    mcp: wiki-fetch\n", 1)
	second := strings.Replace(docsGet, "example-wiki.docs.get", "example-wiki.docs.fetch", 1)
	second = strings.Replace(second, "    path: /docs/{id}\n", "    path: /docs/{id}\n    mcp: wiki-fetch\n", 1)
	mustFail(t, "connector spec file bad.yml: /operations/1/mcp: collides with another tool",
		specFile("bad.yml", wikiHeader+first+second))
}

func TestAListOperationMustMapAField(t *testing.T) {
	list := strings.Replace(docsGet, "    response:\n      fields:\n        title:\n          pointer: /title\n",
		"    response:\n      items: /items\n      fields: {}\n", 1)
	list = strings.Replace(list, "        expect:\n          title: First document\n", "        expect: {}\n", 1)
	list = strings.Replace(list, "            title: First document\n", "            items: []\n", 1)
	mustFail(t, "connector spec file bad.yml: /operations/0/response/fields: a list must map at least one field",
		specFile("bad.yml", wikiHeader+list))
}

func TestTheDigestOfEachSpecIsKeptForItsPin(t *testing.T) {
	file := specFile("example-wiki.yml", wikiHeader+docsGet)
	set := mustLoad(t, file)
	if got := set.Digests["example-wiki"]; got != file.Digest {
		t.Fatalf("Digests[example-wiki] = %q, want the file digest", got)
	}
}

// hasRow reports whether a row list holds the operation id.
func hasRow(rows []operations.Row, id string) bool {
	for _, row := range rows {
		if string(row.ID) == id {
			return true
		}
	}
	return false
}
