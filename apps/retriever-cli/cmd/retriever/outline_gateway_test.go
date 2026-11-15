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
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// cannedEngineTransport answers the Outline methods a gateway test names. Unnamed methods get a 404 without a
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

// gatewayFixture builds the Outline gateway of a test: the bundled outline.yml model over the canned transport.
// The credentials resolve every key to a fixed value, so no environment is read.
func gatewayFixture(t *testing.T, transport *cannedEngineTransport) *outlineGateway {
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
	creds := gatewayCredentials(t)
	eseam := engineSeam{clock: fakeClock{}, transport: transport}
	gw, err := buildOutlineGateway(model, src, creds, seamConfig{}, eseam)
	if err != nil {
		t.Fatalf("buildOutlineGateway: %v", err)
	}
	outline, ok := gw.(*outlineGateway)
	if !ok {
		t.Fatalf("built %T, want the Outline gateway", gw)
	}
	if _, ok := any(gw).(ports.SourceGateway); !ok {
		t.Fatal("the Outline gateway is not a source gateway")
	}
	return outline
}

// gatewayCredentials returns credentials whose Outline key is set, so a gateway built in a test reads it
// without an environment.
func gatewayCredentials(t *testing.T) *envfile.Credentials {
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

// TestGatewayServesAReadOnTheEngine proves the read routing: a read the spec serves runs on the engine, and the
// canned reply maps to its fields. collections.info is the probe: the canned reply carries one collection.
func TestGatewayServesAReadOnTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"collections.info": {status: 200, body: `{"data":{"id":"c1","name":"General"}}`},
	}}
	gw := gatewayFixture(t, transport)
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
	if !transport.called("collections.info") {
		t.Fatal("the engine sent no request to collections.info")
	}
}

// TestGatewayServesDocumentsGetOnTheEngine proves the documents.get routing: the bundled-spec engine maps the
// wrapped reply shape through the primary pointers, with the Markdown fallback still armed.
func TestGatewayServesDocumentsGetOnTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"documents.info": {status: 200, body: `{"data":{"document":{"id":"d1","title":"Setup","text":"Hello"}}}`},
	}}
	gw := gatewayFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "d1"}))
	if err != nil {
		t.Fatalf("Read(documents.get) wrapped = %v, want nil", err)
	}
	if got := rec.Str("title"); got != "Setup" {
		t.Fatalf("title = %q, want Setup", got)
	}
	if got := rec.Str("markdown"); got != "Hello" {
		t.Fatalf("markdown = %q, want Hello", got)
	}
}

// TestGatewayServesTheUnwrappedDocumentShape proves the second shape: the unwrapped reply maps through the
// fallback pointers with no export fallback, because its text is present.
func TestGatewayServesTheUnwrappedDocumentShape(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"documents.info": {status: 200, body: `{"data":{"id":"d1","title":"Setup","text":"Hello"}}`},
	}}
	gw := gatewayFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "d1"}))
	if err != nil {
		t.Fatalf("Read(documents.get) unwrapped = %v, want nil", err)
	}
	if got := rec.Str("id"); got != "d1" {
		t.Fatalf("id = %q, want d1", got)
	}
	if transport.called("documents.export") {
		t.Fatal("the engine read the export although the reply holds text")
	}
}

// TestGatewayTellsAMissingDocumentFromAMissingMethod proves the 404 disambiguation: another document's 404 keeps
// the document-not-found sentinel, and the router's catch-all 404 keeps the unavailable-method failure.
func TestGatewayTellsAMissingDocumentFromAMissingMethod(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"documents.info": {status: 404, body: `{"ok":false,"message":"No document with this ID"}`},
	}}
	gw := gatewayFixture(t, transport)
	_, err := gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "d9"}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
	transport.methods["documents.info"] = cannedReply{status: 404, body: `{"ok":false,"message":"Endpoint not found"}`}
	_, err = gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "d9"}))
	var unavailable domain.OperationUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want OperationUnavailableError", err)
	}
}

// TestGatewayRestoresTheNotFoundSentinel proves the 404 translation: the engine reports every 404 as a plain not
// found, and the gateway restores the collection and document sentinels the retired Go gateway mapped. A first-item
// read with an empty list keeps the plain not found, and an unavailable-method mapping is never translated.
func TestGatewayRestoresTheNotFoundSentinel(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"shares.info": {status: 200, body: `{"data":{"shares":[]}}`},
	}}
	gw := gatewayFixture(t, transport)
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

