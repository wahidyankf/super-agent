package outline

import (
	"errors"
	"strconv"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

const uuidShare = "88888888-8888-4888-8888-888888888888"

// TestUserProjectionNeverKeepsTheEmail checks that a user keeps no e-mail address, even when the reply sends one.
func TestUserProjectionNeverKeepsTheEmail(t *testing.T) {
	rec, err := projectObjectOf(userSpec)(decodeJSON(t,
		`{"data":{"id":"33333333-3333-4333-8333-333333333333","name":"Ann","email":"ann@example.test","role":"admin"}}`))
	if err != nil {
		t.Fatalf("projectObjectOf(userSpec): %v", err)
	}
	if _, has := rec.Get("email"); has {
		t.Fatal("user record keeps an e-mail address")
	}
	if got := rec.Str("name"); got != "Ann" {
		t.Fatalf("name = %q, want Ann", got)
	}
}

// TestShareProjectionNeverKeepsTheLink checks that a share keeps no public link, even when the reply sends one.
func TestShareProjectionNeverKeepsTheLink(t *testing.T) {
	rec, err := projectShareInfo(decodeJSON(t, `{"data":{"shares":[{"id":"`+uuidShare+
		`","url":"https://wiki.example.test/s/abc","urlId":"abc","published":true,"views":3}]}}`))
	if err != nil {
		t.Fatalf("projectShareInfo: %v", err)
	}
	for _, key := range []string{"url", "urlId"} {
		if _, has := rec.Get(key); has {
			t.Fatalf("share record keeps %q", key)
		}
	}
	if v, _ := rec.Get("views"); v != 3 {
		t.Fatalf("views = %v, want 3", v)
	}
}

// TestShareInfoWithNoShareIsNotFound checks that an empty share list is ErrNotFound, not an empty record.
func TestShareInfoWithNoShareIsNotFound(t *testing.T) {
	_, err := projectShareInfo(decodeJSON(t, `{"data":{"shares":[]}}`))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

// TestShareInfoWithoutAListIsUnexpected checks that a reply without the shares list is an unexpected reply.
func TestShareInfoWithoutAListIsUnexpected(t *testing.T) {
	_, err := projectShareInfo(decodeJSON(t, `{"data":{"share":{"id":"`+uuidShare+`"}}}`))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
	}
}

// TestChildrenProjectionFlattensTheNavNode checks that documents.children reads the one NavNode of the reply: each child
// names the document as its parent, and the depth of a child counts from 0.
func TestChildrenProjectionFlattensTheNavNode(t *testing.T) {
	rec, err := projectChildren(decodeJSON(t, `{"data":{"id":"doc-1","title":"Root","url":"/doc/root","children":[`+
		`{"id":"doc-2","title":"A","url":"/doc/a","children":[{"id":"doc-3","title":"B","url":"/doc/b","children":[]}]},`+
		`{"id":"doc-4","title":"C","url":"/doc/c"}]}}`))
	if err != nil {
		t.Fatalf("projectChildren: %v", err)
	}
	items := rec.Items()
	if len(items) != 3 {
		t.Fatalf("children = %d, want 3", len(items))
	}
	wantDepth := []int{0, 1, 0}
	wantIDs := []string{"doc-2", "doc-3", "doc-4"}
	for i, item := range items {
		if got := item.Str("id"); got != wantIDs[i] {
			t.Fatalf("item %d id = %q, want %q", i, got, wantIDs[i])
		}
		if v, _ := item.Get("depth"); v != wantDepth[i] {
			t.Fatalf("item %d depth = %v, want %d", i, v, wantDepth[i])
		}
		if parent, _ := item.Get("parentId"); i == 1 {
			if parent != "doc-2" {
				t.Fatalf("grandchild parent = %v, want doc-2", parent)
			}
		} else if parent != "doc-1" {
			t.Fatalf("item %d parent = %v, want doc-1", i, parent)
		}
	}
}

// TestTreeDepthIsCapped checks that a tree nested deeper than maxTreeDepth is cut at the cap, so no node of the answer
// is deeper than the cap and the indentation of a tree stays bounded (finding a4).
func TestTreeDepthIsCapped(t *testing.T) {
	const levels = 200
	node := `{"id":"doc-leaf","title":"leaf","children":[]}`
	for i := 0; i < levels; i++ {
		node = `{"id":"doc-` + strconv.Itoa(i) + `","title":"t","children":[` + node + `]}`
	}
	rec, err := projectTree(decodeJSON(t, `{"data":[`+node+`]}`))
	if err != nil {
		t.Fatalf("projectTree: %v", err)
	}
	for _, item := range rec.Items() {
		depth, _ := item.Get("depth")
		if d, _ := depth.(int); d > maxTreeDepth {
			t.Fatalf("a node at depth %d is in the answer, want no node deeper than %d", d, maxTreeDepth)
		}
	}
	if got := len(rec.Items()); got != maxTreeDepth+1 {
		t.Fatalf("nodes = %d, want %d (depths 0 to %d)", got, maxTreeDepth+1, maxTreeDepth)
	}
}

// TestChildrenOfADocumentWithNoCollectionIsAnEmptyList checks the reply of documents.children for a document that is in
// no collection. The router answers data: undefined, which JSON sends as no data member at all, and that is no children,
// not an unexpected reply. A null data is the same.
func TestChildrenOfADocumentWithNoCollectionIsAnEmptyList(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":null}`, `{"ok":true,"status":200}`} {
		rec, err := projectChildren(decodeJSON(t, body))
		if err != nil {
			t.Fatalf("projectChildren(%s) error = %v, want an empty list", body, err)
		}
		if !rec.IsList() || len(rec.Items()) != 0 {
			t.Fatalf("projectChildren(%s) = %+v, want an empty list", body, rec)
		}
	}
}

