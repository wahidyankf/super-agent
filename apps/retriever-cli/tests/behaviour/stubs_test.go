package behaviour

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/jira"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/kiwi"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/outline"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/jirastub"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/synthetic"
)

// Names of the environment variables that hold the synthetic keys the stubs check.
const (
	kiwiKeyVar   = "EXAMPLE_KIWI_API_KEY"
	jiraEmailVar = "EXAMPLE_JIRA_EMAIL"
	jiraTokenVar = "EXAMPLE_JIRA_API_TOKEN"
)

// stubSet holds the three loopback stub servers of one scenario, and the Outline dataset they serve. Each stub starts
// when a step first needs it. The seam variables of every kind point at its stub, so no command reaches a real host.
type stubSet struct {
	outline     *outlinestub.Stub
	collections []outlinestub.Collection
	documents   []outlinestub.Document

	kiwi *kiwistub.Stub

	jira           *jirastub.Stub
	jiraSrv        *httptest.Server
	jiraConfigured bool

	// seamed records which kinds already have their seam variables written.
	seamed map[string]bool
}

// close stops every stub the scenario started. Closing a stub twice is harmless.
func (s *stubSet) close() {
	if s.jiraSrv != nil {
		s.jiraSrv.Close()
	}
	if s.kiwi != nil {
		s.kiwi.Close()
	}
	if s.outline != nil {
		s.outline.Close()
	}
}

// datasetMethods are the methods the Outline stub answers from its dataset. A synthetic reply never replaces them.
var datasetMethods = map[string]bool{
	"auth.info":             true,
	"collections.list":      true,
	"collections.info":      true,
	"collections.documents": true,
	"documents.info":        true,
	"documents.export":      true,
	"documents.list":        true,
	"documents.search":      true,
}

// outlineStub returns the Outline stub, starting it on first use with the synthetic replies loaded.
func (w *world) outlineStub() *outlinestub.Stub {
	if w.stubs.outline == nil {
		s := outlinestub.Start(w.outer, w.stubClockOutline()...)
		if err := loadSyntheticReplies(s); err != nil {
			w.outer.Fatalf("load the synthetic Outline replies: %v", err)
		}
		w.stubs.outline = s
	}
	return w.stubs.outline
}

// loadSyntheticReplies gives the stub one fixed reply for each synthetic method that the stub does not answer from its
// dataset.
func loadSyntheticReplies(s *outlinestub.Stub) error {
	entries, err := fs.ReadDir(synthetic.Replies(), ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		method := strings.TrimSuffix(e.Name(), ".json")
		if datasetMethods[method] {
			continue
		}
		raw, err := fs.ReadFile(synthetic.Replies(), e.Name())
		if err != nil {
			return err
		}
		s.SetReply(method, http.StatusOK, string(raw))
	}
	return nil
}

// kiwiStub returns the Kiwi stub, starting it on first use. A scenario can start the stub before its Background sets the
// key, so the stub accepts any Bearer value. Each scenario still checks the exact header, through the digest that the stub
// keeps for every request.
func (w *world) kiwiStub() (*kiwistub.Stub, error) {
	if w.stubs.kiwi == nil {
		opts := []kiwistub.Option{kiwistub.WithAnyBearer()}
		if w.inProcess() {
			opts = append(opts, kiwistub.WithClock(w.clock.Now))
		}
		s := kiwistub.New(w.env[kiwiKeyVar], opts...)
		if err := s.Start(); err != nil {
			return nil, fmt.Errorf("start the Kiwi stub: %w", err)
		}
		w.stubs.kiwi = s
	}
	return w.stubs.kiwi, nil
}

// jiraStub returns the Jira stub, starting its TLS server on first use.
func (w *world) jiraStub() (*jirastub.Stub, error) {
	if w.stubs.jira == nil {
		s := jirastub.New("", "")
		if w.inProcess() {
			s.SetClock(w.clock.Now)
		}
		srv, err := jirastub.NewTLSServer(s)
		if err != nil {
			return nil, fmt.Errorf("start the Jira stub: %w", err)
		}
		w.stubs.jira = s
		w.stubs.jiraSrv = srv
	}
	return w.stubs.jira, nil
}

// syncOutline gives the Outline stub the dataset the scenario holds.
func (w *world) syncOutline() {
	w.outlineStub().SetDataset(w.stubs.collections, w.stubs.documents)
}