// TestBuildOutlineGatewayBuildsTheEngine proves the constructor: the bundled model and a source build a gateway
// whose engine serves the spec. A missing model stops the build, and a seam base the gateway cannot parse stops it
// too, before any request is sent.
func TestBuildOutlineGatewayBuildsTheEngine(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil {
		t.Fatal("the bundled set holds no outline spec")
	}
	src := seamSourceOf(t, "outline", "https://docs.example.test", "OUTLINE_API_KEY")
	creds := gatewayCredentials(t)
	eseam := engineSeam{clock: fakeClock{}, transport: &cannedEngineTransport{}}
	if _, err := buildOutlineGateway(nil, src, creds, seamConfig{}, eseam); err == nil {
		t.Fatal("buildOutlineGateway without a model succeeds")
	}
	seam := seamConfig{base: "https://127.0.0.1:1", client: fakeClient(), on: true}
	if _, err := buildOutlineGateway(model, src, creds, seam, eseam); err != nil {
		t.Fatalf("buildOutlineGateway with a seam = %v, want nil", err)
	}
	seam.base = "://no-scheme"
	if _, err := buildOutlineGateway(model, src, creds, seam, eseam); err == nil {
		t.Fatal("buildOutlineGateway with a base it cannot parse succeeds")
	}
	seam.base = "http://127.0.0.1:1"
	if _, err := buildOutlineGateway(model, src, creds, seam, eseam); err == nil {
		t.Fatal("buildOutlineGateway with a plain HTTP base succeeds")
	}
}

// TestGatewayBoundsEngineRequestsByTheSeamTimeout proves the timeout wiring: a seam with a timeout bounds the engine
// the same way it bounded the retired Go gateway, so a test seam stops a run that gets no reply. The canned
// transport answers at once, so the bound never fires here.
func TestGatewayBoundsEngineRequestsByTheSeamTimeout(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"collections.list": {status: 200, body: `{"data":[]}`},
	}}
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledOutline(set)
	if model == nil {
		t.Fatal("the bundled set holds no outline spec")
	}
	src := seamSourceOf(t, "outline", "https://docs.example.test", "OUTLINE_API_KEY")
	creds := gatewayCredentials(t)
	seam := seamConfig{base: "https://127.0.0.1:1", client: &http.Client{Transport: transport}, on: true, timeout: 50 * time.Millisecond}
	eseam := engineSeam{clock: fakeClock{}, transport: transport}
	gw, err := buildOutlineGateway(model, src, creds, seam, eseam)
	if err != nil {
		t.Fatalf("buildOutlineGateway: %v", err)
	}
	outline, ok := gw.(*outlineGateway)
	if !ok {
		t.Fatalf("built %T, want the Outline gateway", gw)
	}
	if _, err := outline.Read(context.Background(), domain.OpCollectionsList, domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read(collections.list) = %v, want nil", err)
	}
}

