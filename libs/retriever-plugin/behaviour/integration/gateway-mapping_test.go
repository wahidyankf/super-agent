//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A read or write sends its operation and bound arguments in the request and maps the reply fields in name order
func TestARealReadOrWriteSendsItsRequestAndMapsFieldsInNameOrder(t *testing.T) {
	dir := t.TempDir()
	requestFile := filepath.Join(dir, "request.json")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\ncat > '"+requestFile+"'\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"fields\":{\"zeta\":1,\"alpha\":\"a\"}}}'\n")
	gw := gatewayFor(t, script, false, &fixedCredentials{})
	args := domain.NewArgs(map[string]any{"documentId": "doc-1"})
	for _, method := range []string{"read", "write"} {
		var rec domain.Record
		var err error
		if method == "read" {
			rec, err = gw.Read(context.Background(), domain.OpDocumentsGet, args)
		} else {
			rec, err = gw.Write(context.Background(), domain.OpDocumentsGet, args)
		}
		if err != nil {
			t.Fatalf("%s error = %v", method, err)
		}
		var names []string
		for _, f := range rec.Fields() {
			names = append(names, f.Name)
		}
		if !reflect.DeepEqual(names, []string{"alpha", "zeta"}) {
			t.Fatalf("%s fields = %q, want name order", method, names)
		}
		data, err := os.ReadFile(requestFile)
		if err != nil {
			t.Fatalf("read the request the plugin received: %v", err)
		}
		var req struct {
			Method string `json:"method"`
			Params struct {
				Operation string         `json:"operation"`
				Args      map[string]any `json:"args"`
			} `json:"params"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			t.Fatalf("request is not JSON: %v", err)
		}
		if req.Method != method || req.Params.Operation != "documents.get" || req.Params.Args["documentId"] != "doc-1" {
			t.Fatalf("%s request = %+v, want the operation and the bound arguments", method, req)
		}
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A failure of the plugin call reaches the caller unchanged from every gateway method
func TestAFailureOfARealPluginReachesTheCallerUnchangedFromEveryMethod(t *testing.T) {
	dir := t.TempDir()
	silent := writeScript(t, dir, "silent", "#!/bin/sh\nsleep 300\n")
	exitsAt127 := writeScript(t, dir, "dies", "#!/bin/sh\nexit 127\n")
	wrongPin, err := domain.NewPluginRef(helperPath, strings.Repeat("0", 64))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	cases := []struct {
		name  string
		setup func() *plugin.Gateway
		check func(error) bool
	}{
		{"pin mismatch", func() *plugin.Gateway {
			creds := &fixedCredentials{}
			return &plugin.Gateway{Source: wikiSourceWithRef(t, wrongPin, false), Credentials: creds, Identities: creds, Timeout: 10 * time.Second}
		}, func(err error) bool {
			var pin plugin.PinMismatchError
			return errors.As(err, &pin)
		}},
		{"protocol", func() *plugin.Gateway { return gatewayFor(t, exitsAt127, false, &fixedCredentials{}) }, func(err error) bool {
			var protocol plugin.ProtocolError
			return errors.As(err, &protocol)
		}},
		{"timeout", func() *plugin.Gateway {
			gw := gatewayFor(t, silent, false, &fixedCredentials{})
			gw.Timeout = 300 * time.Millisecond
			return gw
		}, func(err error) bool {
			var timeout plugin.TimeoutError
			return errors.As(err, &timeout)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := tc.setup()
			ctx := context.Background()
			doc, err := domain.NewDocumentID("doc-1")
			if err != nil {
				t.Fatalf("NewDocumentID: %v", err)
			}
			col, err := domain.NewCollectionID("col-1")
			if err != nil {
				t.Fatalf("NewCollectionID: %v", err)
			}
			args := domain.NewArgs(map[string]any{"documentId": "doc-1"})
			calls := map[string]func() error{
				"ListCollections": func() error { _, err := gw.ListCollections(ctx); return err },
				"CollectionTree":  func() error { _, err := gw.CollectionTree(ctx, col); return err },
				"Document":        func() error { _, err := gw.Document(ctx, doc); return err },
				"Read":            func() error { _, err := gw.Read(ctx, domain.OpDocumentsGet, args); return err },
				"Write":           func() error { _, err := gw.Write(ctx, domain.OpDocumentsGet, args); return err },
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
func TestATimeoutOfARealPluginSurfacesThroughTheGateway(t *testing.T) {
	dir := t.TempDir()
	silent := writeScript(t, dir, "silent", "#!/bin/sh\nsleep 300\n")
	gw := gatewayFor(t, silent, false, &fixedCredentials{})
	gw.Timeout = 300 * time.Millisecond
	_, err := gw.ListCollections(context.Background())
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("error = %#v, want the timeout that unwraps to domain.ErrTimedOut", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/gateway-mapping.feature: A source with an identity resolves its address and passes both variables to the plugin
func TestASourceWithAnIdentityPassesBothVariablesThroughTheGateway(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.txt")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\nread line\nenv > '"+envFile+"'\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"collections\":[]}}'\n")
	gw := gatewayFor(t, script, true, &fixedCredentials{})
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("read the environment the plugin received: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, want := range []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a", "EXAMPLE_WIKI_EMAIL=name@example.test"} {
		if !contains(lines, want) {
			t.Fatalf("plugin environment %q lacks %q", lines, want)
		}
	}
}

// contains reports whether list holds want.
func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
