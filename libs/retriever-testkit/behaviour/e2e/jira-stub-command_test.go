//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const jiraPrefix = "jira stub listening on "

// jiraMyself makes one GET of the account path of the Jira stub command, with the Basic header of the account.
func jiraMyself(t *testing.T, client *http.Client, base string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+"/rest/api/3/myself", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(jiraEmail+":"+jiraToken)))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("call the stub: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command refuses to start without a certificate file path
func TestJiraCommandRefusesToStartWithoutCAFile(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	res := runCommand(t, nil, bin, "-email", jiraEmail, "-token", jiraToken)
	if res.code != 1 || !strings.Contains(res.stderr, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q; want 1 with the -ca-file message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command refuses to start with no account, no token, and no config file
func TestJiraCommandRefusesToStartWithNoAccountTokenOrConfig(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	res := runCommand(t, nil, bin, "-ca-file", ca)
	if res.code != 1 || !strings.Contains(res.stderr, "the -email and -token flags are required, or a -config file") {
		t.Fatalf("status %d and stderr %q; want 1 with the credentials message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command reports a flag it does not know on its diagnostic output
func TestJiraCommandReportsUnknownFlagOnDiagnosticOutput(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	res := runCommand(t, nil, bin, "-no-such-flag")
	if res.code != 1 || !strings.Contains(res.stderr, "-no-such-flag") {
		t.Fatalf("status %d and stderr %q; want 1 with the flag named", res.code, res.stderr)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file supplies the account and the token in place of the flags
func TestJiraConfigFileSuppliesAccountAndToken(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, []byte(`{"Email": "`+jiraEmail+`", "Token": "`+jiraToken+`"}`), 0o600); err != nil {
		t.Fatalf("write the config file: %v", err)
	}
	r := startCommand(t, nil, bin, "-ca-file", ca, "-config", config)
	defer r.stop(t)
	base := announcedURL(t, r.announce, jiraPrefix)
	if status := jiraMyself(t, tlsClient(t, ca), base); status != http.StatusOK {
		t.Fatalf("the stub answered %d, want 200", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file that cannot be read or parsed is refused with its error
func TestJiraConfigFileThatCannotBeReadOrParsedIsRefused(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	cases := []struct {
		name    string
		problem string
		message string
	}{
		{"cannot be read", "cannot be read", "read the config file"},
		{"is not valid JSON", "is not valid JSON", "parse the config file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			ca := filepath.Join(dir, "ca.pem")
			config := filepath.Join(dir, "config.json")
			if c.problem == "is not valid JSON" {
				if err := os.WriteFile(config, []byte("{not json"), 0o600); err != nil {
					t.Fatalf("write the config file: %v", err)
				}
			}
			res := runCommand(t, nil, bin, "-ca-file", ca, "-config", config)
			if res.code != 1 || !strings.HasPrefix(res.stderr, c.message) {
				t.Fatalf("status %d and stderr %q; want 1 with an error that starts with %q", res.code, res.stderr, c.message)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The record file holds one JSON line for each request and never the token
func TestJiraRecordFileHoldsOneLinePerRequestAndNeverToken(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	record := filepath.Join(dir, "record.jsonl")
	r := startCommand(t, nil, bin, "-ca-file", ca, "-email", jiraEmail, "-token", jiraToken, "-record", record)
	base := announcedURL(t, r.announce, jiraPrefix)
	if status := jiraMyself(t, tlsClient(t, ca), base); status != http.StatusOK {
		t.Fatalf("the stub answered %d, want 200", status)
	}
	if code := r.stop(t); code != 0 {
		t.Fatalf("the command exited %d, want 0", code)
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("read the record file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("the record holds %d lines, want one for the request", len(lines))
	}
	if !json.Valid([]byte(lines[0])) {
		t.Fatalf("the record line is not JSON: %q", lines[0])
	}
	if bytes.Contains(data, []byte(jiraToken)) {
		t.Fatal("the record file holds the token")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command writes the certificate, then announces its address
func TestJiraCommandWritesCertificateThenAnnouncesAddress(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommand(t, nil, bin, "-ca-file", ca, "-email", jiraEmail, "-token", jiraToken)
	defer r.stop(t)
	base := announcedURL(t, r.announce, jiraPrefix)
	if !strings.HasPrefix(base, "https://127.0.0.1:") {
		t.Fatalf("the announced address %q is not a loopback TLS address", base)
	}
	data, err := os.ReadFile(ca)
	if err != nil {
		t.Fatalf("the certificate file is not written before the announcement: %v", err)
	}
	if !bytes.Contains(data, []byte("BEGIN CERTIFICATE")) {
		t.Fatal("the certificate file holds no PEM certificate")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A clean run exits 0, and a failed run prints its error and exits 1
func TestJiraCleanRunExitsZeroAndFailedRunExitsOne(t *testing.T) {
	bin := commandBinary(t, "jirastubcmd")
	clean := startCommand(t, nil, bin, "-ca-file", filepath.Join(t.TempDir(), "ca.pem"), "-email", jiraEmail, "-token", jiraToken)
	if code := clean.stop(t); code != 0 {
		t.Fatalf("the clean run exited %d, want 0", code)
	}
	missing := filepath.Join(t.TempDir(), "missing", "ca.pem")
	failed := runCommand(t, nil, bin, "-ca-file", missing, "-email", jiraEmail, "-token", jiraToken)
	if failed.code != 1 || !strings.Contains(failed.stderr, "no such file") {
		t.Fatalf("the failed run exited %d with stderr %q; want 1 with the error", failed.code, failed.stderr)
	}
}
