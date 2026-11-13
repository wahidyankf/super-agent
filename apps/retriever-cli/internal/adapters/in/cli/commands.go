package cli

import (
	"context"
	"fmt"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// baseCommandTree returns the retriever command tree as the command table lists it. commandTree adds the generated read
// commands of each source kind to it.
func baseCommandTree() *command {
	return &command{
		name: "retriever",
		children: []*command{
			{
				name:    "sources",
				summary: "list the configured sources",
				children: []*command{
					{
						name:    "list",
						summary: "list the configured sources, filtered by kind and tags. Access is shown for each source; a command refuses a source whose level is below the one it needs.",
						leaf: &leaf{
							usage: "retriever sources list [--kind <kind>] [--tag <tag[,tag...]>]",
							flags: []flag{
								{name: "--kind", meta: "<kind>", help: "only sources of this kind (known: outline, kiwi, jira)"},
								{name: "--tag", meta: "<tag[,tag...]>", help: "only sources that have every tag given"},
							},
							run: sourcesList,
						},
					},
				},
			},
			{
				name:    "document",
				summary: "document commands that work across source kinds",
				children: []*command{
					{
						name:    "get",
						summary: "print one document of a source: its Markdown, or with --output json the document fields",
						leaf: &leaf{
							usage:      "retriever document get <documentId> [--kind <kind>] [--source <id>]",
							positional: []string{"documentId"},
							flags: []flag{
								{name: "--kind", meta: "<kind>", help: "source kind: outline (default), kiwi, or jira"},
								sourceFlag(),
							},
							run: documentGet,
						},
					},
				},
			},
			{
				name:     "outline",
				summary:  "Outline wiki commands",
				children: []*command{exportCommand(domain.KindOutline)},
			},
			{
				name:     "kiwi",
				summary:  "Kiwi TCMS commands",
				children: []*command{exportCommand(domain.KindKiwi)},
			},
			{
				name:     "jira",
				summary:  "Jira Cloud commands",
				children: []*command{exportCommand(domain.KindJira)},
			},
			{
				name:    "serve",
				summary: "serve the GraphQL API for source content on 127.0.0.1 (read only)",
				leaf: &leaf{
					usage: "retriever serve [--port <port>]",
					flags: []flag{
						{name: "--port", meta: "<port>", help: "loopback port to listen on; 0 picks a free port (default: 0)"},
					},
					run: serveGraphQL,
				},
			},
			{
				name:    "mcp",
				summary: "serve the source catalog to an MCP client over stdio",
				leaf: &leaf{
					usage: "retriever mcp [--max-access read|write|admin] [--tools <resource,...>]",
					flags: []flag{
						{name: "--max-access", meta: "read|write|admin", help: "the highest access level the tools may offer (default: read)"},
						{name: "--tools", meta: "<resource,...>", help: "only the tools of these resources, comma-separated, such as documents,comments; list_sources is always offered (default: every resource)"},
					},
					protocol: true,
					signals:  true,
					run:      runMCP,
				},
			},
		},
	}
}

// sourceFlag is the --source option of a command that reads one source. It names one configured source by its ID.
func sourceFlag() flag {
	return flag{name: "--source", meta: "<id>", help: "configured source ID; required when more than one source of the kind exists"}
}

// sourceFlagFor is the --source option of the commands of one kind.
func sourceFlagFor(kind domain.SourceKind) flag {
	return flag{name: "--source", meta: "<id>", help: "configured source ID; required when more than one " + kind.String() + " source exists"}
}

// documentGet prints one document. Text mode prints its Markdown body. JSON mode prints one document with the fields the
// GraphQL document query returns: schemaVersion, id, title, url, markdown, and collectionId.
func documentGet(inv *invocation) error {
	rt := inv.rt
	kind, err := kindOf(inv.values["--kind"])
	if err != nil {
		return err
	}
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	doc, err := sources.GetDocument(context.Background(), service, kind, sourceArg(inv), inv.positional[0])
	if err != nil {
		return withKind(kind, err)
	}
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.out, documentJSON{
			SchemaVersion: schemaVersion,
			ID:            doc.ID,
			Title:         doc.Title,
			URL:           doc.URL,
			Markdown:      doc.Markdown,
			CollectionID:  doc.CollectionID,
		})
	}
	_, err = fmt.Fprint(rt.out, doc.Markdown)
	return err
}

// documentJSON is the document of the document get command in JSON mode. Its field order is fixed.
type documentJSON struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	Markdown      string `json:"markdown"`
	CollectionID  string `json:"collectionId"`
}

// kindOf returns the source kind a --kind value names. An empty value is outline, the default of every command that
// predates the kind option.
func kindOf(raw string) (domain.SourceKind, error) {
	if raw == "" {
		return domain.KindOutline, nil
	}
	return domain.NewSourceKind(raw)
}

// exportCommand returns the export command of one kind's group. It writes local files, so it is hand-written rather than
// generated from the catalog, and it has no MCP tool (ADR 009 section 3).
func exportCommand(kind domain.SourceKind) *command {
	return &command{
		name:    "export",
		summary: "export one " + kind.Display() + " document as a Markdown file in the --out folder",
		leaf: &leaf{
			usage:      "retriever " + kind.String() + " export <documentId> --out <dir> [--source <id>]",
			positional: []string{"documentId"},
			flags: []flag{
				{name: "--out", meta: "<dir>", help: "folder to write the Markdown file into; created when missing"},
				sourceFlagFor(kind),
			},
			required: []string{"--out"},
			run:      func(inv *invocation) error { return exportDocument(inv, kind) },
		},
	}
}
