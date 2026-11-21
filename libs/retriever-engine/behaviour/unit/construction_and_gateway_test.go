package unit_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireWikiDoc is the example wiki spec of the engine's own fixture, copied here so that the bindings read no file. It
// names example hosts and example variables only.
const wireWikiDoc = `version: 1
kind: example-wiki
summary: Reads and writes documents of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 5
operations:
  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: cursor
        type: ID
        in: query
        required: false
        help: The cursor of the next page.
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    headers:
      X-Example-Client: retriever
    pagination:
      style: cursor
      max_pages: 5
      cursor_param: cursor
      next_pointer: /next_cursor
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
            next_cursor: null
        expect:
          id: doc-1
          title: First document
  - id: example-wiki.docs.page
    summary: List one page of documents by offset.
    access: read
    method: GET
    path: /docs/page
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
    pagination:
      style: offset
      max_pages: 5
      limit_param: limit
      offset_param: offset
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            items:
              - id: doc-1
              - id: doc-2
        expect:
          id: doc-1
  - id: example-wiki.docs.search
    summary: Search documents, with a note sent in a header.
    access: read
    method: GET
    path: /docs/search
    params:
      - name: note
        type: Text
        in: query
        required: false
        help: A note that is also sent in a header.
    headers:
      X-Example-Note: "{note}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          note: first
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
  - id: example-wiki.docs.get
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
        body:
          pointer: /body
          convert: markdown-from-html
      markdown: body
      errors:
        "404": not_found
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body: <p>Hello</p>
        expect:
          title: First document
          body: <p>Hello</p>
  - id: example-wiki.docs.create
    summary: Create a document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Name
        in: body
        required: true
      - name: text
        type: Text
        in: body
        required: false
    body:
      title: "{{title}}"
      text: "{{text}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First document
          text: Hello
        response:
          status: 201
          body:
            id: doc-2
        expect:
          id: doc-2
  - id: example-wiki.collections.list
    summary: List the collections of the wiki.
    access: read
    method: GET
    path: /collections
    role: list-collections
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
            items:
              - id: col-1
                name: Handbook
        expect:
          id: col-1
          name: Handbook
`

// wireGatewayOps are the collection-tree and get-document operations, appended to wireWikiDoc to give the spec the
// roles that CollectionTree and Document need.
const wireGatewayOps = `  - id: example-wiki.collections.tree
    summary: List the documents of one collection.
    access: read
    method: GET
    path: /collections/{collection_id}/docs
    role: collection-tree
    params:
      - name: collection_id
        type: ID
        in: path
        required: true
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
        url:
          pointer: /url
        depth:
          pointer: /depth
        parent_id:
          pointer: /parent_id
    examples:
      - request:
          collection_id: col-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
                url: https://docs.example.test/doc-1
                depth: 0
                parent_id: null
        expect:
          id: doc-1
  - id: example-wiki.docs.fetch
    summary: Read one document with its Markdown.
    access: read
    method: GET
    path: /documents/{document_id}
    role: get-document
    params:
      - name: document_id
        type: ID
        in: path
        required: true
    response:
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
        url:
          pointer: /url
        markdown:
          pointer: /markdown
        collection_id:
          pointer: /collection_id
    examples:
      - request:
          document_id: doc-1
        response:
          status: 200
          body:
            id: doc-1
            title: First document
            url: https://docs.example.test/doc-1
            markdown: "# Hello"
            collection_id: col-1
        expect:
          id: doc-1
`

// wireReplace returns doc with the first occurrence of from changed to to. It fails the test when from is absent, so a
// document edit can never silently do nothing.
func wireReplace(t *testing.T, doc, from, to string) string {
	t.Helper()
	if !strings.Contains(doc, from) {
		t.Fatalf("the document has no %q to change", from)
	}
	return strings.Replace(doc, from, to, 1)
}

// wireSource returns a source of the named kind on the endpoint, with the example credential name.
func wireSource(t *testing.T, kindName, endpoint string) domain.Source {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: kindName})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := kinds.Kind(kindName)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewSourceID("wiki-main")
	if err != nil {
		t.Fatal(err)
	}
	ep, err := domain.NewEndpoint(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	cred, err := domain.NewCredentialRef(rebindCredential)
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(id, kind, tags, ep, cred)
}

