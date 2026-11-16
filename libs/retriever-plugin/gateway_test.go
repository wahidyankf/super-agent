package plugin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// fakeCredentials resolves one credential and one identity from memory, failing each lookup on demand.
type fakeCredentials struct {
	key         string
	keyErr      error
	identity    string
	identityErr error
	keys        int
	idents      int
}

func (f *fakeCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	f.keys++
	if f.keyErr != nil {
		return domain.ApiKey{}, f.keyErr
	}
	return domain.NewApiKey(ref.Name(), f.key)
}

func (f *fakeCredentials) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	f.idents++
	if f.identityErr != nil {
		return domain.Identity{}, f.identityErr
	}
	return domain.NewIdentity(ref.Name(), f.identity)
}

// mustPluginSource returns the outline source of the gateway fixtures: it carries a plugin whose pin matches
// pluginBytesV1, failing the test on any error.
func mustPluginSource(t *testing.T) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("docs")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	ref, err := domain.NewPluginRef("/opt/plugins/wiki", pinOf(pluginBytesV1))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return domain.NewSource(id, domain.KindOutline, mustTagSet(t), endpoint, credential).WithPlugin(ref)
}

// mustTagSet returns the empty tag set, failing the test on any error.
func mustTagSet(t *testing.T) domain.TagSet {
	t.Helper()
	set, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return set
}

// gatewaySeams returns a gateway over the source that answers reply, with the credentials holding the key. The
// fake files serve the source's plugin path, so the pin check hashes the copy of that path.
func gatewaySeams(t *testing.T, source domain.Source, reply string, creds *fakeCredentials) (*Gateway, *fakeFiles, *fakeStarter, *fakeProc) {
	t.Helper()
	files := &fakeFiles{files: map[string][]byte{source.Plugin().Path(): append([]byte(nil), pluginBytesV1...)}, dir: "/private/1"}
	starter := &fakeStarter{}
	proc := answerProc(reply)
	starter.proc = proc
	seams := Seams{
		Files: files,
		Start: starter.Start,
		After: func(time.Duration) <-chan time.Time { return openChan() },
	}
	return &Gateway{Source: source, Credentials: creds, Identities: creds, Seams: seams}, files, starter, proc
}

// pluginGateway returns the gateway of the plugin source over a fake world that answers reply.
func pluginGateway(t *testing.T, reply string) (*Gateway, *fakeStarter, *fakeProc) {
	t.Helper()
	creds := &fakeCredentials{key: "credential-sentinel-9f2a"}
	gw, _, starter, proc := gatewaySeams(t, mustPluginSource(t), reply, creds)
	return gw, starter, proc
}

