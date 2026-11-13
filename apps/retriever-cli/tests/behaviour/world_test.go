package behaviour

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// level names the test level a scenario runs at. The same step definitions serve every level. Only the place the
// scenario lives and the way a command runs change: in memory and in process (unit), on disk and in process with the
// real environment (integration), or on disk as the built binary (end-to-end).
type level string

const (
	levelUnit        level = "unit"
	levelIntegration level = "integration"
	levelE2E         level = "e2e"
)

// runResult is what the last command printed and returned.
type runResult struct {
	ran    bool
	exit   int
	stdout string
	stderr string
}

// world is the state of one scenario. It is created fresh for every scenario, so no state leaks between scenarios.
type world struct {
	level level
	outer *testing.T
	// be keeps the scenario's files, working directory, and environment, and runs its production adapters.
	be backend

	dir    string            // the scenario's working directory and the root of its files, an absolute path
	env    map[string]string // the environment the command sees, and the only one a process receives
	inputs map[string]bool   // files the scenario wrote as inputs (catalog, env file, CA certificate); never scanned
	// envFileVars holds the values of the env file the scenario wrote, so the secret scan can look for them.
	envFileVars map[string]string

	stubs    stubSet
	last     runResult
	serve    *serveSession
	mcp      *mcpSession
	reply    *graphqlReply
	procKill []func()
	// stdoutClosed is set by a step that makes standard output a pipe with no reader, so the next command meets EPIPE.
	stdoutClosed bool
}

// newWorld returns the state of one scenario at the given level, kept by the given backend.
func newWorld(lvl level, outer *testing.T, be backend) *world {
	return &world{
		level:       lvl,
		outer:       outer,
		be:          be,
		env:         map[string]string{},
		inputs:      map[string]bool{},
		envFileVars: map[string]string{},
		stubs:       stubSet{seamed: map[string]bool{}},
	}
}

// setup creates the scenario folder. The home and configuration folders point inside it, so no user configuration
// reaches a scenario.
func (w *world) setup() error {
	dir, err := w.be.start()
	if err != nil {
		return err
	}
	w.dir = dir
	w.env["HOME"] = w.absPath("home")
	w.env["XDG_CONFIG_HOME"] = w.absPath("config")
	for _, k := range proxyVars {
		w.env[k] = proxyRefusal
	}
	return nil
}

// teardown stops every process and stub the scenario started, and lets the backend restore what it changed.
func (w *world) teardown() {
	for i := len(w.procKill) - 1; i >= 0; i-- {
		w.procKill[i]()
	}
	w.stubs.close()
	w.be.stop()
}

// absPath returns the absolute path of a slash path inside the scenario folder.
func (w *world) absPath(rel string) string {
	return filepath.Join(w.dir, filepath.FromSlash(rel))
}

// inProcess reports whether the scenario runs its commands in this process, which the unit and integration levels do.
func (w *world) inProcess() bool {
	return w.be.inProcess()
}

// getenv is the environment lookup of the in-process levels. It reads only the scenario's variables.
func (w *world) getenv(name string) string {
	return w.env[name]
}

// deps returns the CLI dependencies for one in-process run. The Getenv function reads the scenario environment.
func (w *world) deps(stdout, stderr *bytes.Buffer) cli.Deps {
	return cli.Deps{
		Ports:  ports.Ports{},
		Stdin:  strings.NewReader(""),
		Stdout: stdout,
		Stderr: stderr,
		Getenv: w.getenv,
		Now:    time.Now,
		Sink:   w.be.sink,
	}
}

// execute runs one command line, the argument vector after the program name, and records its result.
func (w *world) execute(argv []string) error {
	if err := w.ensureStubs(); err != nil {
		return err
	}
	if w.inProcess() {
		return w.runIntegration(argv)
	}
	return w.be.runBinary(w, argv)
}

// brokenPipe is a standard output whose reader has closed. Every write fails with EPIPE, as a pipe with no reader does.
type brokenPipe struct{}

// Write fails with EPIPE and writes nothing.
func (brokenPipe) Write([]byte) (int, error) { return 0, syscall.EPIPE }