// wirePorts returns the ports of a binding that answers from an in-memory round tripper.
func wirePorts(clock engine.Clock, rt *rebindTransport) engine.Ports {
	return engine.Ports{
		Credentials: &rebindCredentials{},
		Identities:  rebindIdentities{},
		Clock:       clock,
		Transport:   rt,
	}
}

// TestConstructionBuildsFromTheFixtureWithoutEdit builds an engine from the validated example spec with its ports.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: An engine is built from the validated fixture without any edit
func TestConstructionBuildsFromTheFixtureWithoutEdit(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(200, `{"items":[],"next_cursor":null}`))
	eng, err := engine.New(s, rebindSource(t), wirePorts(&rebindClock{}, rt))
	if err != nil {
		t.Fatalf("engine.New returned %v, want no error", err)
	}
	if eng == nil {
		t.Fatal("engine.New returned no engine")
	}
}

// TestConstructionRefusesASpecOfAnotherKind builds an engine for the jira kind from an example-wiki spec.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: An engine is refused for a spec of another kind
func TestConstructionRefusesASpecOfAnotherKind(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	src := wireSource(t, "jira", "https://docs.example.test")
	eng, err := engine.New(s, src, wirePorts(&rebindClock{}, rebindReplies(rebindJSON(200, "{}"))))
	if !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("err = %v, want ErrWrongSourceKind", err)
	}
	if eng != nil {
		t.Fatal("an engine was returned for a spec of another kind")
	}
}

// TestConstructionRefusesAHostOutsideTheSuffix builds an engine for a host that ends in another domain.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: An engine is refused for a host outside the host suffix of the spec
func TestConstructionRefusesAHostOutsideTheSuffix(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	src := wireSource(t, "example-wiki", "https://docs.example.org")
	eng, err := engine.New(s, src, wirePorts(&rebindClock{}, rebindReplies(rebindJSON(200, "{}"))))
	if !errors.Is(err, domain.ErrInvalidEndpoint) {
		t.Fatalf("err = %v, want ErrInvalidEndpoint", err)
	}
	if eng != nil {
		t.Fatal("an engine was returned for a host outside the suffix")
	}
}

