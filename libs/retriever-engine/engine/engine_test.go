package engine_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestAListOperationSendsOneDeclaredRequestAndMapsItsFields is the first scenario of pull request 3: a list operation
// against the replay stub sends one request with the declared method, path, and header, and the result holds the
// mapped fields.
func TestAListOperationSendsOneDeclaredRequestAndMapsItsFields(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(map[string]any{"limit": 20}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("the stub received %d requests, want 1", len(reqs))
	}
	r := reqs[0]
	if r.Method != "GET" || r.Path != "/v1/docs" || r.Query != "limit=20" {
		t.Fatalf("request = %s %s?%s, want GET /v1/docs?limit=20", r.Method, r.Path, r.Query)
	}
	if got := r.Header.Get("X-Example-Client"); got != "retriever" {
		t.Fatalf("declared header = %q, want retriever", got)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+fixtureKey {
		t.Fatalf("Authorization header is not the bearer form of the key")
	}
	if !rec.IsList() || len(rec.Items()) != 1 {
		t.Fatalf("the result is not a list of one item")
	}
	if got := rec.Items()[0].Str("id"); got != "doc-1" {
		t.Fatalf("mapped id = %q, want doc-1", got)
	}
	if got := rec.Items()[0].Str("title"); got != "First document" {
		t.Fatalf("mapped title = %q, want First document", got)
	}
}

// TestAnObjectReadMapsEachDeclaredField checks an object read that the replay stub answers with its canned body.
func TestAnObjectReadMapsEachDeclaredField(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(map[string]any{"note": "first"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if rec.IsList() {
		t.Fatal("an object read returned a list")
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want doc-1", got)
	}
	if got := stub.Requests()[0].Header.Get("X-Example-Note"); got != "first" {
		t.Fatalf("the declared note header = %q, want first", got)
	}
}

// TestPagingOffsetCollectsEachPageUntilAShortPage reads three offset pages, the last one short.
func TestPagingOffsetCollectsEachPageUntilAShortPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=0", page(`{"items":[{"id":"doc-1"},{"id":"doc-2"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=2", page(`{"items":[{"id":"doc-3"},{"id":"doc-4"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=4", page(`{"items":[{"id":"doc-5"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 5 {
		t.Fatalf("got %d items, want 5", len(rec.Items()))
	}
	if got := len(stub.Requests()); got != 3 {
		t.Fatalf("got %d requests, want 3", got)
	}
	if p, paged := rec.Page(); !paged || p.HasMore {
		t.Fatalf("a short last page must not continue: %+v paged=%v", p, paged)
	}
}

// TestPagingStopsAtTheSpecPageCap checks that a spec which lowers max_pages gets only that many pages, and that the
// result says more rows exist.
func TestPagingStopsAtTheSpecPageCap(t *testing.T) {
	s := loadSpecFrom(t, lowered(t, "limits:\n  timeout_seconds: 20\n  max_pages: 5", "limits:\n  timeout_seconds: 20\n  max_pages: 2"))
	eng, stub, _ := newStubEngine(t, s)
	stub.Script("GET", "/v1/docs/page?limit=2&offset=0", page(`{"items":[{"id":"doc-1"},{"id":"doc-2"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=2", page(`{"items":[{"id":"doc-3"},{"id":"doc-4"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=4", page(`{"items":[{"id":"doc-5"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len(stub.Requests()); got != 2 {
		t.Fatalf("got %d requests, want 2 at the page cap", got)
	}
	if len(rec.Items()) != 4 {
		t.Fatalf("got %d items, want the first 4", len(rec.Items()))
	}
	p, paged := rec.Page()
	if !paged || !p.HasMore || p.NextOffset != 4 {
		t.Fatalf("the truncated read must say more rows follow at offset 4: %+v paged=%v", p, paged)
	}
}

// TestCursorPagingFollowsTheNextCursorUntilItIsNull reads two cursor pages.
func TestCursorPagingFollowsTheNextCursorUntilItIsNull(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`))
	stub.Script("GET", "/v1/docs?cursor=c2&limit=20", page(`{"items":[{"id":"doc-2"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 || len(stub.Requests()) != 2 {
		t.Fatalf("got %d items over %d requests, want 2 over 2", len(rec.Items()), len(stub.Requests()))
	}
	if got := stub.Requests()[1].Query; got != "cursor=c2&limit=20" {
		t.Fatalf("second page query = %q, want cursor=c2&limit=20", got)
	}
}

// TestCursorPagingStopsAtTheCapWithoutAContinuation reads a cursor list that never ends, so the cap stops it.
func TestCursorPagingStopsAtTheCapWithoutAContinuation(t *testing.T) {
	s := loadSpecFrom(t, lowered(t, "      max_pages: 5\n      cursor_param", "      max_pages: 2\n      cursor_param"))
	eng, stub, _ := newStubEngine(t, s)
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`))
	stub.Script("GET", "/v1/docs?cursor=c2&limit=20", page(`{"items":[{"id":"doc-2"}],"next_cursor":"c3"}`))
	stub.Script("GET", "/v1/docs?cursor=c3&limit=20", page(`{"items":[{"id":"doc-3"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(stub.Requests()) != 2 || len(rec.Items()) != 2 {
		t.Fatalf("got %d requests and %d items, want 2 and 2 at the cap", len(stub.Requests()), len(rec.Items()))
	}
}

// TestGatewayListCollectionsReadsTheRoleOperation checks the role read that lists collections.
func TestGatewayListCollectionsReadsTheRoleOperation(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/collections", page(`{"items":[{"id":"col-1","name":"Handbook"}]}`))
	cols, err := eng.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(cols) != 1 || cols[0].ID != "col-1" || cols[0].Name != "Handbook" {
		t.Fatalf("collections = %+v, want one col-1 named Handbook", cols)
	}
}

// TestGatewayReadsWithoutARoleAreUnsupported checks the gateway reads that the spec does not serve.
func TestGatewayReadsWithoutARoleAreUnsupported(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	col, err := domain.NewCollectionID("col-1")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := domain.NewDocumentID("doc-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.CollectionTree(context.Background(), col); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("CollectionTree err = %v, want ErrUnsupportedOperation", err)
	}
	if _, err := eng.Document(context.Background(), doc); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Document err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an unsupported gateway read sent %d requests", n)
	}
}

// TestAWriteOperationIsRefusedBeforeAnyRequest checks that the reader refuses a change, and sends nothing.
func TestAWriteOperationIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "x"}))
	if !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("Read of a write = %v, want ErrWrongOperationClass", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused write sent %d requests", n)
	}
}

// TestAnUnknownOperationIsUnsupported checks an operation the spec does not declare.
func TestAnUnknownOperationIsUnsupported(t *testing.T) {
	eng, _, _ := newStubEngine(t, loadSpec(t))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.nothing", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
}

// TestAConverterIsUnsupportedBeforeAnyRequest checks that a read needing a converter of a later pull request fails
// closed, and sends nothing.
func TestAConverterIsUnsupportedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an unsupported converter sent %d requests", n)
	}
}

// TestS2_AnUndeclaredArgumentIsRefusedBeforeAnyRequest checks that a name the spec does not declare never reaches a
// request.
func TestS2_AnUndeclaredArgumentIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(map[string]any{"undeclared": "x"}))
	if !errors.Is(err, domain.ErrUnknownArgument) {
		t.Fatalf("err = %v, want ErrUnknownArgument", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an undeclared argument sent %d requests", n)
	}
}

// TestS2_ARequestCarriesOnlyTheDeclaredHeaders checks that the request holds the headers of the spec, the accept header,
// and the authorization header, and nothing else that the environment could add.
func TestS2_ARequestCarriesOnlyTheDeclaredHeaders(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := map[string]bool{}
	for name := range stub.Requests()[0].Header {
		got[name] = true
	}
	want := map[string]bool{"Accept": true, "Authorization": true, "X-Example-Client": true}
	if len(got) != len(want) {
		t.Fatalf("headers = %v, want exactly %v", got, want)
	}
	for name := range want {
		if !got[name] {
			t.Fatalf("header %s is missing", name)
		}
	}
}

// TestAMissingPathArgumentIsRefusedBeforeAnyRequest checks that a required path parameter must be set, and that a
// value of the wrong type is refused by its kind, before any request leaves.
func TestAMissingPathArgumentIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(nil)); !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("a missing path argument = %v, want ErrMissingArgument", err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": "many"})); err == nil {
		t.Fatal("a text limit was accepted, want a refusal by the Limit kind")
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused argument sent %d requests", n)
	}
}

// TestTheTimeLimitIsTheSpecLimitWhenLowerAndTheDefaultOtherwise checks the time limit of a request. A spec may lower
// the 30 second default, and it never raises it.
func TestTheTimeLimitIsTheSpecLimitWhenLowerAndTheDefaultOtherwise(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"the spec limit of twenty seconds", wikiYAML, "20s"},
		{"a spec limit lowered to three seconds", lowered(t, "timeout_seconds: 20", "timeout_seconds: 3"), "3s"},
		{"no spec limit, so the default", lowered(t, "limits:\n  timeout_seconds: 20\n  max_pages: 5\n", ""), "30s"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, _, clock := newStubEngineFrom(t, loadSpecFrom(t, c.doc))
			if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got := clock.limitsSeen()[0].String(); got != c.want {
				t.Fatalf("time limit = %s, want %s", got, c.want)
			}
		})
	}
}

// TestNewRefusesAnotherKind checks that a source of another kind is not bound to the spec.
func TestNewRefusesAnotherKind(t *testing.T) {
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"}, domain.KindDefinition{Name: "other-wiki"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := kinds.Kind("other-wiki")
	if err != nil {
		t.Fatal(err)
	}
	src := testSourceOfKind(t, other)
	if _, err := engine.New(loadSpec(t), src, stubPortsFor(t, loadSpec(t))); !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("err = %v, want ErrWrongSourceKind", err)
	}
}

// TestNewRefusesAHostOutsideTheSuffixOfTheSpec checks the host-suffix rule of the spec.
func TestNewRefusesAHostOutsideTheSuffixOfTheSpec(t *testing.T) {
	src := testSourceAt(t, "https://docs.example.invalid")
	if _, err := engine.New(loadSpec(t), src, stubPortsFor(t, loadSpec(t))); !errors.Is(err, domain.ErrInvalidEndpoint) {
		t.Fatalf("err = %v, want ErrInvalidEndpoint", err)
	}
}

// TestNewRefusesAnUnconfiguredEngine checks each port the engine needs before it may run.
func TestNewRefusesAnUnconfiguredEngine(t *testing.T) {
	s := loadSpec(t)
	src := testSource(t)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{}
	cases := []struct {
		name string
		s    *spec.Spec
		p    engine.Ports
	}{
		{"no spec", nil, engine.Ports{Credentials: &fakeCredentials{}, Clock: clock, Transport: stub}},
		{"no credentials port", s, engine.Ports{Clock: clock, Transport: stub}},
		{"no clock", s, engine.Ports{Credentials: &fakeCredentials{}, Transport: stub}},
		{"no transport and no dialer", s, engine.Ports{Credentials: &fakeCredentials{}, Clock: clock}},
		{"no transport and no lookup", s, engine.Ports{Credentials: &fakeCredentials{}, Clock: clock, Network: engine.Network{Dial: noDial}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := engine.New(c.s, src, c.p); !errors.Is(err, engine.ErrNotConfigured) {
				t.Fatalf("err = %v, want ErrNotConfigured", err)
			}
		})
	}
}

// TestNewRefusesBasicAuthWithoutIdentities checks that basic authentication needs the identity port.
func TestNewRefusesBasicAuthWithoutIdentities(t *testing.T) {
	s := basicSpec(t)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	p := engine.Ports{Credentials: &fakeCredentials{}, Clock: &fakeClock{}, Transport: stub}
	if _, err := engine.New(s, basicSource(t), p); !errors.Is(err, engine.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// TestBasicAuthSendsTheIdentityAndTheKey checks the basic form of the authorization header.
func TestBasicAuthSendsTheIdentityAndTheKey(t *testing.T) {
	s := basicSpec(t)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := engine.New(s, basicSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Identities: fakeIdentities{}, Clock: &fakeClock{}, Transport: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":null}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("reader@example.test:"+fixtureKey))
	if got := stub.Requests()[0].Header.Get("Authorization"); got != want {
		t.Fatal("the basic authorization header is not the identity and key pair")
	}
}

// TestHeaderAuthSendsTheKeyInTheDeclaredHeader checks the header form of authentication.
func TestHeaderAuthSendsTheKeyInTheDeclaredHeader(t *testing.T) {
	doc := replaceIn(t, wikiYAML, "type: bearer", "type: header")
	doc = replaceIn(t, doc, "  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: X-Example-Token")
	s := loadSpecFrom(t, doc)
	eng, stub, _ := newStubEngineFrom(t, s)
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := stub.Requests()[0].Header.Get("X-Example-Token"); got != fixtureKey {
		t.Fatal("the key is not in the declared header")
	}
	if got := stub.Requests()[0].Header.Get("Authorization"); got != "" {
		t.Fatal("header authentication also set an Authorization header")
	}
}

// TestNewRefusesAModelThatTheSpecWouldRefuse checks that a model edited by hand is refused by the same rules the spec
// applies: a partial slot, a reserved header, and a pointer that does not start with a slash.
func TestNewRefusesAModelThatTheSpecWouldRefuse(t *testing.T) {
	cases := []struct {
		name string
		edit func(*spec.Spec)
	}{
		{"a partial slot in a path", func(s *spec.Spec) { s.Operations[0].Path = "/docs/x{id}" }},
		{"a dot segment in a path", func(s *spec.Spec) { s.Operations[0].Path = "/docs/../x" }},
		{"a reserved header", func(s *spec.Spec) { s.Operations[0].Headers = map[string]string{"Authorization": "x"} }},
		{"a pointer without a slash", func(s *spec.Spec) {
			s.Operations[0].Response.Fields["id"] = spec.Field{Pointer: "id"}
		}},
		{"a list pointer without a slash", func(s *spec.Spec) { s.Operations[0].Response.Items = "items" }},
		{"a cursor pointer without a slash", func(s *spec.Spec) { s.Operations[0].Pagination.NextPointer = "next" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := loadSpec(t)
			c.edit(s)
			_, err := engine.New(s, testSource(t), stubPortsFor(t, loadSpec(t))) // the stub of the pristine fixture, not of the edited model
			if !errors.Is(err, spec.ErrInvalid) {
				t.Fatalf("err = %v, want the spec's invalid error", err)
			}
		})
	}
}

// TestNewAcceptsTheFixtureWithoutEdit checks that the engine accepts the validated fixture as it is.
func TestNewAcceptsTheFixtureWithoutEdit(t *testing.T) {
	if _, err := engine.New(loadSpec(t), testSource(t), stubPortsFor(t, loadSpec(t))); err != nil {
		t.Fatalf("New on the fixture: %v", err)
	}
}

// newStubEngineFrom is newStubEngine for a spec that the test has already built.
func newStubEngineFrom(t *testing.T, s *spec.Spec) (*engine.Engine, *replay.Stub, *fakeClock) {
	t.Helper()
	return newStubEngine(t, s)
}

// stubPortsFor returns the ports of an engine that answers from a stub built for the spec.
func stubPortsFor(t *testing.T, s *spec.Spec) engine.Ports {
	t.Helper()
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	return stubPorts(stub, &fakeClock{})
}

// basicSpec returns the fixture with basic authentication, which names an identity variable.
func basicSpec(t *testing.T) *spec.Spec {
	t.Helper()
	doc := replaceIn(t, wikiYAML, "type: bearer", "type: basic")
	doc = replaceIn(t, doc, "  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL")
	return loadSpecFrom(t, doc)
}

// basicSource returns a source of the fixture kind that names an identity variable.
func basicSource(t *testing.T) domain.Source {
	t.Helper()
	id, err := domain.NewIdentityRef("EXAMPLE_WIKI_EMAIL")
	if err != nil {
		t.Fatal(err)
	}
	return testSource(t).WithIdentity(id)
}

// testSourceOfKind returns a source of the given kind on the documentation host.
func testSourceOfKind(t *testing.T, kind domain.SourceKind) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("wiki-other")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(id, kind, tags, endpoint, cred)
}

// testSourceAt returns a source of the fixture kind on the given endpoint.
func testSourceAt(t *testing.T, endpointURL string) domain.Source {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := kinds.Kind("example-wiki")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint(endpointURL)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewSourceID("wiki-main")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(id, kind, tags, endpoint, cred)
}

// noDial is a dialer that the test never expects to be called.
func noDial(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("no dial is expected in this test")
}

// page returns a 200 reply with the given JSON body.
func page(body string) replay.Reply {
	return replay.Reply{Status: 200, Body: []byte(body)}
}