// TestGatewaySendsWritesToTheEngine proves the write routing: a write the bundled spec serves runs on the engine,
// and the engine's single request carries the operation. The page-all collections list runs on the engine too, and
// the legacy tree and document reads run on the engine reads they name.
func TestGatewaySendsWritesToTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"documents.create": {status: 200, body: `{"data":{"id":"d1","title":"Setup"}}`},
		"collections.list": {status: 200, body: `{"data":[{"id":"c1","name":"General"}]}`},
		"collections.documents": {status: 200, body: `{"data":[{"id":"d1","title":"Setup","url":"/doc/d1",` +
			`"children":[{"id":"d2","title":"Child","children":[]}]}]}`},
		"documents.info": {status: 200, body: `{"data":{"document":{"id":"d1","title":"Setup","url":"/doc/d1",` +
			`"text":"Hello","collectionId":"c1"}}}`},
	}}
	gw := gatewayFixture(t, transport)
	rec, err := gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Write = %v, want nil", err)
	}
	if got := rec.Str("id"); got != "d1" {
		t.Fatalf("id = %q, want d1", got)
	}
	if !transport.called("documents.create") {
		t.Fatal("the engine sent no request to documents.create")
	}
	colls, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections = %v, want nil", err)
	}
	if len(colls) != 1 || colls[0].ID != "c1" || colls[0].Name != "General" {
		t.Fatalf("collections = %+v, want the one canned collection", colls)
	}
	if !transport.called("collections.list") {
		t.Fatal("the engine sent no request to collections.list")
	}
	collID, err := domain.NewCollectionID("c1")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	nodes, err := gw.CollectionTree(context.Background(), collID)
	if err != nil {
		t.Fatalf("CollectionTree = %v, want nil", err)
	}
	if len(nodes) != 2 || nodes[0].ID != "d1" || nodes[0].Depth != 0 || nodes[0].ParentID != nil {
		t.Fatalf("nodes = %+v, want the root first with no parent", nodes)
	}
	if nodes[1].ID != "d2" || nodes[1].Depth != 1 {
		t.Fatalf("nodes = %+v, want the child second at depth 1", nodes)
	}
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "d1" {
		t.Fatalf("nodes = %+v, want the child under d1", nodes)
	}
	docID, err := domain.NewDocumentID("d1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	doc, err := gw.Document(context.Background(), docID)
	if err != nil {
		t.Fatalf("Document = %v, want nil", err)
	}
	if doc.Markdown != "Hello" || doc.CollectionID != "c1" {
		t.Fatalf("document = %+v, want the canned body and collection", doc)
	}
}

// TestGatewayCollectionTreeKeepsTheCollectionSentinel proves the legacy tree read keeps its failure: a missing
// collection fails as collection-not-found, as the retired Go gateway mapped it.
func TestGatewayCollectionTreeKeepsTheCollectionSentinel(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"collections.documents": {status: 404, body: `{"ok":false,"message":"No collection"}`},
	}}
	gw := gatewayFixture(t, transport)
	collID, err := domain.NewCollectionID("c9")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	if _, err := gw.CollectionTree(context.Background(), collID); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("CollectionTree = %v, want ErrCollectionNotFound", err)
	}
}

// TestGatewayDocumentKeepsTheDocumentSentinel proves the legacy document read keeps its failure: a missing document
// fails as document-not-found, as the retired Go gateway mapped it.
func TestGatewayDocumentKeepsTheDocumentSentinel(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"documents.info": {status: 404, body: `{"ok":false,"message":"No document with this ID"}`},
	}}
	gw := gatewayFixture(t, transport)
	docID, err := domain.NewDocumentID("d9")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	if _, err := gw.Document(context.Background(), docID); !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("Document = %v, want ErrDocumentNotFound", err)
	}
}

// TestGatewayServesNotificationWritesOnTheEngine proves the notification routing: both subscription writes run on
// the bundled-spec engine. Each reply holds every field of its catalog row, so the use-case shape check has its
// known outcome, and the settings keep every event type the wiki returns, sorted, with the exact flags.
func TestGatewayServesNotificationWritesOnTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"users.notificationsSubscribe": {status: 200, body: `{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"name":"Ann","role":"member","isSuspended":false,"notificationSettings":` +
			`{"documents.update":true,"comments.create":false}}}`},
		"users.notificationsUnsubscribe": {status: 200, body: `{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"name":"Ann","role":"member","isSuspended":false,"notificationSettings":{"documents.update":false}}}`},
	}}
	gw := gatewayFixture(t, transport)
	for _, op := range []domain.OperationID{domain.OpUsersNotificationsSubscribe,
		domain.OpUsersNotificationsUnsubscribe} {
		rec, err := gw.Write(context.Background(), op, domain.NewArgs(nil))
		if err != nil {
			t.Fatalf("Write(%s) = %v, want nil", op, err)
		}
		row, ok := operations.Lookup(op)
		if !ok {
			t.Fatalf("no catalog row for %s", op)
		}
		for _, field := range row.Fields {
			if _, found := rec.Get(field); !found {
				t.Fatalf("Write(%s) has no field %q: fields = %v", op, field, rec.Fields())
			}
		}
	}
	rec, err := gw.Write(context.Background(), domain.OpUsersNotificationsSubscribe, domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Write(subscribe) = %v, want nil", err)
	}
	settings := rec.Rec("notificationSettings")
	var names []string
	for _, f := range settings.Fields() {
		names = append(names, f.Name)
	}
	if len(names) != 2 || names[0] != "comments.create" || names[1] != "documents.update" {
		t.Fatalf("settings keys = %v, want [comments.create documents.update]", names)
	}
	if v, _ := settings.Get("documents.update"); v != true {
		t.Fatalf("documents.update = %v, want true", v)
	}
	if !transport.called("users.notificationsSubscribe") || !transport.called("users.notificationsUnsubscribe") {
		t.Fatalf("the engine sent %v, want both subscription calls", transport.calls)
	}
}

