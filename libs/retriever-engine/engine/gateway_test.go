package engine_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// gatewayOps are the two role operations that the fixture does not carry: the flat collection tree and the one document
// read. They are appended to the fixture's operation list, which is the last part of the document.
const gatewayOps = `  - id: example-wiki.collections.tree
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
              - id: doc-2
                title: Second document
                url: https://docs.example.test/doc-2
                depth: 1
                parent_id: doc-1
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

// roundTripFunc is a transport that a test writes as a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// brokenBody is a reply body that fails while it is read, as a connection that drops does.
type brokenBody struct{}

// Read always fails.
func (brokenBody) Read([]byte) (int, error) {
	return 0, errors.New("the connection dropped")
}

// failingCredentials is a credentials port whose key is missing.
type failingCredentials struct{}

// APIKey always fails, and names no value.
func (failingCredentials) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.ApiKey{}, errors.New("the key variable is not set in this test")
}

// failingIdentities is an identity port whose address is missing.
type failingIdentities struct{}

// Identity always fails.
func (failingIdentities) Identity(domain.IdentityRef) (domain.Identity, error) {
	return domain.Identity{}, errors.New("the identity variable is not set in this test")
}

// TestGatewayReadsRunTheirRoleOperations checks the collection tree and the document read: each sends its declared
// request, and maps the flat tree with its parent IDs and its depths.
func TestGatewayReadsRunTheirRoleOperations(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, wikiYAML+gatewayOps))
	col, err := domain.NewCollectionID("col-1")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := eng.CollectionTree(context.Background(), col)
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 2 || nodes[0].ParentID != nil || nodes[0].Depth != 0 {
		t.Fatalf("the root node = %+v, want no parent at depth 0", nodes)
	}
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "doc-1" || nodes[1].Depth != 1 || nodes[1].URL == "" {
		t.Fatalf("the child node = %+v, want parent doc-1 at depth 1 with a URL", nodes[1])
	}
	doc, err := domain.NewDocumentID("doc-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := eng.Document(context.Background(), doc)
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if got.Markdown != "# Hello" || got.CollectionID != "col-1" || got.Title != "First document" {
		t.Fatalf("the document = %+v", got)
	}
	paths := []string{}
	for _, r := range stub.Requests() {
		paths = append(paths, r.Path)
	}
	if len(paths) != 2 || paths[0] != "/v1/collections/col-1/docs" || paths[1] != "/v1/documents/doc-1" {
		t.Fatalf("paths = %v, want the two role paths", paths)
	}
}

// TestListCollectionsWithoutTheRoleIsUnsupported checks a spec that gives no operation the list-collections role.
func TestListCollectionsWithoutTheRoleIsUnsupported(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, replaceIn(t, wikiYAML, "    role: list-collections\n", "")))
	if _, err := eng.ListCollections(context.Background()); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an unsupported role sent %d requests", n)
	}
}

// TestListCollectionsPropagatesAFailedRead checks that a failed reply is returned, not mapped.
func TestListCollectionsPropagatesAFailedRead(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/collections", replay.Reply{Status: 500})
	if _, err := eng.ListCollections(context.Background()); !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("err = %v, want ErrUpstreamFailure", err)
	}
}

// TestListCollectionsRefusesAListThatIsNotAList checks a reply whose items are not a list.
func TestListCollectionsRefusesAListThatIsNotAList(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/collections", page(`{"items":"col-1"}`))
	if _, err := eng.ListCollections(context.Background()); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestObjectReadRefusesAReplyThatIsNotAnObject checks an object read whose reply is a list.
func TestObjectReadRefusesAReplyThatIsNotAnObject(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/search?note=first", page(`[1]`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(map[string]any{"note": "first"})); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestOffsetReadStopsAtAFailedPage checks that a failed page ends the read with its error.
func TestOffsetReadStopsAtAFailedPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=0", replay.Reply{Status: 503})
	if _, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2})); !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("err = %v, want ErrUpstreamFailure", err)
	}
}

// TestOffsetReadRefusesAPageThatIsNotAList checks that a page whose items are not a list fails the read.
func TestOffsetReadRefusesAPageThatIsNotAList(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=0", page(`{"items":"doc-1"}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2})); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestOffsetReadNeedsAPageSize checks an offset read whose page size has no value: the read refuses it before any request.
func TestOffsetReadNeedsAPageSize(t *testing.T) {
	doc := replaceIn(t, wikiYAML, "        default: 2\n      - name: offset", "      - name: offset")
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(nil)); !errors.Is(err, domain.ErrInvalidLimit) {
		t.Fatalf("err = %v, want ErrInvalidLimit", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a read with no page size sent %d requests", n)
	}
}

