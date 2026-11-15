package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/jiraseam"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// jiraDocumentNotFoundReads are the engine-served reads whose 404 names a missing issue: the single-issue and
// comment reads whose path holds the issue key, as the retired Go gateway maps. Every other engine failure passes
// through. The search read keeps its own page-cap refusal, which is never a not found.
var jiraDocumentNotFoundReads = map[domain.OperationID]bool{
	domain.OpJiraIssuesGet:    true,
	domain.OpJiraCommentsList: true,
}

// jiraSpecOperations maps each catalog operation of the Jira kind to the operation of the bundled jira.yml that
// serves it. A spec operation never repeats a catalog row name: WithPinned collects the spec rows by operation ID,
// so an identical name would collide with the built-in row it claims.
var jiraSpecOperations = map[domain.OperationID]domain.OperationID{
	domain.OpJiraWhoami:       "jira.auth.whoami",
	domain.OpJiraProjectsList: "jira.projects.page",
	domain.OpJiraIssuesSearch: "jira.issues.query",
	domain.OpJiraIssuesGet:    "jira.issues.read",
	domain.OpJiraCommentsList: "jira.comments.page",
}

// jiraTreeOperation is the bundled operation of the tree walk: one cursor read of the search endpoint with the
// page-end rule. The gateway passes the JQL of one project, and the engine walks at most five pages of 100 issues.
// jiraProjectsWalk is the bundled operation of the collection walk: one offset read of the project search with the
// page-end rule, walking at most twenty pages of 50 projects.
const (
	jiraTreeOperation = domain.OperationID("jira.issues.tree")
	jiraProjectsWalk  = domain.OperationID("jira.projects.all")
)

// jiraGateway is the Jira gateway: the engine of the bundled jira.yml, with the document-not-found translation the
// retired Go gateway maps and the search page the engine leaves to the caller. Every Jira operation runs on the
// engine: reads through the spec operations they name, the legacy collection, tree, and document reads through the
// walk and get operations, with nothing dropped.
type jiraGateway struct {
	engine *engine.Engine
}

var (
	_ ports.SourceGateway = (*jiraGateway)(nil)
	_ ports.SourceReader  = (*jiraGateway)(nil)
)

// buildJiraGateway returns the gateway of one Jira source: the engine of the bundled jira.yml. A source whose
// engine cannot be built stops the source.
func buildJiraGateway(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (ports.SourceGateway, error) {
	eng, err := buildJiraEngine(model, src, creds, seam, eseam)
	if err != nil {
		return nil, err
	}
	return &jiraGateway{engine: eng}, nil
}

// buildJiraEngine returns the engine of the bundled jira.yml for one source. The seam, when it is on, replaces
// only the connection base: the transport rewrites each request to the seam base and round-trips with the seam
// client, so the engine reaches the stub. The seam's timeout bounds one engine request, so a run that gets no reply
// stops at the bound. Without a seam the engine is built as any other spec source is.
func buildJiraEngine(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (*engine.Engine, error) {
	if !seam.on || seam.client == nil {
		return buildSpec(model, src, eseam, creds)
	}
	base, err := url.Parse(seam.base)
	if err != nil {
		return nil, err
	}
	if err := jiraseam.CheckBaseURL(seam.base); err != nil {
		return nil, err
	}
	opts := eseam.options
	if seam.timeout > 0 {
		opts = append(append([]engine.Option{}, eseam.options...), engine.WithTimeout(seam.timeout))
	}
	return engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       eseam.clock,
		Transport:   seamEngineTransport{base: base, rt: hardenSeamTransport(seam.client.Transport, src.Endpoint().Host())},
		Roots:       eseam.roots,
	}, opts...)
}

// hardenSeamTransport returns a copy of the seam transport that sends the endpoint host as the TLS server name, as
// the retired Go gateway did: the seam client trusts the stub CA but names no server, so the stub would record an empty
// name. A transport that is not an *http.Transport carries no TLS settings of its own, so it is returned as it is.
func hardenSeamTransport(rt http.RoundTripper, host string) http.RoundTripper {
	tr, ok := rt.(*http.Transport)
	if !ok {
		return rt
	}
	clone := tr.Clone()
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		clone.TLSClientConfig = clone.TLSClientConfig.Clone()
	}
	clone.TLSClientConfig.ServerName = host
	return clone
}

// restoreJiraError restores the bare sentinel of an unreachable source, as the retired Go gateway mapped it: the engine
// carries the dial text, which may name the loopback address, and the presenter must never show it. Every other
// error keeps the document translation of the operation.
func restoreJiraError(op domain.OperationID, err error) error {
	if errors.Is(err, domain.ErrUnreachable) {
		return domain.ErrUnreachable
	}
	return translateJiraNotFound(op, err)
}

