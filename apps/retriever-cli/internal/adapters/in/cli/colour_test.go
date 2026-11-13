package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// helpWithEnvironment runs the CLI in process with the given environment and returns its status and standard output.
func helpWithEnvironment(t *testing.T, env map[string]string, argv ...string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps := Deps{
		Ports:  ports.Ports{FS: stubFS{}},
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(name string) string { return env[name] },
		Now:    func() time.Time { return fixedNow },
	}
	code := Run(argv, deps)
	return code, stdout.String()
}

// TestColourOptionAndNoColorLeaveTheOutputUnchanged proves the colour contract while the CLI emits no colour. The --color
// option is accepted with auto, always, and never, and NO_COLOR is set or empty, and every combination gives the same
// standard output as the plain run, with no escape byte in it. When the CLI starts to colour output, this test must change
// with the docs, because the reference says no colour is emitted yet.
func TestColourOptionAndNoColorLeaveTheOutputUnchanged(t *testing.T) {
	baseCode, baseline := helpWithEnvironment(t, nil, "--help")
	if baseCode != 0 {
		t.Fatalf("plain --help exits %d, want 0", baseCode)
	}
	cases := []struct {
		name string
		env  map[string]string
		argv []string
	}{
		{name: "--color auto", argv: []string{"--color", "auto", "--help"}},
		{name: "--color always", argv: []string{"--color", "always", "--help"}},
		{name: "--color never", argv: []string{"--color", "never", "--help"}},
		{name: "NO_COLOR set", env: map[string]string{"NO_COLOR": "1"}, argv: []string{"--help"}},
		{name: "NO_COLOR set with --color always", env: map[string]string{"NO_COLOR": "1"}, argv: []string{"--color", "always", "--help"}},
		{name: "NO_COLOR empty", env: map[string]string{"NO_COLOR": ""}, argv: []string{"--help"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := helpWithEnvironment(t, tc.env, tc.argv...)
			if code != 0 {
				t.Fatalf("exit %d, want 0", code)
			}
			if out != baseline {
				t.Fatalf("standard output differs from the plain run")
			}
			if strings.Contains(out, "\x1b") {
				t.Fatalf("standard output holds an escape byte")
			}
		})
	}
}

// TestColourOptionRefusesAnUnknownValue proves that --color accepts only auto, always, and never.
func TestColourOptionRefusesAnUnknownValue(t *testing.T) {
	code, out := helpWithEnvironment(t, nil, "--color", "sometimes", "--help")
	if code != 2 {
		t.Fatalf("--color sometimes exits %d, want 2", code)
	}
	if out != "" {
		t.Fatalf("a refused --color writes to standard output: %q", out)
	}
}