// TestCursorReadStopsAtAFailedPage checks a failed first page of a cursor read.
func TestCursorReadStopsAtAFailedPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", replay.Reply{Status: 500})
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("err = %v, want ErrUpstreamFailure", err)
	}
}

// TestCursorReadRefusesAPageThatIsNotAList checks a cursor page whose items are not a list.
func TestCursorReadRefusesAPageThatIsNotAList(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":"doc-1","next_cursor":null}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestCursorReadFollowsANumericCursor checks a cursor that the source sends as a whole number.
func TestCursorReadFollowsANumericCursor(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":7}`))
	stub.Script("GET", "/v1/docs?cursor=7&limit=20", page(`{"items":[{"id":"doc-2"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 || stub.Requests()[1].Query != "cursor=7&limit=20" {
		t.Fatalf("items = %d, second query = %q", len(rec.Items()), stub.Requests()[1].Query)
	}
}

// TestCursorReadRefusesAnObjectAsTheCursor checks a next cursor that is neither text nor a number.
func TestCursorReadRefusesAnObjectAsTheCursor(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":{"a":1}}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestAFailedKeyStopsTheRequestBeforeItIsSent checks that a credentials port that fails sends nothing.
func TestAFailedKeyStopsTheRequestBeforeItIsSent(t *testing.T) {
	stub, err := replay.New(loadSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: failingCredentials{}, Clock: &fakeClock{}, Transport: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err == nil {
		t.Fatal("a read with no key succeeded")
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a read with no key sent %d requests", n)
	}
}

// TestAFailedIdentityStopsBasicAuthBeforeItIsSent checks that a basic-auth read with no identity sends nothing.
func TestAFailedIdentityStopsBasicAuthBeforeItIsSent(t *testing.T) {
	s := basicSpec(t)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := engine.New(s, basicSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Identities: failingIdentities{}, Clock: &fakeClock{}, Transport: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err == nil {
		t.Fatal("a basic read with no identity succeeded")
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a basic read with no identity sent %d requests", n)
	}
}

// TestAReplyBodyThatFailsIsUnreachable checks that a connection that drops while the reply is read is reported as
// unreachable, with no key in the text.
func TestAReplyBodyThatFailsIsUnreachable(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(brokenBody{})}, nil
	})
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Clock: &fakeClock{}, Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	assertNoKey(t, err)
}

// TestAllowPrivateNetworkIsOnlyAKeyOfTheCatalog checks the two network settings of a real dial. Without the catalog
// setting a loopback address is refused and never dialed. With it, the checked loopback address is dialed, and the
// fake dialer refuses the connection, so no socket is opened.
func TestAllowPrivateNetworkIsOnlyAKeyOfTheCatalog(t *testing.T) {
	loopback := netip.MustParseAddr("127.0.0.1")
	build := func(opts ...engine.Option) (*engine.Engine, *[]string) {
		var dialed []string
		eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
			Credentials: &fakeCredentials{},
			Clock:       &fakeClock{},
			Network: engine.Network{
				Lookup: func(context.Context, string) ([]netip.Addr, error) { return []netip.Addr{loopback}, nil },
				Dial: func(_ context.Context, _, address string) (net.Conn, error) {
					dialed = append(dialed, address)
					return nil, errors.New("the fake dialer refuses the connection")
				},
			},
		}, opts...)
		if err != nil {
			t.Fatal(err)
		}
		return eng, &dialed
	}

	refused, dialed := build()
	if _, err := refused.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("without the setting: err = %v, want ErrNetworkRefused", err)
	}
	if len(*dialed) != 0 {
		t.Fatalf("without the setting the loopback address was dialed: %v", *dialed)
	}

	allowed, dialed := build(engine.AllowPrivateNetwork())
	if _, err := allowed.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("with the setting: err = %v, want ErrUnreachable from the fake dialer", err)
	}
	if len(*dialed) != 1 || (*dialed)[0] != "127.0.0.1:443" {
		t.Fatalf("with the setting dialed %v, want only 127.0.0.1:443", *dialed)
	}
}

// TestADialDeadlineIsReportedAsTimedOut checks that a dial which reports the deadline is a timeout, with the exit status
// of a timeout. The fake dialer returns the deadline at once, so no socket waits.
func TestADialDeadlineIsReportedAsTimedOut(t *testing.T) {
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{},
		Clock:       &fakeClock{},
		Network: engine.Network{
			Lookup: func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("198.51.100.7")}, nil
			},
			Dial: func(context.Context, string, string) (net.Conn, error) {
				return nil, context.DeadlineExceeded
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrTimedOut) || engine.ExitStatus(err) != engine.ExitTimedOut {
		t.Fatalf("err = %v, want ErrTimedOut and exit 124", err)
	}
}
