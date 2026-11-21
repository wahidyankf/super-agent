//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	outlinecmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/outline"
)

// outlineCommandHost returns the real host of the Outline command with the signal wait the test passes. Its files, clock,
// TLS server, and loopback socket are real.
func outlineCommandHost(notify commandNotify) outlinecmd.Host {
	h := outlinecmd.Real
	h.Notify = notify
	return h
}

// outlineCollectionsList posts one collections.list request to base through client and returns the status.
func outlineCollectionsList(t *testing.T, client *http.Client, base string) (int, error) {
	t.Helper()
	resp, err := client.Post(base+"/api/collections.list", "application/json", strings.NewReader(`{"offset":0,"limit":25}`))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode, nil
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command announces its loopback address after it writes the certificate
func TestOutlineCommandAnnouncesItsLoopbackURLAfterWritingTheCertificate(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return outlinecmd.Run([]string{"-ca-file", ca}, stdout, stderr, outlineCommandHost(notify))
	})
	line := r.readLine(t)
	if !strings.HasPrefix(line, "outline stub listening on https://127.0.0.1:") {
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

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command stops answering when its context ends
func TestOutlineCommandStopsAnsweringOnItsLoopbackPortWhenItsContextEnds(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return outlinecmd.Run([]string{"-ca-file", ca}, stdout, stderr, outlineCommandHost(notify))
	})
	base, found := strings.CutPrefix(r.readLine(t), "outline stub listening on ")
	if !found {
		t.Fatal("the announcement lacks its prefix")
	}
	client := commandCAClient(t, ca)
	if status, err := outlineCollectionsList(t, client, base); err != nil || status != http.StatusOK {
		t.Fatalf("collections.list before the stop = %d, %v; want 200", status, err)
	}

	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
	if status, err := outlineCollectionsList(t, client, base); err == nil {
		t.Fatalf("the stub still answered %d at %s after its context ended", status, base)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: A documents file that is not JSON is refused with its parse error
func TestOutlineCommandRefusesADocumentsFileThatIsNotJSONFromDisk(t *testing.T) {
	dir := t.TempDir()
	documents := filepath.Join(dir, "documents.json")
	if err := os.WriteFile(documents, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write the documents file: %v", err)
	}
	var out, errOut bytes.Buffer
	status := outlinecmd.Run([]string{"-ca-file", filepath.Join(dir, "ca.pem"), "-documents-file", documents}, &out, &errOut, outlineCommandHost(stopAtOnceNotify))
	if status != 1 || !strings.HasPrefix(errOut.String(), "parse the documents file") {
		t.Fatalf("status %d and stderr %q, want 1 with an error that starts with the parse message", status, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("the command announced %q before refusing the documents file", out.String())
	}
}