// TestChildrenProjectionAcceptsANodeWithNoChildren checks that a document with no children is an empty list.
func TestChildrenProjectionAcceptsANodeWithNoChildren(t *testing.T) {
	rec, err := projectChildren(decodeJSON(t, `{"data":{"id":"doc-1","title":"Leaf","children":null}}`))
	if err != nil {
		t.Fatalf("projectChildren: %v", err)
	}
	if !rec.IsList() || len(rec.Items()) != 0 {
		t.Fatalf("record = %+v, want an empty list", rec)
	}
}

// TestNestedListMissingIsUnexpected checks that a membership reply without its list is an unexpected reply, and that
// a null list is one too.
func TestNestedListMissingIsUnexpected(t *testing.T) {
	for _, body := range []string{`{"data":{"users":[]}}`, `{"data":{"memberships":null}}`} {
		_, err := projectNestedOf("memberships", membershipSpec)(decodeJSON(t, body))
		if !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("%s: error = %v, want ErrUnexpectedResponse", body, err)
		}
	}
}

// TestNestedListIgnoresExtraMembers checks that the documents and users next to a membership list are not kept.
func TestNestedListIgnoresExtraMembers(t *testing.T) {
	rec, err := projectNestedOf("memberships", membershipSpec)(decodeJSON(t,
		`{"data":{"memberships":[{"id":"55555555-5555-4555-8555-555555555555","userId":"u-1","permission":"read",`+
			`"extra":"ignored"}],"users":[{"id":"u-1","email":"x@example.test"}]}}`))
	if err != nil {
		t.Fatalf("projectNestedOf: %v", err)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("items = %d, want 1", len(rec.Items()))
	}
	if _, has := rec.Items()[0].Get("extra"); has {
		t.Fatal("an extra field reached the record")
	}
}

// TestCommentProjectionKeepsThePlainTextOnly checks that a comment's editor data is never kept and its text is the
// words of the data.
func TestCommentProjectionKeepsThePlainTextOnly(t *testing.T) {
	rec, err := projectObjectOf(commentSpec)(decodeJSON(t,
		`{"data":{"id":"44444444-4444-4444-8444-444444444444","documentId":"doc-1","createdById":"u-1",`+
			`"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z",`+
			`"data":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]},`+
			`{"type":"paragraph","content":[{"type":"text","text":"World"}]}]}}}`))
	if err != nil {
		t.Fatalf("projectObjectOf(commentSpec): %v", err)
	}
	if _, has := rec.Get("data"); has {
		t.Fatal("comment record keeps the editor data")
	}
	if got := rec.Str("text"); got != "Hello\nWorld" {
		t.Fatalf("text = %q, want %q", got, "Hello\nWorld")
	}
}

// TestPlainTextReadsEditorNodes checks the plain-text extraction of the editor JSON that a comment keeps: the text of
// each block on its own line, a hard break as a line break, and marks dropped.
func TestPlainTextReadsEditorNodes(t *testing.T) {
	cases := []struct {
		name string
		node string
		want string
	}{
		{"one paragraph", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi"}]}]}`, "Hi"},
		{"a bold word", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a ",` +
			`"marks":[]},{"type":"text","text":"bold","marks":[{"type":"strong"}]}]}]}`, "a bold"},
		{"a hard break", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a"},` +
			`{"type":"hard_break"},{"type":"text","text":"b"}]}]}`, "a\nb"},
		{"a list of items", `{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":` +
			`[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},{"type":"listItem","content":` +
			`[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]}]}`, "one\ntwo"},
		{"an empty paragraph", `{"type":"doc","content":[{"type":"paragraph"}]}`, ""},
		{"not an editor node", `"just text"`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := plainText(decodeJSON(t, tc.node)); got != tc.want {
				t.Fatalf("plainText = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCommentWithoutEditorDataKeepsTheServerText checks that a reply with no editor data keeps the text it carries.
func TestCommentWithoutEditorDataKeepsTheServerText(t *testing.T) {
	rec, err := projectObjectOf(commentSpec)(decodeJSON(t,
		`{"data":{"id":"44444444-4444-4444-8444-444444444444","text":"plain words","data":null}}`))
	if err != nil {
		t.Fatalf("projectObjectOf(commentSpec): %v", err)
	}
	if got := rec.Str("text"); got != "plain words" {
		t.Fatalf("text = %q, want %q", got, "plain words")
	}
}

// TestTemplateProjectionNeverKeepsEditorData checks that a template keeps no editor data.
func TestTemplateProjectionNeverKeepsEditorData(t *testing.T) {
	rec, err := projectObjectOf(templateSpec)(decodeJSON(t,
		`{"data":{"id":"tpl-1","title":"Weekly","fullWidth":true,"data":{"type":"doc"}}}`))
	if err != nil {
		t.Fatalf("projectObjectOf(templateSpec): %v", err)
	}
	if _, has := rec.Get("data"); has {
		t.Fatal("template record keeps the editor data")
	}
	if v, _ := rec.Get("fullWidth"); v != true {
		t.Fatalf("fullWidth = %v, want true", v)
	}
}

// TestObjectProjectionRefusesAWrongID checks that a required ID which is not a UUID makes the reply unexpected.
func TestObjectProjectionRefusesAWrongID(t *testing.T) {
	_, err := projectObjectOf(groupSpec)(decodeJSON(t, `{"data":{"id":"not-a-uuid","name":"Eng"}}`))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
	}
}
