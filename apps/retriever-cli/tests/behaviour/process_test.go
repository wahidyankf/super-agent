package behaviour

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// servePrefix is the line the serve command prints once its listener is bound.
const servePrefix = "retriever serve listening on "

// replyTimeout bounds every wait on a process, so a hung process fails the scenario instead of the whole run.
const replyTimeout = 15 * time.Second

// errNoBinary is the error of a process step at a level without a built binary.
var errNoBinary = errors.New("the end-to-end level has no built retriever binary")

// serveSession is a running retriever serve process and the address it bound.
type serveSession struct {
	addr string
}

// graphqlReply is the last HTTP answer of the GraphQL endpoint.
type graphqlReply struct {
	status int
	body   string
	doc    any
}

// mcpTool is one tool the MCP server lists, with the hints the scenario checks.
type mcpTool struct {
	name        string
	readOnly    bool
	destructive bool
}

// mcpCall is the last tools/call answer.
type mcpCall struct {
	isError bool
	text    string
	size    int
}

// mcpSession is a running retriever mcp process that speaks JSON-RPC over stdio.
type mcpSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan string
	nextID int
	tools  []mcpTool
	call   *mcpCall
	exit   int
	exited bool
}

// readLines sends each line of r to out until r ends, and then closes out.
func readLines(r io.Reader, out chan<- string) {
	defer close(out)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		out <- sc.Text()
	}
}

// startServe starts retriever serve on an unused loopback port, once per scenario, and reads the address it reports.
func (w *world) startServe() error {
	if w.serve != nil {
		return nil
	}
	if w.binary == "" {
		return errNoBinary
	}
	if err := w.ensureStubs(); err != nil {
		return err
	}
	cmd := exec.Command(w.binary, "serve", "--port", "0")
	cmd.Dir = w.dir
	cmd.Env = w.envList()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start retriever serve: %w", err)
	}
	w.procKill = append(w.procKill, func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	lines := make(chan string, 16)
	go readLines(stdout, lines)
	select {
	case line, ok := <-lines:
		if !ok || !strings.HasPrefix(line, servePrefix) {
			return fmt.Errorf("retriever serve did not report a listening address, it printed %q (stderr %q)", line, stderr.String())
		}
		w.serve = &serveSession{addr: strings.TrimSpace(strings.TrimPrefix(line, servePrefix))}
		return nil
	case <-time.After(replyTimeout):
		return errors.New("retriever serve did not report a listening address in time")
	}
}

// postGraphQL sends one query to the served endpoint, with the loopback Host the server expects, and records the answer.
func (w *world) postGraphQL(query string) error {
	if err := w.startServe(); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+w.serve.addr+"/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: replyTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("query the served GraphQL endpoint: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	reply := &graphqlReply{status: resp.StatusCode, body: string(raw)}
	if doc, perr := parseJSON(string(raw)); perr == nil {
		reply.doc = doc
	}
	w.reply = reply
	return nil
}

// isLoopback reports whether an address names a loopback host, with or without a port.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// startMCP starts the MCP server a command line names and completes the initialize handshake.
func (w *world) startMCP(line string) error {
	if w.binary == "" {
		return errNoBinary
	}
	args, err := commandArgs(line)
	if err != nil {
		return err
	}
	if err := w.ensureStubs(); err != nil {
		return err
	}
	cmd := exec.Command(w.binary, args...)
	cmd.Dir = w.dir
	cmd.Env = w.envList()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start retriever mcp: %w", err)
	}
	m := &mcpSession{cmd: cmd, stdin: stdin, lines: make(chan string, 64)}
	go readLines(stdout, m.lines)
	w.mcp = m
	w.procKill = append(w.procKill, func() {
		_ = m.stdin.Close()
		_ = cmd.Process.Kill()
	})
	if _, err := m.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "behaviour-harness", "version": "0.0.0"},
	}); err != nil {
		return fmt.Errorf("initialize the MCP session: %w", err)
	}
	return m.notify("notifications/initialized")
}