// TestConstructionRefusesAMissingPort builds an engine with the credentials port left out.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: An engine is refused when a port it needs is not configured
func TestConstructionRefusesAMissingPort(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	ports := engine.Ports{Identities: rebindIdentities{}, Clock: &rebindClock{}, Transport: rebindReplies(rebindJSON(200, "{}"))}
	eng, err := engine.New(s, rebindSource(t), ports)
	if !errors.Is(err, engine.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if eng != nil {
		t.Fatal("an engine was returned with a port missing")
	}
}

// TestConstructionRefusesBasicAuthWithoutIdentityPort builds an engine for basic authentication with no identity port.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: An engine is refused for basic authentication with no identity port
func TestConstructionRefusesBasicAuthWithoutIdentityPort(t *testing.T) {
	doc := wireReplace(t, wireWikiDoc, "auth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY",
		"auth:\n  type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL")
	s := rebindDoc(t, doc)
	ports := engine.Ports{Credentials: &rebindCredentials{}, Clock: &rebindClock{}, Transport: rebindReplies(rebindJSON(200, "{}"))}
	eng, err := engine.New(s, rebindSource(t), ports)
	if !errors.Is(err, engine.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if eng != nil {
		t.Fatal("an engine was returned for basic authentication with no identity port")
	}
}

// TestConstructionBasicAuthSendsIdentityAndKey runs a read under basic authentication and checks the header.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: Basic authentication sends the identity and the key in the authorization header
func TestConstructionBasicAuthSendsIdentityAndKey(t *testing.T) {
	doc := wireReplace(t, wireWikiDoc, "auth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY",
		"auth:\n  type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL")
	s := rebindDoc(t, doc)
	rt := rebindReplies(rebindJSON(200, `{"items":[],"next_cursor":null}`))
	src := rebindSource(t).WithIdentity(mustIdentityRef(t, "EXAMPLE_WIKI_EMAIL"))
	eng, err := engine.New(s, src, wirePorts(&rebindClock{}, rt))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(rebindIdentity+":"+rebindKey))
	if got := rt.seen()[0].header.Get("Authorization"); got != want {
		t.Fatalf("Authorization = %q, want the basic form of the identity and the key", got)
	}
}

// TestConstructionHeaderAuthSendsKeyInTheDeclaredHeader runs a read under header authentication.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: Header authentication sends the key in the header the spec declares
func TestConstructionHeaderAuthSendsKeyInTheDeclaredHeader(t *testing.T) {
	doc := wireReplace(t, wireWikiDoc, "auth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY",
		"auth:\n  type: header\n  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: X-Example-Key")
	s := rebindDoc(t, doc)
	rt := rebindReplies(rebindJSON(200, `{"items":[],"next_cursor":null}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	req := rt.seen()[0]
	if got := req.header.Get("X-Example-Key"); got != rebindKey {
		t.Fatalf("X-Example-Key = %q, want the key in the declared header", got)
	}
	if got := req.header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none under header authentication", got)
	}
}

// TestConstructionTimeLimitFollowsTheSpec runs a read under two time limits: a spec limit of 10 seconds, and a spec with
// no limit, which takes the default of 30 seconds.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A time limit the spec sets applies to the request, and the default applies when the spec sets none
func TestConstructionTimeLimitFollowsTheSpec(t *testing.T) {
	cases := []struct {
		name      string
		doc       string
		effective int
	}{
		{"10 seconds", wireReplace(t, wireWikiDoc, "timeout_seconds: 20", "timeout_seconds: 10"), 10},
		{"no time limit", wireReplace(t, wireWikiDoc, "limits:\n  timeout_seconds: 20\n", "limits:\n"), 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := rebindDoc(t, c.doc)
			clock := &rebindClock{}
			rt := rebindReplies(rebindJSON(200, `{"items":[],"next_cursor":null}`))
			eng := rebindEngine(t, s, rt, clock)
			if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
				t.Fatalf("Read: %v", err)
			}
			limits := clock.recordedLimits()
			if len(limits) == 0 {
				t.Fatal("the request ran under no time limit")
			}
			for _, d := range limits {
				if int(d.Seconds()) != c.effective {
					t.Fatalf("time limit = %v, want %d seconds", d, c.effective)
				}
			}
		})
	}
}

// TestConstructionRefusesAHandBuiltModelThatBreaksASpecRule edits a validated model by hand, so that it breaks one rule
// of the spec, and checks that the engine refuses it before any request.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A hand-built model that breaks a rule of the spec is refused when the engine is built
func TestConstructionRefusesAHandBuiltModelThatBreaksASpecRule(t *testing.T) {
	faults := []struct {
		name   string
		break_ func(op *spec.Operation)
	}{
		{"a partial path slot", func(op *spec.Operation) { op.Path = "/docs/x{id}" }},
		{"a reserved header name in its header list", func(op *spec.Operation) {
			op.Headers = map[string]string{"Authorization": "Bearer other"}
		}},
		{"a pointer that does not start with a slash", func(op *spec.Operation) { op.Response.Items = "items" }},
	}
	for _, f := range faults {
		t.Run(f.name, func(t *testing.T) {
			base := rebindDoc(t, wireWikiDoc)
			model := *base
			ops := append([]spec.Operation(nil), base.Operations...)
			f.break_(&ops[0])
			model.Operations = ops
			rt := rebindReplies(rebindJSON(200, "{}"))
			eng, err := engine.New(&model, rebindSource(t), wirePorts(&rebindClock{}, rt))
			var inv *spec.InvalidError
			if !errors.As(err, &inv) {
				t.Fatalf("err = %v, want the spec refusal", err)
			}
			if eng != nil {
				t.Fatal("an engine was returned for a model that breaks a spec rule")
			}
			if n := len(rt.seen()); n != 0 {
				t.Fatalf("the refused model sent %d requests, want 0", n)
			}
		})
	}
}

// TestConstructionCollectionsReadNeedsTheListCollectionsRole reads the collections of a spec that gives no operation the
// list-collections role.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A collections read refuses a spec that gives no operation the list-collections role
func TestConstructionCollectionsReadNeedsTheListCollectionsRole(t *testing.T) {
	doc := wireReplace(t, wireWikiDoc, "    role: list-collections\n", "")
	s := rebindDoc(t, doc)
	rt := rebindReplies(rebindJSON(200, `{"items":[]}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the unsupported read sent %d requests, want 0", n)
	}
}

// TestConstructionCollectionTreeHasParentIDsAndDepths reads the flat collection tree of a spec that holds the
// list-collections and collection-tree roles. The read returns each node with its parent ID and its depth. The read is
// the collection-tree read, because only that read returns parent IDs and depths.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A collections read returns the flat tree with its parent IDs and depths
func TestConstructionCollectionTreeHasParentIDsAndDepths(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc+wireGatewayOps)
	rt := rebindReplies(rebindJSON(200, `{"items":[`+
		`{"id":"doc-1","title":"First document","url":"https://docs.example.test/doc-1","depth":0,"parent_id":null},`+
		`{"id":"doc-2","title":"Second document","url":"https://docs.example.test/doc-2","depth":1,"parent_id":"doc-1"}`+
		`]}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	col, err := domain.NewCollectionID("col-1")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := eng.CollectionTree(context.Background(), col)
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].ID != "doc-1" || nodes[0].Depth != 0 || nodes[0].ParentID != nil {
		t.Fatalf("first node = %+v, want doc-1 at depth 0 with no parent", nodes[0])
	}
	if nodes[1].ID != "doc-2" || nodes[1].Depth != 1 || nodes[1].ParentID == nil || *nodes[1].ParentID != "doc-1" {
		t.Fatalf("second node = %+v, want doc-2 at depth 1 under doc-1", nodes[1])
	}
}

// TestConstructionCollectionsReadReturnsTheFailedReplyError reads the collections when the source answers 500.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A collections read fails with the read's error when the source reply fails
func TestConstructionCollectionsReadReturnsTheFailedReplyError(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(500, `{"message":"boom"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	cols, err := eng.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("err = %v, want the upstream failure of the reply", err)
	}
	if cols != nil {
		t.Fatalf("collections = %v, want none", cols)
	}
}

// TestConstructionCollectionsReadRefusesItemsThatAreNotAList replies with an object where the list of items belongs.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A collections read refuses a reply whose items are not a list
func TestConstructionCollectionsReadRefusesItemsThatAreNotAList(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(200, `{"items":{"id":"col-1","name":"Handbook"}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestConstructionUndeclaredOperationIsUnsupported reads an operation id that the spec does not declare.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A read of an operation the spec does not declare is unsupported and sends no request
func TestConstructionUndeclaredOperationIsUnsupported(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(200, "{}"))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.docs.unknown", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the unsupported read sent %d requests, want 0", n)
	}
}

// TestConstructionReadPathRefusesAWriteOperation runs a write operation through the read path.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: The read path refuses a write operation and sends no request
func TestConstructionReadPathRefusesAWriteOperation(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(201, `{"id":"doc-2"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"}))
	if !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("err = %v, want ErrWrongOperationClass", err)
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the refused write sent %d requests, want 0", n)
	}
}

// TestConstructionReadNeedingAnUnservedConverterIsUnsupported reads an operation whose field names the HTML converter,
// which the engine does not serve yet.
//
// @scenario specs/libs/retriever-engine/behaviours/construction-and-gateway.feature: A read that needs a converter the engine does not serve fails before any request
func TestConstructionReadNeedingAnUnservedConverterIsUnsupported(t *testing.T) {
	s := rebindDoc(t, wireWikiDoc)
	rt := rebindReplies(rebindJSON(200, "{}"))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the unsupported read sent %d requests, want 0", n)
	}
}

// mustIdentityRef returns the identity name of a test, and fails the test when the name is refused.
func mustIdentityRef(t *testing.T, name string) domain.IdentityRef {
	t.Helper()
	ref, err := domain.NewIdentityRef(name)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
