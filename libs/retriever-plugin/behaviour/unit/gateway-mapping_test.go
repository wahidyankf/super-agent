package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// answering returns a builder of a fake process that answers every call with frame.
func answering(frame string) func() *fakeCommand {
	return func() *fakeCommand { return newFakeCommand(frame) }
}

// collectionTreeOf returns the ID of the collection the tree tests read.
func collectionTreeOf(t *testing.T) domain.CollectionID {
	t.Helper()
	id, err := domain.NewCollectionID("col-1")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	return id
}

// documentOf returns the ID of the document the document tests read.
func documentOf(t *testing.T) domain.DocumentID {
	t.Helper()
	id, err := domain.NewDocumentID("doc-1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	return id
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: The list of collections maps each collection onto a domain value in reply order
func TestTheListOfCollectionsMapsInReplyOrder(t *testing.T) {
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never,
		answering(okFrame(`{"collections":[{"id":"c1","name":"One"},{"id":"c2","name":"Two"}]}`)))
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	want := []domain.Collection{{ID: "c1", Name: "One"}, {ID: "c2", Name: "Two"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collections = %+v, want %+v", got, want)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A collection list reply of a bad shape fails with a protocol error
func TestACollectionListReplyOfABadShapeFailsWithAProtocolError(t *testing.T) {
	cases := []struct {
		name  string
		reply string
	}{
		{"a result that is not a list", `{"collections":"x"}`},
		{"a collection that is not an object", `{"collections":["x"]}`},
		{"a collection with a field of the wrong type", `{"collections":[{"id":1,"name":"One"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(tc.reply)))
			_, err := gw.ListCollections(context.Background())
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: The collection tree maps a null parent to a root and maps each depth onto a number
func TestTheCollectionTreeMapsANullParentToARootAndEachDepthToANumber(t *testing.T) {
	reply := `{"nodes":[` +
		`{"id":"n1","title":"Root","url":"/r","parent_id":null,"depth":0},` +
		`{"id":"n2","title":"Child","url":"/c","parent_id":"n1","depth":1}]}`
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(reply)))
	nodes, err := gw.CollectionTree(context.Background(), collectionTreeOf(t))
	if err != nil {
		t.Fatalf("CollectionTree error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(nodes))
	}
	if nodes[0].ParentID != nil {
		t.Fatalf("root parent = %q, want none", *nodes[0].ParentID)
	}
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "n1" {
		t.Fatalf("child parent = %v, want n1", nodes[1].ParentID)
	}
	if nodes[0].Depth != 0 || nodes[1].Depth != 1 {
		t.Fatalf("depths = %d and %d, want 0 and 1", nodes[0].Depth, nodes[1].Depth)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A collection tree reply of a bad shape fails with a protocol error
func TestACollectionTreeReplyOfABadShapeFailsWithAProtocolError(t *testing.T) {
	cases := []struct {
		name  string
		reply string
	}{
		{"a node that is not an object", `{"nodes":["x"]}`},
		{"a node with a missing or mistyped field", `{"nodes":[{"id":"n1","title":"T","url":"/u"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(tc.reply)))
			_, err := gw.CollectionTree(context.Background(), collectionTreeOf(t))
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A document reply maps onto the domain document
func TestADocumentReplyMapsOntoTheDomainDocument(t *testing.T) {
	reply := `{"id":"doc-1","title":"Notes","url":"/n","markdown":"# Notes","collection_id":"col-1"}`
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(reply)))
	got, err := gw.Document(context.Background(), documentOf(t))
	if err != nil {
		t.Fatalf("Document error = %v", err)
	}
	want := domain.SourceDocument{ID: "doc-1", Title: "Notes", URL: "/n", Markdown: "# Notes", CollectionID: "col-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("document = %+v, want %+v", got, want)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A document reply of a bad shape fails with a protocol error
func TestADocumentReplyOfABadShapeFailsWithAProtocolError(t *testing.T) {
	cases := []struct {
		name  string
		reply string
	}{
		{"a missing object", "null"},
		{"a missing field", `{"id":"doc-1","title":"Notes","url":"/n","markdown":"# Notes"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(tc.reply)))
			_, err := gw.Document(context.Background(), documentOf(t))
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A read or write sends its operation and bound arguments in the request and maps the reply fields in name order
func TestAReadOrWriteSendsItsOperationAndMapsFieldsInNameOrder(t *testing.T) {
	gw, started := gatewayOver(t, false, newMemFiles(pluginBody), never,
		answering(okFrame(`{"fields":{"zeta":1,"alpha":"a"}}`)))
	args := domain.NewArgs(map[string]any{"documentId": "doc-1"})
	op := domain.OpDocumentsGet
	for _, run := range []struct {
		method string
		call   func() (domain.Record, error)
	}{
		{"read", func() (domain.Record, error) { return gw.Read(context.Background(), op, args) }},
		{"write", func() (domain.Record, error) { return gw.Write(context.Background(), op, args) }},
	} {
		rec, err := run.call()
		if err != nil {
			t.Fatalf("%s error = %v", run.method, err)
		}
		var names []string
		for _, f := range rec.Fields() {
			names = append(names, f.Name)
		}
		if !reflect.DeepEqual(names, []string{"alpha", "zeta"}) {
			t.Fatalf("%s fields = %q, want name order", run.method, names)
		}
	}
	if len(*started) != 2 {
		t.Fatalf("processes = %d, want one for the read and one for the write", len(*started))
	}
	for i, method := range []string{"read", "write"} {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Source    string         `json:"source"`
				Operation string         `json:"operation"`
				Args      map[string]any `json:"args"`
			} `json:"params"`
		}
		if err := json.Unmarshal((*started)[i].sentFrames()[0], &req); err != nil {
			t.Fatalf("%s request is not JSON: %v", method, err)
		}
		if req.Method != method || req.Params.Operation != "documents.get" || req.Params.Args["documentId"] != "doc-1" {
			t.Fatalf("%s request = %+v, want the operation and the bound arguments", method, req)
		}
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A read reply of items maps each item onto an object record in order
func TestAReadReplyOfItemsMapsEachItemInOrder(t *testing.T) {
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never,
		answering(okFrame(`{"items":[{"name":"a"},{"name":"b"}]}`)))
	rec, err := gw.Read(context.Background(), domain.OpDocumentsList, domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if !rec.IsList() || len(rec.Items()) != 2 {
		t.Fatalf("record = %+v, want a list of two items", rec)
	}
	first, _ := rec.Items()[0].Get("name")
	second, _ := rec.Items()[1].Get("name")
	if first != "a" || second != "b" {
		t.Fatalf("items = %v and %v, want a then b", first, second)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A read or write reply whose fields or items have the wrong shape fails with a protocol error
func TestAReadOrWriteReplyOfTheWrongShapeFailsWithAProtocolError(t *testing.T) {
	cases := []struct {
		name  string
		reply string
	}{
		{"no fields object", `{}`},
		{"a fields object of the wrong type", `{"fields":"x"}`},
		{"no items list", `{"items":null}`},
		{"an items value that is not a list", `{"items":"x"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never, answering(okFrame(tc.reply)))
			_, err := gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(nil))
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A failure of the plugin call reaches the caller unchanged from every gateway method
func TestAFailureOfThePluginCallReachesTheCallerUnchangedFromEveryMethod(t *testing.T) {
	cases := []struct {
		name  string
		files *memFiles
		after func(time.Duration) <-chan time.Time
		build func() *fakeCommand
		check func(error) bool
	}{
		{"pin mismatch", newMemFiles([]byte("#!/bin/sh\necho v2\n")), never, answering(""), func(err error) bool {
			var pin plugin.PinMismatchError
			return errors.As(err, &pin)
		}},
		{"protocol", newMemFiles(pluginBody), never, answering(""), func(err error) bool {
			var protocol plugin.ProtocolError
			return errors.As(err, &protocol)
		}},
		{"timeout", newMemFiles(pluginBody), fired, func() *fakeCommand {
			cmd := newFakeCommand("")
			cmd.holdStdout = true
			return cmd
		}, func(err error) bool {
			var timeout plugin.TimeoutError
			return errors.As(err, &timeout)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, _ := gatewayOver(t, false, tc.files, tc.after, tc.build)
			ctx := context.Background()
			calls := map[string]func() error{
				"ListCollections": func() error { _, err := gw.ListCollections(ctx); return err },
				"CollectionTree":  func() error { _, err := gw.CollectionTree(ctx, collectionTreeOf(t)); return err },
				"Document":        func() error { _, err := gw.Document(ctx, documentOf(t)); return err },
				"Read": func() error {
					_, err := gw.Read(ctx, domain.OpDocumentsGet, domain.NewArgs(nil))
					return err
				},
				"Write": func() error {
					_, err := gw.Write(ctx, domain.OpDocumentsGet, domain.NewArgs(nil))
					return err
				},
			}
			for method, call := range calls {
				if err := call(); !tc.check(err) {
					t.Fatalf("%s error = %#v, want the %s failure unchanged", method, err, tc.name)
				}
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A timeout of the plugin call surfaces through the gateway with exit status 124
func TestATimeoutSurfacesThroughTheGatewayAsTheTimeoutSentinel(t *testing.T) {
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), fired, func() *fakeCommand {
		cmd := newFakeCommand("")
		cmd.holdStdout = true
		return cmd
	})
	_, err := gw.ListCollections(context.Background())
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("error = %#v, want the timeout that unwraps to domain.ErrTimedOut", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A source with an identity resolves its address and passes both variables to the plugin
func TestASourceWithAnIdentityPassesItsAddressAndBothVariables(t *testing.T) {
	gw, started := gatewayOver(t, true, newMemFiles(pluginBody), never, answering(okFrame(`{"collections":[]}`)))
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a", "EXAMPLE_WIKI_EMAIL=name@example.test"}
	if len(*started) != 1 || !reflect.DeepEqual((*started)[0].env, want) {
		t.Fatalf("plugin environment = %q, want the credential and the resolved address", (*started)[0].env)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: The gateway serves the same three ports as the engine
func TestTheGatewayServesTheSameThreePortsAsTheEngine(t *testing.T) {
	var gw any = &plugin.Gateway{}
	if _, ok := gw.(ports.SourceGateway); !ok {
		t.Fatal("the gateway does not serve ports.SourceGateway")
	}
	if _, ok := gw.(ports.SourceReader); !ok {
		t.Fatal("the gateway does not serve ports.SourceReader")
	}
	if _, ok := gw.(ports.SourceWriter); !ok {
		t.Fatal("the gateway does not serve ports.SourceWriter")
	}
}