// TestGatewayResolvesSecretsPerCall checks that the gateway reads the source's credential for each call and passes
// the value to the plugin only through the scrubbed environment: the secret never reaches an error.
func TestGatewayResolvesSecretsPerCall(t *testing.T) {
	gw, starter, _ := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"collections":[]}}`)
	collections, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v, want nil", err)
	}
	if len(collections) != 0 {
		t.Fatalf("collections = %+v, want none", collections)
	}
	if len(starter.env) != 1 || starter.env[0] != "EXAMPLE_OUTLINE_API_KEY=credential-sentinel-9f2a" {
		t.Fatalf("env = %q, want only the credential", starter.env)
	}
}

// TestGatewayRefusesAPluginlessSource checks that a gateway built for a source with no plugin fails before any
// secret is read: the routing owns the check, and the gateway repeats it.
func TestGatewayRefusesAPluginlessSource(t *testing.T) {
	plain := mustPluginSource(t)
	id, _ := domain.NewSourceID("docs")
	endpoint, _ := domain.NewEndpoint("https://docs.example.test")
	credential, _ := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	plain = domain.NewSource(id, domain.KindOutline, mustTagSet(t), endpoint, credential)
	creds := &fakeCredentials{key: "credential-sentinel-9f2a"}
	gw, _, _, _ := gatewaySeams(t, plain, goodReply, creds)
	if _, err := gw.ListCollections(context.Background()); err == nil {
		t.Fatal("ListCollections accepted a source with no plugin")
	} else if start, ok := err.(StartError); !ok || start.Rule != "the source has no plugin" {
		t.Fatalf("ListCollections error = %#v, want the no-plugin StartError", err)
	}
	if creds.keys != 0 {
		t.Fatalf("resolved %d keys, want none before the plugin check", creds.keys)
	}
}

// TestGatewayPropagatesSecretFailures checks that a missing key and a bad identity fail the call with their own
// errors, before any process starts.
func TestGatewayPropagatesSecretFailures(t *testing.T) {
	boom := errors.New("boom")
	creds := &fakeCredentials{keyErr: boom}
	gw, _, _, _ := gatewaySeams(t, mustPluginSource(t), goodReply, creds)
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("ListCollections error = %v, want the key failure", err)
	}
	withIdentity := mustIdentityPluginSource(t)
	creds = &fakeCredentials{key: "k", identityErr: boom}
	gw, _, _, _ = gatewaySeams(t, withIdentity, goodReply, creds)
	id, err := domain.NewCollectionID("c")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	if _, err := gw.CollectionTree(context.Background(), id); !errors.Is(err, boom) {
		t.Fatalf("CollectionTree error = %v, want the identity failure", err)
	}
}

// mustIdentityPluginSource returns the source of the identity fixtures: a plugin source with an identity name.
func mustIdentityPluginSource(t *testing.T) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("tracker")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://tracker.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_JIRA_API_TOKEN")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	identity, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	ref, err := domain.NewPluginRef("/opt/plugins/tracker", pinOf(pluginBytesV1))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return domain.NewSource(id, domain.KindOutline, mustTagSet(t), endpoint, credential).WithIdentity(identity).WithPlugin(ref)
}

// TestGatewayListsCollections checks the list_collections mapping: collections map onto domain values in order.
func TestGatewayListsCollections(t *testing.T) {
	gw, _, proc := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"collections":[{"id":"c1","name":"One"},{"id":"c2","name":"Two"}]}}`)
	collections, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v, want nil", err)
	}
	if len(collections) != 2 || collections[0].ID != "c1" || collections[0].Name != "One" || collections[1].ID != "c2" {
		t.Fatalf("collections = %+v", collections)
	}
	if len(proc.wrote) != 1 || !containsAll(string(proc.wrote[0]), `"method":"list_collections"`, `"source":"docs"`) {
		t.Fatalf("request = %s, want the method with the source", proc.wrote[0])
	}
}

// TestGatewayRefusesBadCollections checks every bad list_collections shape: each fails with ProtocolError.
func TestGatewayRefusesBadCollections(t *testing.T) {
	cases := []struct {
		name  string
		reply string
		rule  string
	}{
		{"null result", `{"jsonrpc":"2.0","id":1,"result":null}`, "the plugin reply has no result object"},
		{"missing list", `{"jsonrpc":"2.0","id":1,"result":{}}`, "the plugin reply has no collections list"},
		{"mistyped list", `{"jsonrpc":"2.0","id":1,"result":{"collections":{}}}`, "the plugin reply has no collections list"},
		{"mistyped entry", `{"jsonrpc":"2.0","id":1,"result":{"collections":[1]}}`, "the plugin collection has no id and name"},
		{"missing name", `{"jsonrpc":"2.0","id":1,"result":{"collections":[{"id":"c1"}]}}`, "the plugin collection has no id and name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw, _, _ := pluginGateway(t, c.reply)
			if _, err := gw.ListCollections(context.Background()); err == nil {
				t.Fatal("ListCollections accepted a bad shape")
			} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != c.rule {
				t.Fatalf("ListCollections error = %#v, want rule %q", err, c.rule)
			}
		})
	}
}