// write sends one JSON-RPC message, one line each, to the server.
func (m *mcpSession) write(msg map[string]any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = m.stdin.Write(append(data, '\n'))
	return err
}

// notify sends a notification, which has no answer.
func (m *mcpSession) notify(method string) error {
	return m.write(map[string]any{"jsonrpc": "2.0", "method": method})
}

// request sends one request and returns the result of its answer. A JSON-RPC error answer is returned as an error.
func (m *mcpSession) request(method string, params any) (map[string]any, error) {
	reply, err := m.requestReply(method, params)
	if err != nil {
		return nil, err
	}
	if e, ok := reply["error"]; ok {
		return nil, fmt.Errorf("the MCP server returned an error: %v", e)
	}
	result, _ := reply["result"].(map[string]any)
	return result, nil
}

// requestReply sends one request and returns its whole answer, error object included. It skips notifications and other
// answers.
func (m *mcpSession) requestReply(method string, params any) (map[string]any, error) {
	m.nextID++
	id := m.nextID
	if err := m.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	deadline := time.After(replyTimeout)
	for {
		select {
		case line, ok := <-m.lines:
			if !ok {
				return nil, errors.New("the MCP server closed its output before it answered")
			}
			var reply map[string]any
			if err := json.Unmarshal([]byte(line), &reply); err != nil {
				return nil, fmt.Errorf("the MCP server wrote a line that is not JSON: %q", line)
			}
			rid, ok := reply["id"].(float64)
			if !ok || rid != float64(id) {
				continue
			}
			return reply, nil
		case <-deadline:
			return nil, errors.New("the MCP server did not answer in time")
		}
	}
}

// listTools asks the server for its tools and records them.
func (m *mcpSession) listTools() error {
	result, err := m.request("tools/list", map[string]any{})
	if err != nil {
		return err
	}
	raw, _ := result["tools"].([]any)
	m.tools = nil
	for _, item := range raw {
		obj, _ := item.(map[string]any)
		name, _ := obj["name"].(string)
		ann, _ := obj["annotations"].(map[string]any)
		readOnly, _ := ann["readOnlyHint"].(bool)
		destructive, _ := ann["destructiveHint"].(bool)
		m.tools = append(m.tools, mcpTool{name: name, readOnly: readOnly, destructive: destructive})
	}
	return nil
}

// callTool calls one tool and records the answer: whether it is an error, its text, and its size in bytes.
func (m *mcpSession) callTool(name string, args map[string]any) error {
	if args == nil {
		args = map[string]any{}
	}
	reply, err := m.requestReply("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return err
	}
	raw, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	// A call the server refuses at the protocol level, such as a tool above the access ceiling that it does not offer,
	// is a refused tool call. Its text is the error message, so the scenario can check that nothing else leaked.
	if e, ok := reply["error"]; ok {
		m.call = &mcpCall{isError: true, text: fmt.Sprint(e), size: len(raw)}
		return nil
	}
	result, _ := reply["result"].(map[string]any)
	isError, _ := result["isError"].(bool)
	var text strings.Builder
	if content, ok := result["content"].([]any); ok {
		for _, c := range content {
			obj, _ := c.(map[string]any)
			if t, ok := obj["text"].(string); ok {
				text.WriteString(t)
			}
		}
	}
	m.call = &mcpCall{isError: isError, text: text.String(), size: len(raw)}
	return nil
}

// finish closes the input and waits for the server to exit, and records the exit status.
func (m *mcpSession) finish() error {
	_ = m.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()
	select {
	case err := <-done:
		m.exited = true
		if err == nil {
			m.exit = 0
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			m.exit = exitErr.ExitCode()
			return nil
		}
		return err
	case <-time.After(replyTimeout):
		_ = m.cmd.Process.Kill()
		return errors.New("the MCP server did not exit after its input closed")
	}
}
