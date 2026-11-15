package connectorspec

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// bundledOutlineStub is a byte copy of a bundled Outline stub: a spec of the built-in kind outline whose operation
// claims the catalog row documents.get. The user-folder loader must refuse these bytes.
const bundledOutlineStub = `version: 1
kind: outline
summary: Reads documents of an Outline wiki.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_OUTLINE_API_KEY
operations:
  - id: outline.documents.get
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    catalog_id: documents.get
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

// bundledJiraStub claims three catalog rows of the jira kind. It mirrors the stub of the spec module's bundled
// tests, because only operations whose parameters match their rows load.
const bundledJiraStub = `version: 1
kind: jira
summary: Reads issues of an example Jira source.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_JIRA_API_KEY
operations:
  - id: jira.auth.whoami
    summary: Show the account that signs in.
    access: read
    method: GET
    path: /whoami
    catalog_id: jira.whoami
    response:
      fields:
        name:
          pointer: /displayName
    examples:
      - request: {}
        response:
          status: 200
          body:
            displayName: Ada
        expect:
          name: Ada
  - id: jira.projects.list
    summary: List the projects of the source.
    access: read
    method: GET
    path: /projects
    catalog_id: jira.projects.list
    params:
      - name: query
        type: Query
        in: query
        required: false
      - name: limit
        type: Int
        in: query
        required: false
        default: 25
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
    response:
      fields:
        key:
          pointer: /key
    examples:
      - request:
          query: example
        response:
          status: 200
          body:
            key: EX
        expect:
          key: EX
  - id: jira.issues.search
    summary: Search the issues that match a query.
    access: read
    method: GET
    path: /search
    catalog_id: jira.issues.search
    params:
      - name: jql
        type: Query
        in: query
        required: true
      - name: limit
        type: Limit
        in: query
        required: false
        default: 50
    response:
      fields:
        key:
          pointer: /key
    examples:
      - request:
          jql: project = EX
        response:
          status: 200
          body:
            key: EX-1
        expect:
          key: EX-1
`

// bundledExampleStub is an embedded stub of a kind that is not built in. Its only operation is hidden from both
// surfaces, so the stub loads with no catalog row claimed.
const bundledExampleStub = `version: 1
kind: example-bundled
summary: Reads things of a bundled example.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_BUNDLED_API_KEY
operations:
  - id: example-bundled.things.get
    summary: Read one thing.
    access: read
    method: GET
    path: /things/{id}
    cli: false
    mcp: false
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
          id: thing-1
        response:
          status: 200
          body:
            title: First thing
        expect:
          title: First thing
`

// userExampleDoc is a user-folder spec of the example-bundled kind. It loads on its own, and it shadows the
// embedded stub of the same kind once the two sets merge.
const userExampleDoc = `version: 1
kind: example-bundled
summary: Reads things of a user example.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_BUNDLED_API_KEY
operations:
  - id: example-bundled.things.list
    summary: List the things.
    access: read
    method: GET
    path: /things
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: thing-1
        expect:
          id: thing-1