// TestGatewayReadsTheCollectionTree checks the collection_tree mapping: a null parent reads as a root, and depths
// map onto numbers.
func TestGatewayReadsTheCollectionTree(t *testing.T) {
	gw, _, proc := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"nodes":[
		{"id":"d1","title":"Root","url":"https://docs.example.test/d1","parent_id":null,"depth":0},
		{"id":"d2","title":"Leaf","url":"https://docs.example.test/d2","parent_id":"d1","depth":1}]}}`)
	id, err := domain.NewCollectionID("c1")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	nodes, err := gw.CollectionTree(context.Background(), id)
	if err != nil {
		t.Fatalf("CollectionTree error = %v, want nil", err)
	}
	if len(nodes) != 2 || nodes[0].ParentID != nil || nodes[0].Depth != 0 {
		t.Fatalf("root = %+v, want a root at depth 0", nodes[0])
	}
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "d1" || nodes[1].Depth != 1 {
		t.Fatalf("leaf = %+v, want d1 at depth 1", nodes[1])
	}
	if !containsAll(string(proc.wrote[0]), `"method":"collection_tree"`, `"collection_id":"c1"`) {
		t.Fatalf("request = %s, want the method with the collection", proc.wrote[0])
	}
}

// TestGatewayRefusesBadNodes checks every bad collection_tree shape.
func TestGatewayRefusesBadNodes(t *testing.T) {
	cases := []struct {
		name  string
		nodes string
		rule  string
	}{
		{"missing list", `{}`, "the plugin reply has no nodes list"},
		{"null result", `null`, "the plugin reply has no result object"},
		{"mistyped result", `5`, "the plugin reply has no result object"},
		{"missing title", `{"nodes":[{"id":"d1","url":"u","depth":0}]}`, "the plugin node has no id, title, and url"},
		{"bad parent", `{"nodes":[{"id":"d1","title":"t","url":"u","parent_id":1,"depth":0}]}`, "the plugin node parent is not text or null"},
		{"textual depth", `{"nodes":[{"id":"d1","title":"t","url":"u","depth":"0"}]}`, "the plugin node depth is not a whole number"},
		{"fractional depth", `{"nodes":[{"id":"d1","title":"t","url":"u","depth":0.5}]}`, "the plugin node depth is not a whole number"},
		{"negative depth", `{"nodes":[{"id":"d1","title":"t","url":"u","depth":-1}]}`, "the plugin node depth is not a whole number"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw, _, _ := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":`+c.nodes+`}`)
			id, err := domain.NewCollectionID("c1")
			if err != nil {
				t.Fatalf("NewCollectionID: %v", err)
			}
			if _, err := gw.CollectionTree(context.Background(), id); err == nil {
				t.Fatal("CollectionTree accepted a bad shape")
			} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != c.rule {
				t.Fatalf("CollectionTree error = %#v, want rule %q", err, c.rule)
			}
		})
	}
}

// TestGatewayReadsADocument checks the document mapping onto the domain value.
func TestGatewayReadsADocument(t *testing.T) {
	gw, _, proc := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{
		"id":"d1","title":"T","url":"https://docs.example.test/d1","markdown":"# T","collection_id":"c1"}}`)
	id, err := domain.NewDocumentID("d1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	doc, err := gw.Document(context.Background(), id)
	if err != nil {
		t.Fatalf("Document error = %v, want nil", err)
	}
	if doc.ID != "d1" || doc.Title != "T" || doc.Markdown != "# T" || doc.CollectionID != "c1" || doc.URL == "" {
		t.Fatalf("document = %+v", doc)
	}
	if !containsAll(string(proc.wrote[0]), `"method":"document"`, `"document_id":"d1"`) {
		t.Fatalf("request = %s, want the method with the document", proc.wrote[0])
	}
}

// TestGatewayRefusesABadDocument checks the document shape: a missing object and missing fields each fail.
func TestGatewayRefusesABadDocument(t *testing.T) {
	for _, result := range []string{`{"id":"d1","title":"T"}`, `null`, `5`} {
		gw, _, _ := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":`+result+`}`)
		id, err := domain.NewDocumentID("d1")
		if err != nil {
			t.Fatalf("NewDocumentID: %v", err)
		}
		if _, err := gw.Document(context.Background(), id); err == nil {
			t.Fatalf("Document accepted %s", result)
		} else if _, ok := err.(ProtocolError); !ok {
			t.Fatalf("Document error = %#v, want ProtocolError", err)
		}
	}
}

// TestGatewayReadsAndWritesOperations checks the read and write mapping: the operation and the bound arguments
// travel in the params, and the reply fields map onto a record in name order.
func TestGatewayReadsAndWritesOperations(t *testing.T) {
	gw, _, proc := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"fields":{"b":"two","a":"one"}}}`)
	rec, err := gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "d1"}))
	if err != nil {
		t.Fatalf("Read error = %v, want nil", err)
	}
	fields := rec.Fields()
	if len(fields) != 2 || fields[0].Name != "a" || fields[1].Name != "b" || fields[0].Value != "one" {
		t.Fatalf("record = %+v, want the fields in name order", fields)
	}
	if !containsAll(string(proc.wrote[0]), `"method":"read"`, `"operation":"documents.get"`, `"documentId":"d1"`) {
		t.Fatalf("request = %s, want the method with the operation and the args", proc.wrote[0])
	}
	gw, _, proc = pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"fields":{"ok":true}}}`)
	rec, err = gw.Write(context.Background(), domain.OpDocumentsArchived, domain.NewArgs(map[string]any{}))
	if err != nil {
		t.Fatalf("Write error = %v, want nil", err)
	}
	if got, found := rec.Get("ok"); !found || got != true {
		t.Fatalf("record = %+v, want the ok field", rec.Fields())
	}
	if !containsAll(string(proc.wrote[0]), `"method":"write"`) {
		t.Fatalf("request = %s, want the write method", proc.wrote[0])
	}
}

