package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// fakeGoOutline is the Go gateway of a composite test. It records the calls it receives and answers canned records,
// so a test sees which side of the composite served an operation.
type fakeGoOutline struct {
	mu         sync.Mutex
	reads      []domain.OperationID
	writes     []domain.OperationID
	lists      int
	trees      int
	documents  int
	readRecord domain.Record
	readErr    error
}

func (f *fakeGoOutline) ListCollections(context.Context) ([]domain.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	return nil, nil
}

func (f *fakeGoOutline) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.trees++
	return nil, nil
}

func (f *fakeGoOutline) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.documents++
	return domain.SourceDocument{}, nil
}

func (f *fakeGoOutline) Read(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, op)
	return f.readRecord, f.readErr
}

func (f *fakeGoOutline) Write(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, op)
	return domain.Record{}, nil
}

// cannedEngineTransport answers the Outline methods a composite test names. Unnamed methods get a 404 without a
// router message, so the engine reports a plain not found.
type cannedEngineTransport struct {
	mu      sync.Mutex
	methods map[string]cannedReply
	calls   []string
}

type cannedReply struct {
	status int
	body   string
}

func (c *cannedEngineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	method := strings.TrimPrefix(r.URL.EscapedPath(), "/api/")
	c.calls = append(c.calls, r.Method+" "+method)
	rep, ok := c.methods[method]
	if !ok {
		rep = cannedReply{status: 404, body: `{"ok":false,"message":"nothing here"}`}
	}
	return &http.Response{
		StatusCode: rep.status,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte(rep.body))),
	}, nil
}

func (c *cannedEngineTransport) called(method string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, call := range c.calls {
		if call == "POST "+method {
			return true
		}
	}
	return false
}

// compositeFixture builds the composite of a test: the bundled outline.yml model, a fake Go gateway, and an engine
// over the canned transport. The credentials resolve every key to a fixed value, so no environment is read.
func compositeFixture(t *testing.T, transport *cannedEngineTransport) (*outlineComposite, *fakeGoOutline) {
	t.Helper()
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil {
		t.Fatal("the bundled set holds no outline spec")
	}
	src := seamSourceOf(t, "outline", "https://docs.example.test", "OUTLINE_API_KEY")
	goGW := &fakeGoOutline{}
	eng, err := engine.New(model, src, engine.Ports{
		Credentials: stubCredentials{},
		Clock:       fakeClock{},
		Transport:   transport,
	})
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	reader, ok := any(goGW).(ports.SourceReader)
	if !ok {
		t.Fatal("the fake Go gateway is not a reader")
	}
	writer, ok := any(goGW).(ports.SourceWriter)
	if !ok {
		t.Fatal("the fake Go gateway is not a writer")
	}
	return &outlineComposite{goGateway: goGW, goReader: reader, goWriter: writer, engine: eng}, goGW
}

// stubCredentials resolves every key to one fixed value, so an engine of a test signs its requests.
type stubCredentials struct{}

func (stubCredentials) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey("OUTLINE_API_KEY", "test-key")
}

// compositeCredentials returns credentials whose Outline key is set, so a Go gateway built in a test reads it
// without an environment.
func compositeCredentials(t *testing.T) *envfile.Credentials {
	t.Helper()
	creds, err := envfile.NewWithFiles("/work/project/keys.env", func(name string) string {
		if name == "OUTLINE_API_KEY" {
			return "test-key"
		}
		return ""
	}, envfile.Files{FS: fstest.MapFS{}, SameFile: sameName})
	if err != nil {
		t.Fatalf("envfile.NewWithFiles: %v", err)
	}
	return creds
}

// TestCompositeServesAnEngineReadOnTheEngine proves the routing of pull request 16: a read the spec serves runs on
// the engine, and the Go gateway sees no call. collections.info is the probe: the canned reply carries one
// collection, and the composite must map it with the spec.
func TestCompositeServesAnEngineReadOnTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"collections.info": {status: 200, body: `{"data":{"id":"c1","name":"General"}}`},
	}}
	gw, goGW := compositeFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpCollectionsInfo,
		domain.NewArgs(map[string]any{"collectionId": "c1"}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if got := rec.Str("id"); got != "c1" {
		t.Fatalf("id = %q, want c1", got)
	}
	if got := rec.Str("name"); got != "General" {
		t.Fatalf("name = %q, want General", got)
	}
	if len(goGW.reads) != 0 {
		t.Fatalf("the Go gateway served %v, want no call", goGW.reads)
	}
	if !transport.called("collections.info") {
		t.Fatal("the engine sent no request to collections.info")
	}
}