// Read runs one catalog operation of the Jira source against the stub on the engine, restoring the document
// sentinel the retired Go gateway maps and setting the search page the engine leaves to the caller.
func (g *jiraGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	specOp, ok := jiraSpecOperations[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	rec, err := g.engine.Read(ctx, specOp, args)
	if err != nil {
		return domain.Record{}, restoreJiraError(op, err)
	}
	if op == domain.OpJiraIssuesSearch {
		rec = withSearchPage(rec)
	}
	return rec, nil
}

// translateJiraNotFound restores the document-not-found sentinel of an engine-served read, which the engine reports
// as a plain not found. Any other outcome passes through, so an unavailable method keeps its own failure.
func translateJiraNotFound(op domain.OperationID, err error) error {
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}
	if jiraDocumentNotFoundReads[op] {
		return domain.ErrDocumentNotFound
	}
	return err
}

// withSearchPage sets the page of an issue search from the truncated flag of its items, as the retired Go gateway
// mapped it: the flag is on exactly when more issues match than the page holds, and the next offset is the number
// of issues shown. The engine marks each item but sets no page, so the gateway restores it here.
func withSearchPage(rec domain.Record) domain.Record {
	items := rec.Items()
	more := false
	if len(items) > 0 {
		if flag, ok := items[0].Get("truncated"); ok {
			more, _ = flag.(bool)
		}
	}
	return rec.WithPage(domain.Page{Paged: true, HasMore: more, NextOffset: len(items)})
}

// ListCollections returns the projects of the site as collections, read from the bundled-spec engine, which walks
// every page of the page-end rule. The spec maps the raw project key, so the gateway prefixes it, as the retired Go
// gateway did: a key outside the project shape stops the listing instead of naming a bad collection.
func (g *jiraGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	rec, err := g.engine.Read(ctx, jiraProjectsWalk, domain.NewArgs(nil))
	if err != nil {
		return nil, restoreJiraError(domain.OpJiraProjectsList, err)
	}
	out := make([]domain.Collection, 0, len(rec.Items()))
	for _, item := range rec.Items() {
		id, err := jiraseam.ProjectCollectionID(item.Str("id"))
		if err != nil {
			return nil, domain.ErrUnexpectedResponse
		}
		out = append(out, domain.Collection{ID: id.String(), Name: item.Str("name")})
	}
	return out, nil
}

// CollectionTree returns the issues of one project as a flat list, read from the engine's cursor walk of the
// bundled spec: the JQL of the project key in key order, walked at most five pages. A project the server refuses
// keeps the collection-not-found sentinel of the tree walk.
func (g *jiraGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	key, err := jiraseam.ProjectKey(id)
	if err != nil {
		return nil, err
	}
	rec, err := g.engine.Read(ctx, jiraTreeOperation,
		domain.NewArgs(map[string]any{"jql": `project = "` + key + `" ORDER BY key ASC`}))
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, restoreJiraError(domain.OpJiraIssuesSearch, err)
	}
	return jiraTreeNodes(rec)
}

// jiraTreeNodes maps the issues of the tree walk to flat nodes. A parent ID is set only when the parent is in the
// list. The depth is the length of the parent chain inside the list, and a chain that does not end is an unexpected
// reply, as the retired Go gateway maps it.
func jiraTreeNodes(rec domain.Record) ([]domain.TreeNode, error) {
	items := rec.Items()
	index := make(map[string]int, len(items))
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.Str("key")
		index[keys[i]] = i
	}
	parentOf := make([]int, len(items))
	nodes := make([]domain.TreeNode, len(items))
	for i, item := range items {
		node := domain.TreeNode{
			ID:    "issue-" + keys[i],
			Title: item.Str("title"),
			URL:   item.Str("url"),
		}
		parentOf[i] = -1
		if parent, ok := item.Get("parent"); ok {
			if text, isText := parent.(string); isText && text != "" {
				if _, err := jiraseam.ParseIssueKey(text); err == nil {
					if j, ok := index[text]; ok {
						parentOf[i] = j
						issue := "issue-" + text
						node.ParentID = &issue
					}
				}
			}
		}
		nodes[i] = node
	}
	for i := range nodes {
		depth := 0
		for j := parentOf[i]; j >= 0; j = parentOf[j] {
			depth++
			if depth > len(nodes) {
				return nil, domain.ErrUnexpectedResponse
			}
		}
		nodes[i].Depth = depth
	}
	return nodes, nil
}

// Document returns one issue as a document from the engine's get read of the bundled spec, with its Markdown body:
// the id, title, URL, Markdown, and collection ID the spec maps. The ID must be issue-<KEY-123>. A missing issue
// keeps the document-not-found sentinel of the get read.
func (g *jiraGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	key, err := jiraseam.IssueKey(id)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	rec, err := g.engine.Read(ctx, jiraSpecOperations[domain.OpJiraIssuesGet],
		domain.NewArgs(map[string]any{"issueKey": key}))
	if err != nil {
		return domain.SourceDocument{}, restoreJiraError(domain.OpJiraIssuesGet, err)
	}
	return domain.SourceDocument{
		ID:           rec.Str("id"),
		Title:        rec.Str("title"),
		URL:          rec.Str("url"),
		Markdown:     rec.Str("markdown"),
		CollectionID: rec.Str("collectionId"),
	}, nil
}