// startNamedStubs starts the stub of each kind that a step of the scenario names, before the first command runs, so that
// the seam of the kind points at the stub. A step that only asserts that a stub received no requests does not name it,
// because a scenario that expects a refusal before any request must keep the endpoint in force.
func (w *world) startNamedStubs(texts []string) error {
	for _, text := range texts {
		if strings.Contains(text, "server received no requests") {
			continue
		}
		if strings.Contains(text, "the stub Outline server") {
			w.outlineStub()
		}
		if strings.Contains(text, "the stub Kiwi server") {
			if _, err := w.kiwiStub(); err != nil {
				return err
			}
		}
		if strings.Contains(text, "the stub Jira server") {
			if _, err := w.jiraStub(); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureStubs points the seam of every kind that a scenario started at its stub, once per kind. A kind whose stub was
// never started keeps no seam, so its endpoint is used, and the proxy variables of the scenario refuse that request
// without reaching a real host. The Jira credentials come from the scenario environment unless a configuration step set
// them.
func (w *world) ensureStubs() error {
	if w.stubs.outline != nil && !w.stubs.seamed[outlineKind] {
		pem, err := w.stubs.outline.CAPEM()
		if err != nil {
			return fmt.Errorf("read the Outline certificate: %w", err)
		}
		path, err := w.writeInput("outline-stub-ca.pem", string(pem))
		if err != nil {
			return fmt.Errorf("write the Outline certificate: %w", err)
		}
		w.env[outline.BaseURLEnv] = w.stubs.outline.URL()
		w.env[outline.CAFileEnv] = path
		w.stubs.seamed[outlineKind] = true
	}
	if w.stubs.kiwi != nil && !w.stubs.seamed[kiwiKind] {
		path, err := w.writeInput("kiwi-ca.pem", string(w.stubs.kiwi.CAPEM()))
		if err != nil {
			return fmt.Errorf("write the Kiwi certificate: %w", err)
		}
		w.env[kiwi.BaseURLEnv] = w.stubs.kiwi.URL()
		w.env[kiwi.CAFileEnv] = path
		w.stubs.seamed[kiwiKind] = true
	}
	if w.stubs.jira != nil {
		if !w.stubs.jiraConfigured {
			w.stubs.jira.SetCredentials(w.env[jiraEmailVar], w.env[jiraTokenVar])
		}
		if !w.stubs.seamed[jiraKind] {
			pem, err := jirastub.CertificatePEM()
			if err != nil {
				return fmt.Errorf("read the Jira certificate: %w", err)
			}
			path, err := w.writeInput("jira-ca.pem", string(pem))
			if err != nil {
				return err
			}
			w.env[jira.BaseURLEnv] = w.stubs.jiraSrv.URL
			w.env[jira.CAFileEnv] = path
			w.stubs.seamed[jiraKind] = true
		}
	}
	return nil
}

// Names of the source kinds in the stub set, used as keys of the seam flags.
const (
	outlineKind = "Outline"
	kiwiKind    = "Kiwi"
	jiraKind    = "Jira"
)

// proxyRefusal is the address the proxy variables name. Nothing listens there, so a request that is not answered by a
// stub is refused at once and never reaches a real host. Loopback addresses are never proxied, so the stubs still answer.
const proxyRefusal = "http://127.0.0.1:9"

// proxyVars are the environment variables that route every outbound request through proxyRefusal.
var proxyVars = []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"}

// kiwiMethodNames maps the method names the feature files use for Kiwi to the JSON-RPC names the stub records.
var kiwiMethodNames = map[string]string{
	"test-cases.create": "TestCase.create",
	"test-cases.update": "TestCase.update",
	"test-cases.filter": "TestCase.filter",
}

// kiwiMethod returns the JSON-RPC method name a feature step names.
func kiwiMethod(name string) string {
	if m, ok := kiwiMethodNames[name]; ok {
		return m
	}
	return name
}

// sha256Hex returns the SHA-256 digest of text in hex, the form the Kiwi stub keeps for an Authorization value.
func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// requireInside returns an error when the path is not inside the scenario directory.
func (w *world) requireInside(path string) error {
	rel, err := filepath.Rel(w.dir, path)
	if err != nil || rel == ".." || (len(rel) >= 3 && rel[:3] == "../") {
		return fmt.Errorf("the path %q is outside the scenario folder", path)
	}
	return nil
}

// ensureOutputFolder creates an output folder inside the scenario directory, for a step that prepares one.
func (w *world) ensureOutputFolder(folder string) (string, error) {
	path := w.absPath(folder)
	if err := w.requireInside(path); err != nil {
		return "", err
	}
	if err := w.be.mkdir(path); err != nil {
		return "", err
	}
	return path, nil
}