// TestCompositeKeepsFlaggedReadsOnGo proves the other side of the routing: collections.list and documents.get stay
// on the Go gateway, because the spec cannot page the first and cannot tell the 404s of the second apart. The canned
// engine transport must see no call.
func TestCompositeKeepsFlaggedReadsOnGo(t *testing.T) {
	transport := &cannedEngineTransport{}
	gw, goGW := compositeFixture(t, transport)
	for _, op := range []domain.OperationID{domain.OpCollectionsList, domain.OpDocumentsGet} {
		if _, err := gw.Read(context.Background(), op, domain.NewArgs(nil)); err != nil {
			t.Fatalf("Read(%s) = %v, want nil", op, err)
		}
	}
	if len(goGW.reads) != 2 || goGW.reads[0] != domain.OpCollectionsList || goGW.reads[1] != domain.OpDocumentsGet {
		t.Fatalf("go reads = %v, want the two flagged reads", goGW.reads)
	}
	if len(transport.calls) != 0 {
		t.Fatalf("the engine sent %v, want no call", transport.calls)
	}
}

// TestCompositeRestoresTheNotFoundSentinel proves the 404 translation: the engine reports every 404 as a plain not
// found, and the composite restores the collection and document sentinels the Go gateway maps. A first-item read with
// an empty list keeps the plain not found, and an unavailable-method mapping is never translated.
func TestCompositeRestoresTheNotFoundSentinel(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"shares.info": {status: 200, body: `{"data":{"shares":[]}}`},
	}}
	gw, _ := compositeFixture(t, transport)
	if _, err := gw.Read(context.Background(), domain.OpDocumentsSearch,
		domain.NewArgs(map[string]any{"query": "setup"})); engine.Code("outline", err) != "retriever.outline.operation_unavailable" {
		t.Fatalf("documents.search 404 = %v, want operation_unavailable untouched", err)
	}
	if _, err := gw.Read(context.Background(), domain.OpCollectionsInfo,
		domain.NewArgs(map[string]any{"collectionId": "c1"})); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("collections.info 404 = %v, want collection_not_found", err)
	}
	if _, err := gw.Read(context.Background(), domain.OpDocumentsChildren,
		domain.NewArgs(map[string]any{"documentId": "d1"})); !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("documents.children 404 = %v, want document_not_found", err)
	}
	if _, err := gw.Read(context.Background(), domain.OpSharesGet,
		domain.NewArgs(map[string]any{"shareId": "s1"})); !errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrCollectionNotFound) || errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("shares.get empty list = %v, want the plain not found", err)
	}
}

// TestBuildOutlineCompositeBuildsBothSides proves the constructor: the bundled model and a source build a composite
// whose Go side answers without a network. A Go-served read fails in the fake client, never in the engine, and an
// engine-served read fails past the fake, in the engine's refused network.
func TestBuildOutlineCompositeBuildsBothSides(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil {
		t.Fatal("the bundled set holds no outline spec")
	}
	src := seamSourceOf(t, "outline", "https://docs.example.test", "OUTLINE_API_KEY")
	creds := compositeCredentials(t)
	seam := seamConfig{base: "https://127.0.0.1:1", client: fakeClient(), on: true}
	canned := &cannedEngineTransport{}
	eseam := engineSeam{clock: fakeClock{}, transport: canned}
	gw, err := buildOutlineComposite(model, src, creds, seam, eseam)
	if err != nil {
		t.Fatalf("buildOutlineComposite: %v", err)
	}
	comp, ok := gw.(*outlineComposite)
	if !ok || comp.engine == nil || comp.goGateway == nil {
		t.Fatalf("built %T, want a composite with both sides", gw)
	}
	if _, err := gw.(ports.SourceReader).Read(context.Background(), domain.OpDocumentsGet,
		domain.NewArgs(map[string]any{"documentId": "d1"})); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("documents.get = %v, want the Go client's unreachable", err)
	}
	if len(canned.calls) != 0 {
		t.Fatalf("the engine sent %v for a Go-served read", canned.calls)
	}
	if _, err := gw.(ports.SourceReader).Read(context.Background(), domain.OpCollectionsInfo,
		domain.NewArgs(map[string]any{"collectionId": "c1"})); err == nil ||
		!strings.Contains(err.Error(), "sends no request") {
		t.Fatalf("collections.info = %v, want the seam fake's refusal, never the canned transport", err)
	}
	if len(canned.calls) != 0 {
		t.Fatalf("the engine sent %v, want the seam transport only", canned.calls)
	}
	if _, err := buildOutlineComposite(nil, src, creds, seam, eseam); err == nil {
		t.Fatal("buildOutlineComposite without a model succeeds")
	}
}