// TestGatewayServesTheParamKindWritesOnTheEngine proves the seven writes that the Icon, URLList, and EventList
// kinds move run on the bundled-spec engine: each canned wiki reply maps to its declared fields. A blank icon is
// refused before any request, so the engine enforces the catalog validation.
func TestGatewayServesTheParamKindWritesOnTheEngine(t *testing.T) {
	transport := &cannedEngineTransport{methods: map[string]cannedReply{
		"templates.create":            {status: 200, body: `{"ok":true,"data":{"id":"t1","title":"Weekly"}}`},
		"templates.update":            {status: 200, body: `{"ok":true,"data":{"id":"t1","title":"Weekly"}}`},
		"groups.create":               {status: 200, body: `{"ok":true,"data":{"id":"223e4567-e89b-12d3-a456-426614174000","name":"Editors"}}`},
		"groups.update":               {status: 200, body: `{"ok":true,"data":{"id":"223e4567-e89b-12d3-a456-426614174000","name":"Editors"}}`},
		"oauthClients.create":         {status: 200, body: `{"ok":true,"data":{"id":"323e4567-e89b-12d3-a456-426614174000","name":"App"}}`},
		"webhookSubscriptions.create": {status: 200, body: `{"ok":true,"data":{"id":"423e4567-e89b-12d3-a456-426614174000","name":"Hook"}}`},
		"webhookSubscriptions.update": {status: 200, body: `{"ok":true,"data":{"id":"423e4567-e89b-12d3-a456-426614174000","name":"Hook"}}`},
	}}
	gw := gatewayFixture(t, transport)
	calls := []struct {
		op   domain.OperationID
		args map[string]any
		want string
	}{
		{domain.OpTemplatesCreate, map[string]any{"title": "Weekly", "icon": "star"}, "t1"},
		{domain.OpTemplatesUpdate, map[string]any{"templateId": "t1", "title": "Weekly"}, "t1"},
		{domain.OpGroupsCreate, map[string]any{"name": "Editors"}, "223e4567-e89b-12d3-a456-426614174000"},
		{domain.OpGroupsUpdate, map[string]any{"groupId": "123e4567-e89b-12d3-a456-426614174000", "name": "Editors"}, "223e4567-e89b-12d3-a456-426614174000"},
		{domain.OpOAuthClientsCreate, map[string]any{"name": "App", "redirectUris": []any{"https://app.example.test/callback"}}, "323e4567-e89b-12d3-a456-426614174000"},
		{domain.OpWebhookSubscriptionsCreate, map[string]any{"url": "https://hooks.example.test/synthetic", "name": "Hook", "events": []any{"documents.create"}}, "423e4567-e89b-12d3-a456-426614174000"},
		{domain.OpWebhookSubscriptionsUpdate, map[string]any{"webhookSubscriptionId": "123e4567-e89b-12d3-a456-426614174000", "url": "https://hooks.example.test/synthetic", "name": "Hook", "events": []any{"*"}}, "423e4567-e89b-12d3-a456-426614174000"},
	}
	for _, c := range calls {
		rec, err := gw.Write(context.Background(), c.op, domain.NewArgs(c.args))
		if err != nil {
			t.Fatalf("Write(%s) = %v, want nil", c.op, err)
		}
		if got := rec.Str("id"); got != c.want {
			t.Fatalf("Write(%s) id = %q, want %q", c.op, got, c.want)
		}
	}
	if _, err := gw.Write(context.Background(), domain.OpGroupsCreate,
		domain.NewArgs(map[string]any{"name": "Editors", "icon": "   "})); !errors.Is(err, domain.ErrInvalidIcon) {
		t.Fatalf("Write(groups.create) with a blank icon = %v, want ErrInvalidIcon", err)
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
// a set without specs holds none, so no Outline gateway is built from it.
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
