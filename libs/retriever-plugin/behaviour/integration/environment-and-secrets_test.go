//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// envOf returns the environment entries the fixture plugin reported in its reply.
func envOf(t *testing.T, result json.RawMessage) []string {
	t.Helper()
	var decoded struct {
		Env []string `json:"env"`
	}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("decode the env reply: %v", err)
	}
	return decoded.Env
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: The plugin process receives exactly its own credential variable and nothing of the caller
func TestTheRealPluginReceivesOnlyItsOwnCredentialVariable(t *testing.T) {
	t.Setenv("PARENT_SECRET_API_KEY", "parent-secret-3d0f")
	reply, err := plugin.Call(context.Background(), helperConfig(t, 10*time.Second), "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	// The reply is redacted, so the marker shows the secret reached the plugin and the exact names show nothing else did.
	want := []string{"EXAMPLE_WIKI_API_KEY=[REDACTED]"}
	if got := envOf(t, reply.Result); !reflect.DeepEqual(got, want) {
		t.Fatalf("plugin environment = %q, want only its own credential variable", got)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A source with an identity passes its identity variable as well as its credential variable
func TestARealSourceWithAnIdentityPassesBothVariables(t *testing.T) {
	cfg := helperConfig(t, 10*time.Second)
	cfg.IdentityEnv = "EXAMPLE_WIKI_EMAIL"
	cfg.IdentityValue = "name@example.test"
	reply, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=[REDACTED]", "EXAMPLE_WIKI_EMAIL=[REDACTED]"}
	if got := envOf(t, reply.Result); !reflect.DeepEqual(got, want) {
		t.Fatalf("plugin environment = %q, want its identity and credential variables", got)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A source with no identity passes only its credential variable
func TestARealSourceWithNoIdentityPassesOnlyItsCredentialVariable(t *testing.T) {
	cfg := helperConfig(t, 10*time.Second)
	cfg.IdentityEnv = ""
	cfg.IdentityValue = ""
	reply, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=[REDACTED]"}
	if got := envOf(t, reply.Result); !reflect.DeepEqual(got, want) {
		t.Fatalf("plugin environment = %q, want only the credential variable", got)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: The host reads the source credential for each call
func TestTheHostReadsTheCredentialOncePerRealCall(t *testing.T) {
	creds := &fixedCredentials{}
	gw := gatewayFor(t, helperPath, false, creds)
	for i := 0; i < 2; i++ {
		if _, err := gw.ListCollections(context.Background()); err != nil {
			t.Fatalf("call %d: ListCollections error = %v", i+1, err)
		}
	}
	if creds.apiReads != 2 {
		t.Fatalf("credential reads = %d, want one for each of the two calls", creds.apiReads)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A missing credential or a bad identity fails the call before any process starts
func TestAMissingCredentialOrBadIdentityStartsNoRealProcess(t *testing.T) {
	cases := []struct {
		name     string
		creds    *fixedCredentials
		identity bool
		check    func(error) bool
	}{
		{"missing credential", &fixedCredentials{apiErr: domain.MissingKeyError{Variable: "EXAMPLE_WIKI_API_KEY"}}, false,
			func(err error) bool {
				var missing domain.MissingKeyError
				return errors.As(err, &missing)
			}},
		{"bad identity", &fixedCredentials{identErr: domain.InvalidIdentityError{Variable: "EXAMPLE_WIKI_EMAIL"}}, true,
			func(err error) bool {
				var bad domain.InvalidIdentityError
				return errors.As(err, &bad)
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "started")
			script := writeScript(t, dir, "plugin", "#!/bin/sh\ntouch '"+marker+"'\n")
			gw := gatewayFor(t, script, tc.identity, tc.creds)
			_, err := gw.ListCollections(context.Background())
			if !tc.check(err) {
				t.Fatalf("error = %#v, want its own error for the %s", err, tc.name)
			}
			if _, statErr := os.Stat(marker); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("the plugin ran after the %s", tc.name)
			}
		})
	}
}