// TestBuildOutlineEngineHonorsTheSeam proves the seam path of the composite engine: with the seam on, the engine
// sends through the seam client, so it reaches the same stub the Go gateway reaches. With the seam off it builds
// over the given seam, and a base it cannot parse stops the source.
func TestBuildOutlineEngineHonorsTheSeam(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil {
		t.Fatal("the bundled set holds no outline spec")
	}
	src := seamSourceOf(t, "outline", "https://docs.example.test", "OUTLINE_API_KEY")
	creds := compositeCredentials(t)
	eseam := engineSeam{clock: fakeClock{}, transport: &cannedEngineTransport{}}
	if _, err := buildOutlineEngine(model, src, creds, seamConfig{}, eseam); err != nil {
		t.Fatalf("buildOutlineEngine without a seam = %v, want nil", err)
	}
	seam := seamConfig{base: "https://127.0.0.1:1", client: fakeClient(), on: true}
	eng, err := buildOutlineEngine(model, src, creds, seam, eseam)
	if err != nil {
		t.Fatalf("buildOutlineEngine with a seam = %v, want nil", err)
	}
	if _, err := eng.Read(context.Background(), specOperationID(domain.OpCollectionsInfo),
		domain.NewArgs(map[string]any{"collectionId": "c1"})); err == nil ||
		!strings.Contains(err.Error(), "could not be reached") {
		t.Fatalf("collections.info = %v, want the seam client's refusal", err)
	}
	seam.base = "://no-scheme"
	if _, err := buildOutlineEngine(model, src, creds, seam, eseam); err == nil {
		t.Fatal("buildOutlineEngine with a base it cannot parse succeeds")
	}
}

// TestCompositeSendsWritesAndLegacyReadsToGo proves what stays with the Go gateway until pull request 17: change
// operations, and the legacy collection, tree, and document reads the GraphQL service and the old commands call.
func TestCompositeSendsWritesAndLegacyReadsToGo(t *testing.T) {
	transport := &cannedEngineTransport{}
	gw, goGW := compositeFixture(t, transport)
	if _, err := gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(nil)); err != nil {
		t.Fatalf("Write = %v, want nil", err)
	}
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections = %v, want nil", err)
	}
	collID, err := domain.NewCollectionID("c1")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	if _, err := gw.CollectionTree(context.Background(), collID); err != nil {
		t.Fatalf("CollectionTree = %v, want nil", err)
	}
	docID, err := domain.NewDocumentID("d1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	if _, err := gw.Document(context.Background(), docID); err != nil {
		t.Fatalf("Document = %v, want nil", err)
	}
	if len(goGW.writes) != 1 || goGW.writes[0] != domain.OpDocumentsCreate {
		t.Fatalf("go writes = %v, want the one create", goGW.writes)
	}
	if goGW.lists != 1 || goGW.trees != 1 || goGW.documents != 1 {
		t.Fatalf("legacy calls = %d/%d/%d, want one each", goGW.lists, goGW.trees, goGW.documents)
	}
	if len(transport.calls) != 0 {
		t.Fatalf("the engine sent %v, want no call", transport.calls)
	}
}

// TestSpecOperationIDPrefixesTheCatalogKind proves the engine lookup: the service passes the catalog operation, and
// the bundled spec serves it under the kind prefix.
func TestSpecOperationIDPrefixesTheCatalogKind(t *testing.T) {
	if got := specOperationID(domain.OpDocumentsList); got != "outline.documents.list" {
		t.Fatalf("specOperationID = %q, want outline.documents.list", got)
	}
}

// TestBundledOutlineFindsTheBundledModel proves the wiring lookup: the merged set holds the bundled outline.yml, and
// a set without specs holds none, so it keeps the pure Go gateway.
func TestBundledOutlineFindsTheBundledModel(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil || model.Kind != "outline" {
		t.Fatalf("bundledOutline = %+v, want the outline model", model)
	}
	if bundledOutline(connectorspec.Set{}) != nil {
		t.Fatal("bundledOutline of an empty set is not nil")
	}
}
