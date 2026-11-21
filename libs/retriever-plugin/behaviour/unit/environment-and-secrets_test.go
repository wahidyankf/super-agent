package unit_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: The plugin process receives exactly its own credential variable and nothing of the caller
func TestThePluginReceivesOnlyItsOwnCredentialVariable(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	if _, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a"}
	if !reflect.DeepEqual(cmd.env, want) {
		t.Fatalf("plugin environment = %q, want only its own credential variable", cmd.env)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A source with an identity passes its identity variable as well as its credential variable
func TestASourceWithAnIdentityPassesBothVariables(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	cfg := callConfig(t, time.Second)
	cfg.IdentityEnv = "EXAMPLE_WIKI_EMAIL"
	cfg.IdentityValue = "name@example.test"
	if _, err := plugin.Call(context.Background(), cfg, "env", nil, seamsWith(files, cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a", "EXAMPLE_WIKI_EMAIL=name@example.test"}
	if !reflect.DeepEqual(cmd.env, want) {
		t.Fatalf("plugin environment = %q, want the identity and the credential variables", cmd.env)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A source with no identity passes only its credential variable
func TestASourceWithNoIdentityPassesOnlyItsCredentialVariable(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	cfg := callConfig(t, time.Second)
	cfg.IdentityEnv = ""
	cfg.IdentityValue = "ignored@example.test"
	if _, err := plugin.Call(context.Background(), cfg, "env", nil, seamsWith(files, cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if len(cmd.env) != 1 || !strings.HasPrefix(cmd.env[0], "EXAMPLE_WIKI_API_KEY=") {
		t.Fatalf("plugin environment = %q, want only the credential variable", cmd.env)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: The host reads the source credential for each call
func TestTheHostReadsTheCredentialForEachCall(t *testing.T) {
	creds := &fakeCredentials{apiKey: "credential-sentinel-9f2a"}
	// Each call starts its own process, so the seam builds a fresh command for every call.
	seams := plugin.Seams{
		Files: newMemFiles(pluginBody),
		NewCommand: func(string, string, []string) plugin.Command {
			return newFakeCommand(okFrame(`{"collections":[]}`))
		},
		After: never,
	}
	gw := &plugin.Gateway{
		Source:      wikiSource(t, true, false),
		Credentials: creds,
		Identities:  creds,
		Seams:       seams,
		Timeout:     time.Second,
	}
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
func TestAMissingCredentialOrABadIdentityFailsBeforeAnyProcessStarts(t *testing.T) {
	cases := []struct {
		name  string
		creds *fakeCredentials
		check func(error) bool
	}{
		{"missing credential", &fakeCredentials{apiErr: domain.MissingKeyError{Variable: "EXAMPLE_WIKI_API_KEY"}}, func(err error) bool {
			var missing domain.MissingKeyError
			return errors.As(err, &missing)
		}},
		{"bad identity", &fakeCredentials{apiKey: "credential-sentinel-9f2a", identErr: domain.InvalidIdentityError{Variable: "EXAMPLE_WIKI_EMAIL"}}, func(err error) bool {
			var bad domain.InvalidIdentityError
			return errors.As(err, &bad)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newFakeCommand(okFrame(`{"collections":[]}`))
			gw := &plugin.Gateway{
				Source:      wikiSource(t, true, tc.name == "bad identity"),
				Credentials: tc.creds,
				Identities:  tc.creds,
				Seams:       seamsWith(newMemFiles(pluginBody), cmd, never),
				Timeout:     time.Second,
			}
			_, err := gw.ListCollections(context.Background())
			if !tc.check(err) {
				t.Fatalf("error = %#v, want its own error for the %s", err, tc.name)
			}
			if cmd.builds != 0 {
				t.Fatalf("a process was built %d time(s) after the %s", cmd.builds, tc.name)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: A gateway for a source with no plugin is refused before any secret is read
func TestAGatewayForASourceWithNoPluginIsRefusedBeforeASecretIsRead(t *testing.T) {
	creds := &fakeCredentials{apiKey: "credential-sentinel-9f2a"}
	cmd := newFakeCommand(okFrame(`{"collections":[]}`))
	gw := &plugin.Gateway{
		Source:      wikiSource(t, false, false),
		Credentials: creds,
		Identities:  creds,
		Seams:       seamsWith(newMemFiles(pluginBody), cmd, never),
		Timeout:     time.Second,
	}
	_, err := gw.ListCollections(context.Background())
	var start plugin.StartError
	if !errors.As(err, &start) {
		t.Fatalf("error = %#v, want the refusal of a source with no plugin", err)
	}
	if creds.apiReads != 0 || cmd.builds != 0 {
		t.Fatalf("a secret was read %d time(s) or a process built %d time(s)", creds.apiReads, cmd.builds)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: Every occurrence of a secret in a plugin reply or error reads as the redaction marker
func TestEveryOccurrenceOfASecretReadsAsTheRedactionMarker(t *testing.T) {
	const secret = "credential-sentinel-9f2a"
	reply := okFrame(`{"note":"` + secret + " and " + secret + `","other":"keep me"}`)
	cmd := newFakeCommand(reply)
	cmd.stderr = strings.NewReader("error: " + secret + " failed twice with " + secret + "\n")
	got, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	want := `{"note":"[REDACTED] and [REDACTED]","other":"keep me"}`
	if string(got.Result) != want {
		t.Fatalf("result = %s, want %s", got.Result, want)
	}
	if got.Stderr != "error: [REDACTED] failed twice with [REDACTED]\n" {
		t.Fatalf("stderr = %q, want every occurrence redacted", got.Stderr)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/environment-and-secrets.feature: An empty secret changes no text when redacted
func TestAnEmptySecretChangesNoText(t *testing.T) {
	const body = `{"note":"unchanged text"}`
	cfg := plugin.Config{
		Plugin:          checkedRef(t),
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: "",
		Timeout:         time.Second,
	}
	got, err := plugin.Call(context.Background(), cfg, "env", nil, seamsWith(newMemFiles(pluginBody), newFakeCommand(okFrame(body)), never))
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	if string(got.Result) != body {
		t.Fatalf("result = %s, want the text unchanged", got.Result)
	}
}
