// Package cli is the inbound adapter: it parses the command line, calls the core use cases through the ports it is given,
// and reports the result and every failure in one place.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/configpaths"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// Version is the retriever release printed by --version.
const Version = "0.1.0"

// alphaBanner follows the version on --version. The tool is pre-release, so its interface may still change.
const alphaBanner = "ALPHA: expect breaking changes"

// The exit statuses a command may return (CLI convention 001). A command returns only the ones its help page lists.
const (
	exitOK       = 0
	exitNegative = 1
	exitFailed   = 2
	exitTimedOut = 124
	exitRefused  = 125
)

// schemaVersion is the version of the JSON documents the CLI prints, including the failure body.
const schemaVersion = 1

// ServeFunc runs the GraphQL server on the loopback address until ctx ends, then shuts down. The composition root
// binds the listener, where port 0 picks a free port. It calls ready with the bound address once the listener is up.
type ServeFunc func(ctx context.Context, port int, ready func(addr string)) error

// MCPFunc runs one MCP session, offering the tools of
// the resources that --tools names (nil when the flag is absent, which offers every resource). It reads requests from in
// and writes replies to out, until the client closes in or ctx ends. The composition root binds it to the source service
// and the logger. The CLI passes the process streams.
type MCPFunc func(ctx context.Context, resources []string, in io.Reader, out io.Writer) error

// SinkFactory returns the export sink that writes into the folder the --out flag names. The composition root chooses the
// sink, so another sink can replace the default Markdown-files sink without a change to the CLI.
type SinkFactory func(dir string) ports.ExportSink

// Deps are the outside dependencies the CLI needs. The composition root builds them.
type Deps struct {
	Ports  ports.Ports
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) string
	Now    func() time.Time
	// Context is the parent of the context that serve and mcp run under. SIGINT and SIGTERM cancel that context with
	// SignalCause, so a caller can end either command as a signal would, without sending one. Nil means Background.
	Context context.Context
	// Service serves the sources, outline, kiwi, jira, and document commands. Serve runs the GraphQL server, MCP runs the
	// MCP server, and Sink builds the export sink. SetupErr says why the wiring is missing, when it is.
	Service  sources.Service
	Serve    ServeFunc
	MCP      MCPFunc
	Sink     SinkFactory
	SetupErr error
}

// runtime is what a leaf action reads: the ports, the output writers, the parsed output format, and the wiring.
type runtime struct {
	ports    ports.Ports
	ctx      context.Context
	out      io.Writer
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	getenv   func(string) string
	format   domain.OutputFormat
	service  sources.Service
	serve    ServeFunc
	mcp      MCPFunc
	sink     SinkFactory
	setupErr error
	// command is the full path of the command that runs, such as "retriever outline documents get", for the failure body.
	command string
}

// exitStatus is an error a leaf returns to set the process status itself. A nil err means the status carries no message,
// which is how a search with no match returns 1.
type exitStatus struct {
	code int
	err  error
}

func (e exitStatus) Error() string {
	if e.err == nil {
		return fmt.Sprintf("exit status %d", e.code)
	}
	return e.err.Error()
}

func (e exitStatus) Unwrap() error { return e.err }

// Run executes one retriever command line and returns the process exit status. A panic inside a command is reported as
// the internal failure with status 2, and it never reaches the caller as a crash. A write to standard output or standard
// error that meets a closed reader ends the command with status 141, as the built binary ends by SIGPIPE.
func Run(argv []string, d Deps) int {
	stdout := &pipeWatch{w: orDiscard(d.Stdout)}
	stderr := &pipeWatch{w: orDiscard(d.Stderr)}
	d.Stdout, d.Stderr = stdout, stderr
	status := run(argv, d)
	if stdout.broken || stderr.broken {
		return exitBrokenPipe
	}
	return status
}

