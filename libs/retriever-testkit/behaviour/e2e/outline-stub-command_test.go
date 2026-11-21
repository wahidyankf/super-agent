//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const outlinePrefix = "outline stub listening on "

// outlineCollectionInfo asks the running Outline stub for one collection and returns its name.
func outlineCollectionInfo(t *testing.T, client *http.Client, base, id string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"id": id})
	if err != nil {
		t.Fatalf("encode the body: %v", err)
	}
	resp, err := client.Post(base+"/api/collections.info", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("call the stub: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the reply: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the stub answered %d for %s: %s", resp.StatusCode, id, data)
	}
	var reply struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		t.Fatalf("the reply is not JSON: %v", err)
	}
	return reply.Data.Name
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command refuses to start without a certificate file path
func TestOutlineCommandRefusesToStartWithoutCAFile(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	res := runCommand(t, nil, bin)
	if res.code != 1 || !strings.Contains(res.stderr, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q; want 1 with the -ca-file message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command refuses both status faults at once
func TestOutlineCommandRefusesBothStatusFaultsAtOnce(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	res := runCommand(t, nil, bin, "-ca-file", ca, "-list-status", "500", "-document-status", "503")
	if res.code != 1 || !strings.Contains(res.stderr, "only one of -list-status and -document-status can be set") {
		t.Fatalf("status %d and stderr %q; want 1 with the two-faults message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: A collection flag is read as id=name in the order given
func TestOutlineCollectionFlagIsReadAsIDNameInOrderGiven(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"c1=Eng", "c1=Eng", "Eng"},
		{"c1=", "c1=", "-collection wants id=name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ca := filepath.Join(t.TempDir(), "ca.pem")
			if c.value != "c1=Eng" {
				res := runCommand(t, nil, bin, "-ca-file", ca, "-collection", c.value)
				if res.code != 1 || !strings.Contains(res.stderr, c.want) {
					t.Fatalf("status %d and stderr %q; want 1 with an error that holds %q", res.code, res.stderr, c.want)
				}
				return
			}
			r := startCommand(t, nil, bin, "-ca-file", ca, "-collection", c.value)
			defer r.stop(t)
			base := announcedURL(t, r.announce, outlinePrefix)
			if got := outlineCollectionInfo(t, tlsClient(t, ca), base, "c1"); got != c.want {
				t.Fatalf("the collection c1 is named %q, want %q", got, c.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command announces its loopback address after it writes the certificate
func TestOutlineCommandAnnouncesLoopbackAddressAfterWritingCertificate(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommand(t, nil, bin, "-ca-file", ca)
	defer r.stop(t)
	base := announcedURL(t, r.announce, outlinePrefix)
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

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command stops answering when its context ends
func TestOutlineCommandStopsAnsweringWhenContextEnds(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommand(t, nil, bin, "-ca-file", ca)
	base := announcedURL(t, r.announce, outlinePrefix)
	client := tlsClient(t, ca)
	before, err := client.Post(base+"/api/collections.list", "application/json", strings.NewReader(`{"offset":0,"limit":1}`))
	if err != nil {
		t.Fatalf("call the stub before the stop: %v", err)
	}
	before.Body.Close()
	if before.StatusCode != http.StatusOK {
		t.Fatalf("the stub answered %d before the stop, want 200", before.StatusCode)
	}
	if code := r.stop(t); code != 0 {
		t.Fatalf("the command returned %d, want 0", code)
	}
	resp, err := client.Post(base+"/api/collections.info", "application/json", strings.NewReader("{}"))
	if err == nil {
		resp.Body.Close()
		t.Fatalf("the stub still answers at %s after its context ended", base)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: A documents file that is not JSON is refused with its parse error
func TestOutlineDocumentsFileThatIsNotJSONIsRefused(t *testing.T) {
	bin := commandBinary(t, "outlinestubcmd")
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	documents := filepath.Join(dir, "documents.json")
	if err := os.WriteFile(documents, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write the documents file: %v", err)
	}
	res := runCommand(t, nil, bin, "-ca-file", ca, "-documents-file", documents)
	if res.code != 1 || !strings.HasPrefix(res.stderr, "parse the documents file") {
		t.Fatalf("status %d and stderr %q; want 1 with an error that starts with the parse message", res.code, res.stderr)
	}
}
