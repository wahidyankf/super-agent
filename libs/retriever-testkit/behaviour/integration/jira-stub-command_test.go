//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
	jiracmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/jira"
)

// jiraCommandHost returns the real host of the Jira command with the signal wait the test passes. Its files, clock, TLS
// server, and record file are real.
func jiraCommandHost(notify commandNotify) jiracmd.Host {
	h := jiracmd.Real
	h.Notify = notify
	return h
}

// jiraBasicHeader returns the HTTP Basic header of an account and a token.
func jiraBasicHeader(email, token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token))
}

// jiraCallAs sends one GET of jiraMyself to base with the Basic header of the account, through client, and returns the status.
func jiraCallAs(t *testing.T, client *http.Client, base, email, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+jiraMyself, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", jiraBasicHeader(email, token))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", jiraMyself, err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file supplies the account and the token in place of the flags
func TestJiraCommandConfigFileSuppliesTheAccountOverRealTLS(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "stub.json")
	data, err := json.Marshal(jirastub.Config{Email: "config@example.test", Token: "config-token"})
	if err != nil {
		t.Fatalf("marshal the config: %v", err)
	}
	if err := os.WriteFile(config, data, 0o600); err != nil {
		t.Fatalf("write the config file: %v", err)
	}
	ca := filepath.Join(dir, "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return jiracmd.Run([]string{"-ca-file", ca, "-config", config}, stdout, stderr, jiraCommandHost(notify))
	})
	base, found := strings.CutPrefix(r.readLine(t), "jira stub listening on ")
	if !found {
		t.Fatalf("the announcement lacks its prefix: %q", base)
	}

	if status := jiraCallAs(t, commandCAClient(t, ca), base, "config@example.test", "config-token"); status != http.StatusOK {
		t.Fatalf("the stub answered %d to the config account, want 200", status)
	}
	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file that cannot be read or parsed is refused with its error
func TestJiraCommandRefusesAConfigFileThatCannotBeReadOrParsedFromDisk(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		absent  bool
		message string
	}{
		{name: "cannot be read", absent: true, message: "read the config file"},
		{name: "is not valid JSON", content: []byte("{not json"), message: "parse the config file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "stub.json")
			if !tc.absent {
				if err := os.WriteFile(config, tc.content, 0o600); err != nil {
					t.Fatalf("write the config file: %v", err)
				}
			}
			var out, errOut bytes.Buffer
			status := jiracmd.Run([]string{"-ca-file", filepath.Join(dir, "ca.pem"), "-config", config}, &out, &errOut, jiracmd.Real)
			if status != 1 || !strings.HasPrefix(errOut.String(), tc.message) {
				t.Fatalf("status %d and stderr %q, want 1 with an error that starts with %q", status, errOut.String(), tc.message)
			}
			if out.Len() != 0 {
				t.Fatalf("the command announced %q before refusing the config", out.String())
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The record file holds one JSON line for each request and never the token
func TestJiraCommandRecordFileHoldsOneLineAndNeverTheTokenOverRealTLS(t *testing.T) {
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	record := filepath.Join(dir, "requests.jsonl")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return jiracmd.Run([]string{"-ca-file", ca, "-email", jiraEmail, "-token", jiraToken, "-record", record}, stdout, stderr, jiraCommandHost(notify))
	})
	base, _ := strings.CutPrefix(r.readLine(t), "jira stub listening on ")
	if status := jiraCallAs(t, commandCAClient(t, ca), base, jiraEmail, jiraToken); status != http.StatusOK {
		t.Fatalf("the stub answered %d to the synthetic account, want 200", status)
	}
	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}

	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("read the record file: %v", err)
	}
	if strings.Count(string(data), "\n") != 1 || !strings.Contains(string(data), `"Path":"`+jiraMyself+`"`) {
		t.Fatalf("the record = %q, want one line for the request", data)
	}
	if strings.Contains(string(data), jiraToken) {
		t.Fatal("the record file holds the token")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command writes the certificate, then announces its address
func TestJiraCommandWritesTheCertificateBeforeAnnouncingItsAddress(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return jiracmd.Run([]string{"-ca-file", ca, "-email", jiraEmail, "-token", jiraToken}, stdout, stderr, jiraCommandHost(notify))
	})
	// The certificate is written before the announcement, so the file exists by the time the line is read.
	line := r.readLine(t)
	if !strings.HasPrefix(line, "jira stub listening on https://127.0.0.1:") {
		t.Fatalf("the announcement is %q, want a loopback TLS address", line)
	}
	data, err := os.ReadFile(ca)
	if err != nil || !bytes.Contains(data, []byte("BEGIN CERTIFICATE")) {
		t.Fatalf("the certificate file = %v, want a PEM certificate by the announcement", err)
	}
	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A clean run exits 0, and a failed run prints its error and exits 1
func TestJiraCommandCleanRunExitsZeroAndFailedRunExitsOneOverRealFiles(t *testing.T) {
	dir := t.TempDir()
	clean := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return jiracmd.Run([]string{"-ca-file", filepath.Join(dir, "ca.pem"), "-email", jiraEmail, "-token", jiraToken}, stdout, stderr, jiraCommandHost(notify))
	})
	clean.readLine(t)
	if status := clean.finish(t); status != 0 {
		t.Fatalf("clean run status %d, want 0", status)
	}

	var out, errOut bytes.Buffer
	missing := filepath.Join(dir, "missing", "ca.pem")
	status := jiracmd.Run([]string{"-ca-file", missing, "-email", jiraEmail, "-token", jiraToken}, &out, &errOut, jiraCommandHost(stopAtOnceNotify))
	if status != 1 || !strings.Contains(errOut.String(), "write the CA file") {
		t.Fatalf("failed run status %d and stderr %q, want 1 with the write error", status, errOut.String())
	}
}
