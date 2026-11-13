package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// command is one node of the retriever command tree. A node either groups other commands (children) or is a leaf that
// runs an action (leaf).
type command struct {
	name     string
	summary  string
	children []*command
	leaf     *leaf
}

// leaf describes a runnable command: its usage line, positional arguments, flags, and the action to run once the
// arguments parse.
type leaf struct {
	usage      string
	positional []string
	variadic   bool
	flags      []flag
	// required lists the flags, by name such as "--out", that the command cannot run without. parse refuses a command line
	// that leaves one out, and names it.
	required []string
	// protocol marks a leaf whose standard output carries messages for a client, so the CLI keeps notes and help off that
	// stream.
	protocol bool
	// noMatch marks a catalog read whose empty result exits 1, as its row's NoMatch says.
	noMatch bool
	run     func(*invocation) error
}

// flag is a long option a leaf accepts. A switch has no meta; a valued flag has a placeholder such as <path> in meta.
type flag struct {
	name string
	meta string
	help string
}

// invocation is a leaf's parsed arguments, plus the runtime it runs against.
type invocation struct {
	rt         *runtime
	positional []string
	values     map[string]string
	switches   map[string]bool
}

func (c *command) child(name string) *command {
	for _, ch := range c.children {
		if ch.name == name {
			return ch
		}
	}
	return nil
}

func (l *leaf) flagNamed(name string) *flag {
	for i := range l.flags {
		if l.flags[i].name == name {
			return &l.flags[i]
		}
	}
	return nil
}

// parse reads a leaf's tokens. Long flags may appear anywhere; a bare "--" ends option parsing. Positional counts are
// checked against the leaf's declaration.
func (l *leaf) parse(tokens []string) (*invocation, error) {
	inv := &invocation{values: map[string]string{}, switches: map[string]bool{}}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" {
			inv.positional = append(inv.positional, tokens[i+1:]...)
			break
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			inv.positional = append(inv.positional, tok)
			continue
		}

		name, value, hasValue := strings.Cut(tok, "=")
		f := l.flagNamed(name)
		if f == nil {
			return nil, fmt.Errorf("unrecognized argument: %s", tok)
		}
		if f.meta == "" {
			if hasValue {
				return nil, fmt.Errorf("%s does not take a value", name)
			}
			inv.switches[f.name] = true
			continue
		}
		if !hasValue {
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", name)
			}
			i++
			value = tokens[i]
		}
		inv.values[f.name] = value
	}

	need := len(l.positional)
	if !l.variadic && len(inv.positional) > need {
		return nil, fmt.Errorf("unrecognized argument: %s", inv.positional[need])
	}
	var missing []string
	if len(inv.positional) < need {
		missing = append(missing, angled(l.positional[len(inv.positional):]))
	}
	for _, name := range l.required {
		if _, given := inv.values[name]; !given {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing arguments: %s", strings.Join(missing, " "))
	}
	return inv, nil
}

func angled(names []string) string {
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = "<" + n + ">"
	}
	return strings.Join(parts, " ")
}

// globals are the options accepted before, after, or between subcommand names.
type globals struct {
	output  string
	color   string
	catalog string
	envFile string
	version bool
	help    bool
}

// format returns the output format the --output flag selects. Text is the default, and json is the machine-readable mode.
func (g globals) format() domain.OutputFormat {
	return formatFor(g.output)
}

// splitGlobals removes the global options from argv and returns what is left. A value that is missing is an error.
func splitGlobals(argv []string) (globals, []string, error) {
	var g globals
	var rest []string

	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if tok == "--" {
			rest = append(rest, argv[i:]...)
			break
		}

		name, value, hasValue := strings.Cut(tok, "=")
		switch name {
		case "-o", "--output", "--color", "--catalog", "--env-file":
			if !hasValue {
				if i+1 >= len(argv) {
					return g, nil, fmt.Errorf("missing value for %s", name)
				}
				i++
				value = argv[i]
			}
			switch name {
			case "--color":
				g.color = value
			case "--catalog":
				g.catalog = value
			case "--env-file":
				g.envFile = value
			default:
				g.output = value
			}
		case "-V", "--version":
			g.version = true
		case "-h", "--help":
			g.help = true
		default:
			rest = append(rest, tok)
		}
	}
	return g, rest, nil
}