// run is Run without the closed-pipe check. It parses the command line, runs the leaf, and reports every failure.
func run(argv []string, d Deps) (status int) {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Context == nil {
		d.Context = context.Background()
	}
	command := "retriever"
	defer func() {
		if r := recover(); r != nil {
			status = reportPanic(d.Stderr, outputFormat(argv), command)
		}
	}()

	g, rest, err := splitGlobals(argv)
	if err != nil {
		return reportUsage(d.Stderr, g.format(), command, "retriever", err.Error(), "")
	}
	if g.color != "" && g.color != "auto" && g.color != "always" && g.color != "never" {
		return reportUsage(d.Stderr, g.format(), command, "retriever", "--color must be auto, always, or never", "")
	}
	if g.output != "" && g.output != "text" && g.output != "json" {
		return reportUsage(d.Stderr, domain.FormatText, command, "retriever", "--output must be text or json", "")
	}
	if g.version {
		fmt.Fprintf(d.Stdout, "retriever %s\n%s\n", Version, alphaBanner)
		return exitOK
	}

	if len(rest) > 0 && rest[0] == "help" {
		g.help = true
		rest = rest[1:]
	}
	cur, path, idx := walk(rest)
	command = strings.Join(path, " ")
	area := areaOf(path)

	// A protocol leaf keeps standard output for its messages, so its help and notes go to standard error.
	protocol := cur.leaf != nil && cur.leaf.protocol
	if g.help {
		helpOut := d.Stdout
		if protocol {
			helpOut = d.Stderr
		}
		writeHelp(helpOut, path, cur)
		return exitOK
	}
	if cur.leaf == nil {
		if idx < len(rest) {
			return reportUsage(d.Stderr, g.format(), command, area, "unrecognized argument: "+rest[idx], path[0]+" --help")
		}
		return reportUsage(d.Stderr, g.format(), command, area, "no valid subcommand has been specified", command+" --help")
	}

	inv, err := cur.leaf.parse(rest[idx:])
	if err != nil {
		return reportUsage(d.Stderr, g.format(), command, area, err.Error(), command+" --help")
	}

	stdin := d.Stdin
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	inv.rt = &runtime{
		ports:    d.Ports,
		ctx:      d.Context,
		out:      d.Stdout,
		stdin:    stdin,
		stdout:   d.Stdout,
		stderr:   d.Stderr,
		getenv:   d.Getenv,
		format:   g.format(),
		service:  d.Service,
		serve:    d.Serve,
		mcp:      d.MCP,
		sink:     d.Sink,
		setupErr: d.SetupErr,
		command:  command,
	}
	if err := cur.leaf.run(inv); err != nil {
		var st exitStatus
		if errors.As(err, &st) {
			if st.err != nil {
				return report(d.Stderr, inv.rt.format, command, area, st.code, st.err)
			}
			return st.code
		}
		return report(d.Stderr, inv.rt.format, command, area, exitStatusOf(err), err)
	}
	return exitOK
}

// walk descends the command tree along rest. It returns the deepest command it reaches, that command's path, and how many
// tokens of rest it used. A node that is both a leaf and a group, such as outline collections, descends only when the next
// token names one of its children; any other token is left for the leaf.
func walk(rest []string) (cur *command, path []string, idx int) {
	cur, path = commandTree(), []string{"retriever"}
	for idx < len(rest) {
		next := cur.child(rest[idx])
		if next == nil {
			break
		}
		cur = next
		path = append(path, next.name)
		idx++
	}
	return cur, path, idx
}

// StartupStatus returns the exit status of a failure that stops the process before Run is called, such as a failed
// startup of the wiring. Every failure of the tool that is not a result exits 2, so the status is the same for every
// command (CLI convention 001).
func StartupStatus(argv []string) int {
	return exitFailed
}

// Globals are the path options a caller may give before, after, or between subcommand names. The composition root reads
// them before it wires the sources, so the catalog and the env file are found from the same flags the CLI parses.
type Globals struct {
	// Catalog and EnvFile hold the values of --catalog and --env-file, or "" when the flag is absent.
	Catalog string
	EnvFile string
}

// ParseGlobals returns the path options of argv. A malformed command line returns the error that Run would report.
func ParseGlobals(argv []string) (Globals, error) {
	g, _, err := splitGlobals(argv)
	return Globals{Catalog: g.catalog, EnvFile: g.envFile}, err
}

// outputFormat returns the format named by --output in argv, text when it is absent or malformed. It chooses the stream
// form of a failure that happens before the command line parses.
func outputFormat(argv []string) domain.OutputFormat {
	g, _, _ := splitGlobals(argv)
	return g.format()
}