// TestGatewayReadsLists checks the items shape of a read reply: each item maps onto an object record, in order.
func TestGatewayReadsLists(t *testing.T) {
	gw, _, _ := pluginGateway(t, `{"jsonrpc":"2.0","id":1,"result":{"items":[{"b":"two","a":"one"},{"a":"three"}]}}`)
	rec, err := gw.Read(context.Background(), domain.OpDocumentsList, domain.NewArgs(map[string]any{}))
	if err != nil {
		t.Fatalf("Read error = %v, want nil", err)
	}
	if !rec.IsList() || len(rec.Items()) != 2 {
		t.Fatalf("record = %+v, want a list of two", rec)
	}
	first := rec.Items()[0].Fields()
	if len(first) != 2 || first[0].Name != "a" || first[1].Value != "two" {
		t.Fatalf("first item = %+v, want the fields in name order", first)
	}
}

// TestGatewayRefusesBadFieldShapes checks the read and write shapes: a missing or mistyped fields or items shape
// fails.
func TestGatewayRefusesBadFieldShapes(t *testing.T) {
	for _, method := range []string{"read", "write"} {
		for _, reply := range []string{
			`{"jsonrpc":"2.0","id":1,"result":null}`,
			`{"jsonrpc":"2.0","id":1,"result":{}}`,
			`{"jsonrpc":"2.0","id":1,"result":{"fields":[]}}`,
			`{"jsonrpc":"2.0","id":1,"result":{"items":{}}}`,
			`{"jsonrpc":"2.0","id":1,"result":{"items":[1]}}`,
		} {
			gw, _, _ := pluginGateway(t, reply)
			var err error
			if method == "read" {
				_, err = gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{}))
			} else {
				_, err = gw.Write(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{}))
			}
			if err == nil {
				t.Fatalf("%s accepted %s", method, reply)
			} else if _, ok := err.(ProtocolError); !ok {
				t.Fatalf("%s error = %#v, want ProtocolError", method, err)
			}
		}
	}
}

// TestGatewayPropagatesTheCallFailure checks that a pin mismatch, a protocol refusal, and a timeout surface from
// the gateway unchanged, with the codes the CLI reports.
func TestGatewayPropagatesTheCallFailure(t *testing.T) {
	ref, err := domain.NewPluginRef("/opt/plugins/wiki", pinOf(pluginBytesV2))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	id, _ := domain.NewSourceID("docs")
	endpoint, _ := domain.NewEndpoint("https://docs.example.test")
	credential, _ := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	mismatched := domain.NewSource(id, domain.KindOutline, mustTagSet(t), endpoint, credential).WithPlugin(ref)
	creds := &fakeCredentials{key: "credential-sentinel-9f2a"}
	gw, _, _, _ := gatewaySeams(t, mismatched, goodReply, creds)
	if _, err := gw.ListCollections(context.Background()); err == nil {
		t.Fatal("ListCollections accepted a pin mismatch")
	} else if _, ok := err.(PinMismatchError); !ok {
		t.Fatalf("ListCollections error = %#v, want PinMismatchError", err)
	}
	gw, _, _ = pluginGateway(t, `{"jsonrpc":"2.0","id":9,"result":{}}`)
	if _, err := gw.ListCollections(context.Background()); err == nil {
		t.Fatal("ListCollections accepted a wrong id")
	} else if _, ok := err.(ProtocolError); !ok {
		t.Fatalf("ListCollections error = %#v, want ProtocolError", err)
	}
}