// Exit statuses that every help page lists, with the meaning each one carries for the command.
var (
	statusCompleted = [2]string{"0", "the command completed"}
	statusNoMatch   = [2]string{"1", "the search matched no item; an empty result is still printed"}
	statusFailed    = [2]string{"2", "the command did not complete: a usage mistake, a configuration error, or a failed request"}
	statusTimedOut  = [2]string{"124", "a bounded wait elapsed before the command finished; the failure is retryable"}
	statusRefused   = [2]string{"125", "the environment refused before the command started, such as a port already in use"}
)

// writeHelp prints the help page for c, whose full path is path.
func writeHelp(w io.Writer, path []string, c *command) {
	switch {
	case c.leaf != nil:
		fmt.Fprintf(w, "USAGE: %s\n\n%s\n", c.leaf.usage, c.summary)
		if len(c.leaf.positional) > 0 {
			rows := make([][2]string, len(c.leaf.positional))
			for i, p := range c.leaf.positional {
				rows[i] = [2]string{"<" + p + ">", ""}
			}
			writeList(w, "ARGUMENTS", rows)
		}
		rows := [][2]string{}
		for _, f := range c.leaf.flags {
			label := f.name
			if f.meta != "" {
				label += " " + f.meta
			}
			rows = append(rows, [2]string{label, f.help})
		}
		rows = append(rows, [2]string{"-h, --help", "show help"})
		writeList(w, "OPTIONS", rows)
		if len(c.children) > 0 {
			writeList(w, "SUBCOMMANDS", childRows(c))
		}
		writeList(w, "EXIT STATUS", exitRows(c))

	case len(path) == 1:
		fmt.Fprintf(w, "USAGE: retriever <subcommand> [options]\n")
		writeList(w, "SUBCOMMANDS", childRows(c))
		writeList(w, "OPTIONS", globalOptionRows())
		writeList(w, "EXIT STATUS", exitRows(c))
		fmt.Fprintf(w, "\nUse 'retriever <subcommand> --help' for more information on a subcommand.\n")

	default:
		fmt.Fprintf(w, "USAGE: %s <subcommand> [options]\n\n%s\n", strings.Join(path, " "), c.summary)
		writeList(w, "SUBCOMMANDS", childRows(c))
		writeList(w, "OPTIONS", [][2]string{{"-h, --help", "show help"}})
		writeList(w, "EXIT STATUS", exitRows(c))
	}
}

// globalOptionRows lists the options the root accepts before any subcommand, with the value each one takes.
func globalOptionRows() [][2]string {
	return [][2]string{
		{"-o, --output <format>", "output format: text (default) or json; json prints one document with a schemaVersion field"},
		{"--color <when>", "colour control: auto (default), always, or never; the output carries no escape sequences either way"},
		{"--catalog <path>", "sources catalog file (default: RETRIEVER_CATALOG, ./sources.yml, or the user configuration folder)"},
		{"--env-file <path>", "local env file holding source keys (default: RETRIEVER_ENV_FILE, ./.env.local, or the user configuration folder)"},
		{"-V, --version", "print the retriever version and the alpha notice, then exit"},
		{"-h, --help", "show help"},
	}
}

// exitRows returns the exit statuses a help page lists. Every command lists the statuses it can return, and a command that
// searches also lists 1 for an empty result (CLI convention 001 and 008).
func exitRows(c *command) [][2]string {
	rows := [][2]string{statusCompleted}
	if c.leaf != nil && c.leaf.noMatch {
		rows = append(rows, statusNoMatch)
	}
	rows = append(rows, statusFailed, statusTimedOut, statusRefused)
	return rows
}

func childRows(c *command) [][2]string {
	rows := make([][2]string, 0, len(c.children))
	for _, ch := range c.children {
		rows = append(rows, [2]string{ch.name, ch.summary})
	}
	return rows
}

func writeList(w io.Writer, heading string, rows [][2]string) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	fmt.Fprintf(w, "\n%s:\n", heading)
	for _, r := range rows {
		if r[1] == "" {
			fmt.Fprintf(w, "  %s\n", r[0])
			continue
		}
		fmt.Fprintf(w, "  %-*s  %s\n", width, r[0], r[1])
	}
}