// record stores the result of the last command.
func (w *world) record(code int, out, errb *bytes.Buffer) {
	w.last = runResult{ran: true, exit: code, stdout: out.String(), stderr: errb.String()}
}

// lastRun returns the last command's result, or an error when no command has run in the scenario.
func (w *world) lastRun() (runResult, error) {
	if !w.last.ran {
		return runResult{}, errors.New("no command has run in this scenario")
	}
	return w.last, nil
}

// envList renders the scenario environment as KEY=VALUE pairs, sorted for a stable process environment.
func (w *world) envList() []string {
	keys := make([]string, 0, len(w.env))
	for k := range w.env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	list := make([]string, 0, len(keys))
	for _, k := range keys {
		list = append(list, k+"="+w.env[k])
	}
	return list
}

// valueOf returns the value of a secret the scenario named, from the environment or from the env file it wrote.
func (w *world) valueOf(name string) (string, error) {
	if v, ok := w.env[name]; ok && v != "" {
		return v, nil
	}
	if v, ok := w.envFileVars[name]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("the scenario holds no value for %q", name)
}

// writeInput writes a file the scenario reads as an input, and records it so the secret scan skips it. It returns the
// absolute path of the file.
func (w *world) writeInput(name, content string) (string, error) {
	path := w.absPath(name)
	w.inputs[filepath.ToSlash(name)] = true
	if err := w.be.writeFile(path, []byte(content)); err != nil {
		return "", err
	}
	return path, nil
}

// scanFiles returns every file under the scenario folder except the inputs the scenario wrote, keyed by the slash path
// relative to the scenario folder.
func (w *world) scanFiles() (map[string]string, error) {
	all, err := w.be.walk(w.dir)
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for rel, text := range all {
		if !w.inputs[rel] {
			files[rel] = text
		}
	}
	return files, nil
}

// folderFiles returns the files directly inside an output folder, keyed by file name. A folder that does not exist
// holds no files.
func (w *world) folderFiles(folder string) (map[string]string, error) {
	return w.be.dirFiles(w.absPath(folder))
}

// splitArgs splits a command line into arguments. Single and double quotes group words; there are no escapes.
func splitArgs(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inToken := false
	var quote rune
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case r == ' ' || r == '\t':
			if inToken {
				args = append(args, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, errors.New("the command line has an unterminated quote")
	}
	if inToken {
		args = append(args, cur.String())
	}
	return args, nil
}

// commandArgs turns a "retriever ..." command line into the argument vector the CLI receives.
func commandArgs(line string) ([]string, error) {
	args, err := splitArgs(line)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 || args[0] != "retriever" {
		return nil, fmt.Errorf("a command line must start with retriever: %q", line)
	}
	return args[1:], nil
}

// parseJSON decodes one JSON document. Numbers stay as json.Number, so "2" and "2.0" are told apart.
func parseJSON(text string) (any, error) {
	var doc any
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// lookupPath follows a dotted path through objects and arrays. A numeric segment indexes an array.
func lookupPath(doc any, path string) (any, bool) {
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			var idx int
			if _, err := fmt.Sscanf(seg, "%d", &idx); err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			cur = node[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// valueMatches compares a decoded JSON value with the expected text: a quoted string, true, false, a number, or absent.
func valueMatches(got any, found bool, expected string) bool {
	expected = strings.TrimSpace(expected)
	switch {
	case expected == "absent":
		return !found
	case !found:
		return false
	case len(expected) >= 2 && expected[0] == '"' && expected[len(expected)-1] == '"':
		s, ok := got.(string)
		return ok && s == expected[1:len(expected)-1]
	case expected == "true" || expected == "false":
		b, ok := got.(bool)
		return ok && fmt.Sprint(b) == expected
	default:
		n, ok := got.(json.Number)
		return ok && n.String() == expected
	}
}

// controlByte matches a byte below 0x20 other than a line break or a tab, and the DEL byte.
var controlByte = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]`)

// hasControlBytes reports whether text holds a control byte that a terminal would misread.
func hasControlBytes(text string) bool {
	return controlByte.MatchString(text)
}