// TestGatewaySurfacesCallFailuresFromEveryMethod checks that a call failure reaches the caller unchanged from every
// gateway method: the pin, the protocol, and the secrets are checked once, in call, not per method.
func TestGatewaySurfacesCallFailuresFromEveryMethod(t *testing.T) {
	ref, err := domain.NewPluginRef("/opt/plugins/wiki", pinOf(pluginBytesV2))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	id, _ := domain.NewSourceID("docs")
	endpoint, _ := domain.NewEndpoint("https://docs.example.test")
	credential, _ := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	mismatched := domain.NewSource(id, domain.KindOutline, mustTagSet(t), endpoint, credential).WithPlugin(ref)
	creds := &fakeCredentials{key: "credential-sentinel-9f2a"}
	gw, _, _, _ := gatewaySeams(t, mismatched, goodReply, creds)
	collection, _ := domain.NewCollectionID("c1")
	document, _ := domain.NewDocumentID("d1")
	args := domain.NewArgs(map[string]any{})
	calls := []struct {
		name string
		run  func() error
	}{
		{"collections", func() error { _, err := gw.ListCollections(context.Background()); return err }},
		{"tree", func() error { _, err := gw.CollectionTree(context.Background(), collection); return err }},
		{"document", func() error { _, err := gw.Document(context.Background(), document); return err }},
		{"read", func() error { _, err := gw.Read(context.Background(), domain.OpDocumentsGet, args); return err }},
		{"write", func() error { _, err := gw.Write(context.Background(), domain.OpDocumentsList, args); return err }},
	}
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			if err := c.run(); err == nil {
				t.Fatalf("%s accepted a pin mismatch", c.name)
			} else if _, ok := err.(PinMismatchError); !ok {
				t.Fatalf("%s error = %#v, want PinMismatchError", c.name, err)
			}
		})
	}
}

// TestGatewayTimesOutThroughTheCall checks that a plugin that never answers fails the gateway with the timeout the
// CLI reports as 124.
func TestGatewayTimesOutThroughTheCall(t *testing.T) {
	creds := &fakeCredentials{key: "credential-sentinel-9f2a"}
	files, starter := filesWithV1(), &fakeStarter{}
	proc := &fakeProc{blockRead: true, unblock: make(chan struct{})}
	starter.proc = proc
	gw, _, _, _ := gatewaySeams(t, mustPluginSource(t), goodReply, creds)
	gw.Seams = Seams{Files: files, Start: starter.Start, After: func(time.Duration) <-chan time.Time { return closedChan() }}
	if _, err := gw.ListCollections(context.Background()); err == nil {
		t.Fatal("ListCollections accepted a silent plugin")
	} else if _, ok := err.(TimeoutError); !ok {
		t.Fatalf("ListCollections error = %#v, want TimeoutError", err)
	}
	if proc.killed == 0 {
		t.Fatal("a silent plugin was not killed")
	}
}

// TestGatewayPassesTheIdentity checks that a source with an identity resolves its address and passes both
// variables to the plugin.
func TestGatewayPassesTheIdentity(t *testing.T) {
	creds := &fakeCredentials{key: "token-sentinel-1a", identity: "sentinel-77@example.test"}
	gw, _, starter, _ := gatewaySeams(t, mustIdentityPluginSource(t),
		`{"jsonrpc":"2.0","id":1,"result":{"collections":[]}}`, creds)
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections error = %v, want nil", err)
	}
	if len(starter.env) != 2 || starter.env[0] != "EXAMPLE_JIRA_API_TOKEN=token-sentinel-1a" ||
		starter.env[1] != "EXAMPLE_JIRA_EMAIL=sentinel-77@example.test" {
		t.Fatalf("env = %q, want the credential and the identity", starter.env)
	}
	if creds.keys != 1 || creds.idents != 1 {
		t.Fatalf("resolved %d keys and %d identities, want one of each", creds.keys, creds.idents)
	}
}

// containsAll reports whether text holds every part.
func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}

// TestGatewayImplementsTheThreePorts pins the layering: the plugin serves a source through the same ports as the
// engine, so the runtime sees one shape.
func TestGatewayImplementsTheThreePorts(t *testing.T) {
	var gateway any = &Gateway{}
	if _, ok := gateway.(ports.SourceGateway); !ok {
		t.Fatal("the gateway must implement SourceGateway")
	}
	if _, ok := gateway.(ports.SourceReader); !ok {
		t.Fatal("the gateway must implement SourceReader")
	}
	if _, ok := gateway.(ports.SourceWriter); !ok {
		t.Fatal("the gateway must implement SourceWriter")
	}
}