`

// TestUserSpecCannotClaimBuiltinKind proves the first half of L9: a byte copy of a bundled stub in the user folder
// is refused, naming the file and the kind.
func TestUserSpecCannotClaimBuiltinKind(t *testing.T) {
	mustFail(t, "connector spec file outline.yml: /kind: is the kind of a built-in source",
		specFile("outline.yml", bundledOutlineStub))
}

// TestCatalogIDIsRefusedInUserSpec proves the second half of L9: a user spec that names a catalog_id is refused at
// the field, because only the loader of the embedded specs may take the exception.
func TestCatalogIDIsRefusedInUserSpec(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "    path: /docs/{id}\n",
		"    path: /docs/{id}\n    catalog_id: example-wiki.docs.get\n", 1)
	mustFail(t, "connector spec file bad.yml: /operations/0/catalog_id: is a field that only a bundled spec may name",
		specFile("bad.yml", data))
}

// TestUserFolderCannotShadowBundledSpec proves the shadow rule: a user spec whose kind the embedded set claims
// stops start-up, even when the user spec is valid on its own.
func TestUserFolderCannotShadowBundledSpec(t *testing.T) {
	bundled := mustBundled(t, specFile("things.yml", bundledExampleStub))
	user := mustLoad(t, specFile("custom.yml", userExampleDoc))
	_, err := Merge(user, bundled)
	if err == nil {
		t.Fatal("Merge accepts a user spec that shadows a bundled spec")
	}
	if got, want := err.Error(), "connector spec file custom.yml: /kind: shadows a bundled spec"; got != want {
		t.Fatalf("Merge() error = %q, want %q", got, want)
	}
}

// TestBundledSpecCoversEveryCatalogRow proves L10: the embed loader refuses a bundled set that leaves a catalog row
// without an operation, except for the documented unclaimed change rows, and it refuses an operation whose catalog_id
// names no row. A set that covers every row of the Outline kind is the bundled outline.yml, which
// TestLoadBundledLoadsTheOutlineSpec checks.
func TestBundledSpecCoversEveryCatalogRow(t *testing.T) {
	_, err := loadBundled([]specfolder.File{specFile("jira.yml", bundledJiraStub)})
	if err == nil {
		t.Fatal("loadBundled accepts a bundled set that leaves catalog rows without an operation")
	}
	if got, want := err.Error(), "connector spec file jira.yml: (document): leaves a row of the kind without an operation"; got != want {
		t.Fatalf("loadBundled() error = %q, want %q", got, want)
	}
	bogus := strings.Replace(bundledJiraStub, "    catalog_id: jira.issues.search",
		"    catalog_id: jira.no-such-operation", 1)
	_, err = loadBundled([]specfolder.File{specFile("jira.yml", bogus)})
	if err == nil {
		t.Fatal("loadBundled accepts a bundled operation that names no row")
	}
	if got, want := err.Error(), "connector spec file jira.yml: /operations/2/catalog_id: names no row of the kind"; got != want {
		t.Fatalf("loadBundled() error = %q, want %q", got, want)
	}
}

// TestTwoBundledSpecsOfOneKindCollide proves the limit that one row is claimed once at the kind level: one file
// holds each bundled kind, so a second spec of the kind, which would claim its rows again, is refused.
func TestTwoBundledSpecsOfOneKindCollide(t *testing.T) {
	_, err := loadBundled([]specfolder.File{
		specFile("a.yml", bundledJiraStub),
		specFile("b.yml", bundledJiraStub),
	})
	if err == nil {
		t.Fatal("loadBundled accepts two specs of one kind")
	}
	if got, want := err.Error(), "connector spec file b.yml: /kind: collides with another spec's kind"; got != want {
		t.Fatalf("loadBundled() error = %q, want %q", got, want)
	}
}

// TestMergeKeepsTheUserSetWhenNothingIsShadowed proves the quiet path of the merge: with no bundled spec of the
// user's kind, the merged set holds the user's specs, digests, and rows, plus the bundled digests.
func TestMergeKeepsTheUserSetWhenNothingIsShadowed(t *testing.T) {
	user := mustLoad(t, specFile("example-wiki.yml", wikiHeader+docsGet))
	bundled := mustBundled(t)
	merged, err := Merge(user, bundled)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(merged.Specs) != 1+len(bundled.Specs) || merged.Specs[0].Kind.String() != "example-wiki" {
		t.Fatalf("specs = %+v, want the user spec first", merged.Specs)
	}
	if got := len(merged.Operations.Rows()); got != len(operations.Rows())+1 {
		t.Fatalf("catalog rows = %d, want the built-in rows plus one", got)
	}
	if _, ok := merged.Digests["example-wiki"]; !ok {
		t.Fatalf("digests = %v, want the user digest", merged.Digests)
	}
	for _, sp := range bundled.Specs {
		if _, ok := merged.Digests[sp.Model.Kind]; !ok {
			t.Fatalf("digests = %v, want the digest of %s", merged.Digests, sp.File)
		}
	}
}

// mustBundled loads the bundled files and requires the load to succeed.
func mustBundled(t *testing.T, files ...specfolder.File) Set {
	t.Helper()
	set, err := loadBundled(files)
	if err != nil {
		t.Fatalf("loadBundled() error = %v, want nil", err)
	}
	return set
}

// TestLoadBundledFromReadsSpecFilesSkippingOtherEntries proves the file-system layer: a .yml and a .yaml file load
// with the digests of their bytes, and any other entry is skipped.
func TestLoadBundledFromReadsSpecFilesSkippingOtherEntries(t *testing.T) {
	other := strings.ReplaceAll(bundledExampleStub, "example-bundled", "example-second")
	fsys := fstest.MapFS{
		"bundled/a.yml":   &fstest.MapFile{Data: []byte(bundledExampleStub)},
		"bundled/b.yaml":  &fstest.MapFile{Data: []byte(other)},
		"bundled/README":  &fstest.MapFile{Data: []byte("a note")},
		"elsewhere/c.yml": &fstest.MapFile{Data: []byte(bundledExampleStub)},
	}
	set, err := loadBundledFrom(fsys, "bundled")
	if err != nil {
		t.Fatalf("loadBundledFrom: %v", err)
	}
	if len(set.Specs) != 2 || set.Specs[0].File != "a.yml" || set.Specs[1].File != "b.yaml" {
		t.Fatalf("specs = %+v, want a.yml and b.yaml in order", set.Specs)
	}
	for _, file := range []specfolder.File{specFile("a.yml", bundledExampleStub), specFile("b.yaml", other)} {
		var kind string
		for _, sp := range set.Specs {
			if sp.File == file.Name {
				kind = sp.Model.Kind
			}
		}
		if kind == "" {
			t.Fatalf("no spec named %s", file.Name)
		}
		if set.Digests[kind] != file.Digest {
			t.Fatalf("digest of %s = %q, want %q", file.Name, set.Digests[kind], file.Digest)
		}
	}
}

// failFS is a file system whose listing always fails, so the embed loader reports the folder it cannot read.
type failFS struct{}

// Open fails with a refusal for every name.
func (failFS) Open(string) (fs.File, error) {
	return nil, fs.ErrPermission
}

// TestLoadBundledFromRefusesAFolderItCannotRead proves the failure of the file-system layer.
func TestLoadBundledFromRefusesAFolderItCannotRead(t *testing.T) {
	_, err := loadBundledFrom(failFS{}, "bundled")
	if err == nil {
		t.Fatal("loadBundledFrom accepts a folder it cannot read")
	}
	if got, want := err.Error(), "connector spec file bundled: (document): could not be read"; got != want {
		t.Fatalf("loadBundledFrom() error = %q, want %q", got, want)
	}
}

// TestLoadBundledLoadsTheOutlineSpec proves the ports of pull requests 16 and 17: the embedded tree loads the
// bundled outline.yml, which claims every catalog row of the Outline kind and the fallback target documents.export,
// except the documented unclaimed change rows. The set holds the built-in operation rows only.
func TestLoadBundledLoadsTheOutlineSpec(t *testing.T) {
	set, err := LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v, want nil", err)
	}
	model := bundledModelOf(t, set, "outline.yml", domain.KindOutline)
	if _, ok := set.Digests["outline"]; !ok {
		t.Fatalf("digests = %v, want the outline digest", set.Digests)
	}
	if got := len(set.Operations.Rows()); got != len(operations.Rows()) {
		t.Fatalf("catalog rows = %d, want the built-in rows only", got)
	}
	claimed := map[string]bool{}
	for _, op := range model.Operations {
		if op.CatalogID != "" {
			claimed[op.CatalogID] = true
		}
	}
	var missing []string
	for _, row := range operations.RowsOf(domain.KindOutline) {
		if !claimed[string(row.ID)] && !unclaimedChangeRows[string(row.ID)] {
			missing = append(missing, string(row.ID))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("unclaimed outline rows = %v", missing)
	}
	for row := range unclaimedChangeRows {
		if claimed[row] {
			t.Fatalf("row %s is claimed but listed as unclaimed", row)
		}
		looked, ok := operations.Lookup(domain.OperationID(row))
		if !ok || looked.Effect == domain.EffectRead {
			t.Fatalf("unclaimed row %s is not an Outline change row", row)
		}
	}
	wantUnclaimed := 0
	if got := len(unclaimedChangeRows); got != wantUnclaimed {
		t.Fatalf("unclaimed change rows = %d, want %d", got, wantUnclaimed)
	}
}

// TestBundledSpecCoversEveryCatalogRowForJira proves the gate of pull request 37: the embedded tree loads the
// bundled jira.yml, which claims every read row of the Jira kind, with the catalog names the probe found covered.
// The Jira kind holds no write row, so every row it claims is a read.
func TestBundledSpecCoversEveryCatalogRowForJira(t *testing.T) {
	set, err := LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v, want nil", err)
	}
	model := bundledModelOf(t, set, "jira.yml", domain.KindJira)
	if _, ok := set.Digests["jira"]; !ok {
		t.Fatalf("digests = %v, want the jira digest", set.Digests)
	}
	claimed := map[string]bool{}
	for _, op := range model.Operations {
		if op.CatalogID != "" {
			claimed[op.CatalogID] = true
		}
	}
	var missing []string
	for _, row := range operations.RowsOf(domain.KindJira) {
		if !claimed[string(row.ID)] {
			missing = append(missing, string(row.ID))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("unclaimed jira rows = %v", missing)
	}
}

// bundledModelOf returns the model of the bundled file of the kind, requiring the set to hold exactly the two
// bundled specs: the jira.yml before the outline.yml, in embedded name order.
func bundledModelOf(t *testing.T, set Set, file string, kind domain.SourceKind) *spec.Spec {
	t.Helper()
	if len(set.Specs) != 2 || set.Specs[0].File != "jira.yml" || set.Specs[1].File != "outline.yml" {
		t.Fatalf("specs = %+v, want jira.yml then outline.yml", set.Specs)
	}
	for _, sp := range set.Specs {
		if sp.File == file {
			if sp.Model.Kind != kind.String() {
				t.Fatalf("kind of %s = %q, want %s", file, sp.Model.Kind, kind.String())
			}
			return sp.Model
		}
	}
	t.Fatalf("no bundled spec named %s", file)
	return nil
}

// TestMustKindPanicsOnAnUnregisteredKind checks the broken invariant: a name
// the set never registered panics instead of building half a registration.
func TestMustKindPanicsOnAnUnregisteredKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustKind() of an unregistered kind does not panic")
		}
	}()
	mustKind(kindSet(nil), "no-such-kind")
}
