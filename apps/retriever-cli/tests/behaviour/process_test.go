package behaviour

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
)

// servePrefix is the line the serve command prints once its listener is bound.
const servePrefix = "retriever serve listening on "

// replyTimeout bounds every wait on a session, so a hung session fails the scenario instead of the whole run.
const replyTimeout = 15 * time.Second

// replyDeadline returns the channel that ends a wait on a session. At the unit level it is nil: the in-memory session always
// answers, so a wait ends with its answer and no timer is armed. The disk levels set it to a real timer, in the tagged file
// backend_disk_test.go, so a hung process fails its scenario instead of the whole run.
var replyDeadline = func() <-chan time.Time { return nil }

// endState is how a session ends. Its status arrives once on exited, and is kept in code after that.
type endState struct {
	exited chan int
	kill   func()
	done   bool
	code   int
}

// wait returns the exit status of the session. A session that does not end in time is killed.
func (e *endState) wait() (int, error) {
	if !e.done {
		select {
		case e.code = <-e.exited:
			e.done = true
		case <-replyDeadline():
			e.kill()
			return 0, errors.New("the server did not exit in time")
		}
	}
	return e.code, nil
}

// serveSession is a running retriever serve and the address it bound. The session runs in process at the unit and
// integration levels, and as the built binary at the end-to-end level.
type serveSession struct {
	addr   string
	signal func(os.Signal) error
	end    endState
}

// graphqlReply is the last HTTP answer of the GraphQL endpoint.
type graphqlReply struct {
	status int
	body   string
	doc    any
	header http.Header
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

// mcpSession is a running retriever mcp that speaks JSON-RPC over stdio. Like serveSession, it runs in process or as
// the built binary.
type mcpSession struct {
	stdin  io.WriteCloser
	lines  chan string
	signal func(os.Signal) error
	end    endState
	nextID int
	tools  []mcpTool
	call   *mcpCall
	// signalled is set once a signal was sent, so finish waits for the signal to end the server rather than closing the
	// input first: a closed input ends the server by EOF, which would hide the signal's status.
	signalled bool
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

// awaitServeLine reads the first line of a serve session and returns the address it names.
func awaitServeLine(lines <-chan string) (string, error) {
	select {
	case line, ok := <-lines:
		if !ok || !strings.HasPrefix(line, servePrefix) {
			return "", fmt.Errorf("retriever serve did not report a listening address, it printed %q", line)
		}
		return strings.TrimSpace(strings.TrimPrefix(line, servePrefix)), nil
	case <-replyDeadline():
		return "", errors.New("retriever serve did not report a listening address in time")
	}
}

// startServe starts retriever serve once per scenario and reads the address it reports.
func (w *world) startServe() error {
	if w.serve != nil {
		return nil
	}
	if err := w.ensureStubs(); err != nil {
		return err
	}
	var (
		s   *serveSession
		err error
	)
	if w.inProcess() {
		s, err = w.serveInProcess()
	} else {
		s, err = w.be.spawnServe(w, []string{"serve", "--port", "0"})
	}
	if err != nil {
		return err
	}
	w.serve = s
	return nil
}

// serveInProcess runs the serve command in this process. Its standard output is a pipe the scenario reads, and an
// interrupt or SIGTERM is delivered by cancelling the context the command runs under, with the signal as the cause.
func (w *world) serveInProcess() (*serveSession, error) {
	argv := []string{"serve", "--port", "0"}
	ctx, cancel := context.WithCancelCause(context.Background())
	stdoutR, stdoutW := io.Pipe()
	deps := w.protocolDeps(ctx, strings.NewReader(""), stdoutW, argv)
	s := &serveSession{
		signal: func(sig os.Signal) error {
			cancel(cli.SignalCause(sig))
			return nil
		},
		end: endState{exited: make(chan int, 1), kill: func() { cancel(nil) }},
	}
	go func() {
		s.end.exited <- cli.Run(argv, deps)
		_ = stdoutW.Close()
	}()
	lines := make(chan string, 16)
	go readLines(stdoutR, lines)
	w.procKill = append(w.procKill, s.end.kill)
	addr, err := awaitServeLine(lines)
	if err != nil {
		return nil, err
	}
	s.addr = addr
	return s, nil
}

// postGraphQL sends one query to the served endpoint, with the loopback Host the server expects, and records the answer.
func (w *world) postGraphQL(query string) error {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return err
	}
	return w.sendGraphQL(http.MethodPost, nil, body)
}

// sendGraphQL sends one request to the served endpoint with the method, the headers, and the body given, and records the
// answer. The request has a JSON Content-Type unless the headers replace it. A Host header sets the request's Host, which
// a Go client does not send as an ordinary header. A "{port}" in a header value becomes the served port.
func (w *world) sendGraphQL(method string, headers map[string]string, body []byte) error {
	if err := w.startServe(); err != nil {
		return err
	}
	req, err := http.NewRequest(method, "http://"+w.serve.addr+"/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		value = strings.ReplaceAll(value, "{port}", portOf(w.serve.addr))
		if name == "Host" {
			req.Host = value
			continue
		}
		req.Header.Set(name, value)
	}
	client := w.be.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("query the served GraphQL endpoint: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	reply := &graphqlReply{status: resp.StatusCode, body: string(raw), header: resp.Header}
	if doc, perr := parseJSON(string(raw)); perr == nil {
		reply.doc = doc
	}
	w.reply = reply
	return nil
}

// portOf returns the port of a host:port address, or the address itself when it has no port.
func portOf(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return port
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
	args, err := commandArgs(line)
	if err != nil {
		return err
	}
	if err := w.ensureStubs(); err != nil {
		return err
	}
	var m *mcpSession
	if w.inProcess() {
		m, err = w.mcpInProcess(args)
	} else {
		m, err = w.be.spawnMCP(w, args)
	}
	if err != nil {
		return err
	}
	w.mcp = m
	if _, err := m.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "behaviour-harness", "version": "0.0.0"},
	}); err != nil {
		return fmt.Errorf("initialize the MCP session: %w", err)
	}
	return m.notify("notifications/initialized")
}

