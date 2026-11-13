package behaviour

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// level names the test level a scenario runs at. The same step definitions serve every level. Only the way a command
// runs changes: in process with no source (unit), in process with the real adapters and loopback stubs (integration),
// or as the built binary (end-to-end).
type level string

const (
	levelUnit        level = "unit"
	levelIntegration level = "integration"
	levelE2E         level = "e2e"
)

// errNoSourcesAtUnit is the setup error of the unit level. It keeps every source command from reaching a gateway.
var errNoSourcesAtUnit = errors.New("no source is wired at the unit level")

// runResult is what the last command printed and returned.
type runResult struct {
	ran    bool
	exit   int
	stdout string
	stderr string
}

// world is the state of one scenario. It is created fresh for every scenario, so no state leaks between scenarios.
type world struct {
	level  level
	outer  *testing.T
	binary string // the built retriever executable; only the end-to-end level uses it

	dir    string            // the scenario's working directory and the root of its files
	cwd    string            // the process working directory before the scenario; restored at the integration level
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
	// savedEnv is the process environment before an integration scenario, restored after it. The integration level
	// makes the process environment the scenario environment, because the credentials adapter reads it directly.
	savedEnv []string
}

// newWorld returns the state of one scenario at the given level.
func newWorld(lvl level, outer *testing.T, binary string) *world {
	return &world{
		level:       lvl,
		outer:       outer,
		binary:      binary,
		env:         map[string]string{},
		inputs:      map[string]bool{},
		envFileVars: map[string]string{},
		stubs:       stubSet{seamed: map[string]bool{}},
	}
}

// setup creates the scenario directory. The home and configuration folders point inside it, so no user configuration
// reaches a scenario. At the integration level the scenario directory becomes the working directory.
func (w *world) setup() error {
	w.dir = w.outer.TempDir()
	w.env["HOME"] = filepath.Join(w.dir, "home")
	w.env["XDG_CONFIG_HOME"] = filepath.Join(w.dir, "config")
	for _, k := range proxyVars {
		w.env[k] = proxyRefusal
	}
	if w.level == levelIntegration {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		w.cwd = cwd
		w.savedEnv = os.Environ()
		return os.Chdir(w.dir)
	}
	return nil
}

// teardown stops every process and stub the scenario started, and restores the working directory and, at the
// integration level, the process environment.
func (w *world) teardown() {
	for i := len(w.procKill) - 1; i >= 0; i-- {
		w.procKill[i]()
	}
	w.stubs.close()
	if w.level == levelIntegration {
		if w.cwd != "" {
			_ = os.Chdir(w.cwd)
		}
		if w.savedEnv != nil {
			os.Clearenv()
			for _, kv := range w.savedEnv {
				k, v, _ := strings.Cut(kv, "=")
				_ = os.Setenv(k, v)
			}
		}
	}
}

// adoptProcessEnv makes the process environment exactly the scenario environment. The credentials adapter reads secrets
// with os.Getenv, so a variable the scenario does not set must be absent from the process, not inherited from the
// developer's shell.
func (w *world) adoptProcessEnv() {
	os.Clearenv()
	for k, v := range w.env {
		_ = os.Setenv(k, v)
	}
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
		Sink:   exportSink,
	}
}

// execute runs one command line, the argument vector after the program name, and records its result.
func (w *world) execute(argv []string) error {
	switch w.level {
	case levelUnit:
		return w.runUnit(argv)
	case levelIntegration:
		if err := w.ensureStubs(); err != nil {
			return err
		}
		return w.runIntegration(argv)
	default:
		if err := w.ensureStubs(); err != nil {
			return err
		}
		return w.runProcess(argv)
	}
}

// runUnit calls the CLI adapter directly with no source wired. Every source command reports the setup error, so the
// unit level can never reach a gateway, a stub, or the network.
func (w *world) runUnit(argv []string) error {
	var out, errb bytes.Buffer
	deps := w.deps(&out, &errb)
	deps.SetupErr = errNoSourcesAtUnit
	w.record(cli.Run(argv, deps), &out, &errb)
	return nil
}

// runProcess runs the built binary with exactly the scenario's environment, in the scenario's directory.
func (w *world) runProcess(argv []string) error {
	if w.binary == "" {
		return errors.New("the end-to-end level has no built retriever binary")
	}
	cmd := exec.Command(w.binary, argv...)
	cmd.Dir = w.dir
	cmd.Env = w.envList()
	cmd.Stdin = strings.NewReader("")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		return fmt.Errorf("start retriever: %w", err)
	}
	w.record(code, &out, &errb)
	return nil
}

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

// writeInput writes a file the scenario reads as an input, and records it so the secret scan skips it.
func (w *world) writeInput(name, content string) (string, error) {
	path := filepath.Join(w.dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	w.inputs[filepath.ToSlash(name)] = true
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// scanFiles returns every file under the scenario directory except the inputs the scenario wrote, keyed by the slash
// path relative to the scenario directory.
func (w *world) scanFiles() (map[string]string, error) {
	files := map[string]string{}
	err := filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(w.dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if w.inputs[rel] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = string(data)
		return nil
	})
	return files, err
}

// folderFiles returns the files directly inside an output folder, keyed by file name. A folder that does not exist
// holds no files.
func (w *world) folderFiles(folder string) (map[string]string, error) {
	dir := filepath.Join(w.dir, filepath.FromSlash(folder))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	files := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}

// regularFile reports whether path names a regular file, as the composition root checks it.
func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
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