// areaOf returns the namespace of the codes a command reports: the word after the tool name, such as outline, sources,
// or serve. The tool's own root reports under usage.
func areaOf(path []string) string {
	if len(path) >= 2 {
		return path[1]
	}
	return "usage"
}

// exitStatusOf returns the exit status of an error that carries no status of its own. A timeout and a rate limit are
// both bounded waits that ended in a retryable failure, so both exit 124. A refused setup is the environment refusing
// before work starts. Every other failure did not complete.
func exitStatusOf(err error) int {
	var refused refusedError
	switch {
	case errors.Is(err, domain.ErrTimedOut), errors.Is(err, context.DeadlineExceeded), errors.Is(err, domain.ErrSourceRateLimited):
		return exitTimedOut
	case errors.As(err, &refused), errors.Is(err, configpaths.ErrRefused):
		return exitRefused
	}
	return exitFailed
}

// refusedError marks a failure that the environment caused before any work started, such as a loopback port that another
// process already holds. The CLI reports it with status 125.
type refusedError struct {
	err error
}

func (e refusedError) Error() string { return e.err.Error() }

func (e refusedError) Unwrap() error { return e.err }

// report prints one failure and returns its exit status. In JSON mode the failure body goes to standard error. In text
// mode the message goes to standard error as one "retriever: " line. Standard output stays empty either way.
func report(stderr io.Writer, format domain.OutputFormat, command, area string, status int, err error) int {
	message := publicMessage(err)
	if format == domain.FormatJSON {
		writeErrorBody(stderr, command, status, errorCode(err, area), message, retryable(err))
		return status
	}
	fmt.Fprintf(stderr, "retriever: %s\n", message)
	return status
}

// reportUsage prints a usage mistake and returns status 2. A usage mistake is never a result, so standard output stays
// empty. hint, when given, names the help command to run for the correct form.
func reportUsage(stderr io.Writer, format domain.OutputFormat, command, area, message, hint string) int {
	if format == domain.FormatJSON {
		writeErrorBody(stderr, command, exitFailed, usageCode(area, message), message, false)
		return exitFailed
	}
	fmt.Fprintf(stderr, "retriever: %s\n", message)
	if hint != "" {
		fmt.Fprintf(stderr, "Use '%s' for usage.\n", hint)
	}
	return exitFailed
}

// reportPanic prints the internal failure of a recovered panic. It never prints a stack trace, because a trace is not part
// of the interface (CLI convention 001), and it returns status 2.
func reportPanic(stderr io.Writer, format domain.OutputFormat, command string) int {
	const message = "an internal failure stopped the command"
	if format == domain.FormatJSON {
		writeErrorBody(stderr, command, exitFailed, "retriever.panic", message, false)
		return exitFailed
	}
	fmt.Fprintf(stderr, "retriever: %s (retriever.panic)\n", message)
	return exitFailed
}

// errorBody is the one failure shape every command writes in JSON mode (CLI convention 004 and the tiers record, item 2).
// The field order is fixed, so a caller can read it as a stable document.
type errorBody struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	ExitCode      int         `json:"exitCode"`
	Error         errorDetail `json:"error"`
}

// errorDetail is the error object of the failure body.
type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// writeErrorBody writes the failure body to w as one JSON line.
func writeErrorBody(w io.Writer, command string, status int, code, message string, retry bool) {
	_ = writeJSON(w, errorBody{
		SchemaVersion: schemaVersion,
		Command:       command,
		ExitCode:      status,
		Error:         errorDetail{Code: code, Message: message, Retryable: retry},
	})
}

// usageCode returns the code of a usage mistake: retriever.<area>.<reason>, where the reason follows from the message.
func usageCode(area, message string) string {
	reason := "invalid_argument"
	switch {
	case strings.HasPrefix(message, "unrecognized argument"):
		reason = "unrecognized_argument"
	case strings.HasPrefix(message, "missing"):
		reason = "missing_argument"
	case strings.HasPrefix(message, "--output"):
		reason = "invalid_output"
	case strings.HasPrefix(message, "--color"):
		reason = "invalid_color"
	case strings.Contains(message, "subcommand"):
		reason = "no_subcommand"
	}
	return "retriever." + area + ".usage_" + reason
}