// mcpInProcess runs the mcp command in this process over two in-memory pipes: the scenario writes requests to one, and
// reads the protocol answers from the other.
func (w *world) mcpInProcess(args []string) (*mcpSession, error) {
	ctx, cancel := context.WithCancelCause(context.Background())
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	deps := w.protocolDeps(ctx, inR, outW, args)
	m := &mcpSession{
		stdin: inW,
		lines: make(chan string, 64),
		signal: func(sig os.Signal) error {
			cancel(cli.SignalCause(sig))
			return nil
		},
		end: endState{
			exited: make(chan int, 1),
			kill: func() {
				_ = inW.Close()
				cancel(nil)
			},
		},
	}
	go func() {
		m.end.exited <- cli.Run(args, deps)
		_ = outW.Close()
	}()
	go readLines(outR, m.lines)
	w.procKill = append(w.procKill, m.end.kill)
	return m, nil
}

// signalRunning delivers a signal to the server the scenario started, the MCP server or the serve command.
func (w *world) signalRunning(sig os.Signal) error {
	switch {
	case w.mcp != nil:
		w.mcp.signalled = true
		return w.mcp.signal(sig)
	case w.serve != nil:
		return w.serve.signal(sig)
	}
	return errors.New("no server is running in this scenario")
}

// serveExitStatus returns the exit status of the serve command, after it has ended.
func (w *world) serveExitStatus() (int, error) {
	if w.serve == nil {
		return 0, errors.New("retriever serve has not started in this scenario")
	}
	return w.serve.end.wait()
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
	deadline := replyDeadline()
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
	// A call the server refuses at the protocol level, such as a call to a tool the server does not offer,
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

// finish closes the input unless a signal was sent, waits for the server to end, and returns its exit status.
func (m *mcpSession) finish() (int, error) {
	if !m.signalled {
		_ = m.stdin.Close()
	}
	return m.end.wait()
}
