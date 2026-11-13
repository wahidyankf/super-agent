package jira

import (
	"context"
	"errors"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// The generic reads of the source port (ADR 010 section 3). A project is a collection, and an issue is a document and a
// node of its project's tree.
const (
	projectPageSize = 50
	projectPageCap  = 20
	treePageSize    = 100
	treePageCap     = 5
)

// Compile-time checks: the gateway is a generic source and a read port. It is not a writer or an admin.
var (
	_ ports.SourceGateway = (*Gateway)(nil)
	_ ports.SourceReader  = (*Gateway)(nil)
)

// ListCollections returns every project of the site as a collection. It reads pages of 50 with startAt, and stops at the last
// page or at an empty page. At most 20 pages are read (1000 projects); a listing that does not end returns ErrPageCapReached.
func (g *Gateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	var out []domain.Collection
	startAt := 0
	for page := 0; page < projectPageCap; page++ {
		items, more, err := g.projectPage(ctx, startAt, projectPageSize, "")
		if err != nil {
			return nil, kindError(err)
		}
		for _, p := range items {
			id, err := ProjectCollectionID(p.Key)
			if err != nil {
				return nil, kindError(domain.ErrUnexpectedResponse)
			}
			out = append(out, domain.Collection{ID: id.String(), Name: p.Name})
		}
		if !more || len(items) == 0 {
			return out, nil
		}
		startAt += len(items)
	}
	return nil, kindError(domain.ErrPageCapReached)
}

// CollectionTree returns the issues of one project as a flat list, in key order, with the parent of each issue when that parent
// is in the list, and its depth from the parent chain. The one JQL is project = "<KEY>" ORDER BY key ASC, where KEY passed the
// project pattern. At most five pages of 100 are read; more returns ErrPageCapReached. A project the server refuses (400) is
// ErrCollectionNotFound. The node IDs are issue document IDs (issue-<KEY-123>).
func (g *Gateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	key, err := ProjectKey(id)
	if err != nil {
		return nil, kindError(err)
	}
	issues, err := g.treeIssues(ctx, `project = "`+key+`" ORDER BY key ASC`)
	if err != nil {
		return nil, kindError(err)
	}
	nodes, err := treeNodes(issues, g.browse)
	if err != nil {
		return nil, kindError(err)
	}
	return nodes, nil
}

// treeIssues walks the pages of one constant search. A page is kept only when the reply is well formed. The walk ends at the
// last page. A token that is repeated, or an empty page that carries a token, is an unexpected reply, and so is a page cut to
// the limit that has no token after it, because those issues could not be read.
func (g *Gateway) treeIssues(ctx context.Context, jql string) ([]issueBody, error) {
	var all []issueBody
	seen := map[string]bool{}
	token := ""
	for page := 0; page < treePageCap; page++ {
		reply, err := g.jqlPage(ctx, jql, treePageSize, token)
		if errors.Is(err, domain.ErrBadRequest) {
			return nil, domain.ErrCollectionNotFound
		}
		if err != nil {
			return nil, err
		}
		issues, err := reply.issues()
		if err != nil {
			return nil, err
		}
		trimmed := len(issues) > treePageSize
		if trimmed {
			issues = issues[:treePageSize]
		}
		all = append(all, issues...)
		next := reply.NextPageToken
		if next == "" || reply.IsLast {
			if trimmed {
				return nil, domain.ErrUnexpectedResponse
			}
			return all, nil
		}
		if len(issues) == 0 || seen[next] {
			return nil, domain.ErrUnexpectedResponse
		}
		seen[next] = true
		token = next
	}
	return nil, domain.ErrPageCapReached
}

// treeNodes turns the issues of a project into tree nodes. A parent ID is set only when the parent is in the list. The depth is
// the length of the parent chain inside the list, and a chain that does not end is an unexpected reply.
func treeNodes(issues []issueBody, browse string) ([]domain.TreeNode, error) {
	index := make(map[string]int, len(issues))
	for i, b := range issues {
		index[b.Key] = i
	}
	parentOf := make([]int, len(issues))
	nodes := make([]domain.TreeNode, len(issues))
	for i, b := range issues {
		node := domain.TreeNode{
			ID:    issuePrefix + b.Key,
			Title: titleText(*b.Fields.Summary),
			URL:   browse + "/browse/" + b.Key,
		}
		parentOf[i] = -1
		if p := parentKey(b.Fields); p != "" {
			if j, ok := index[p]; ok {
				parentOf[i] = j
				parent := issuePrefix + p
				node.ParentID = &parent
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

// Document returns one issue as a document with its Markdown body. The ID must be issue-<KEY-123>. A project ID, a lower-case
// key, or any other form is refused before any request.
func (g *Gateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	key, err := IssueKey(id)
	if err != nil {
		return domain.SourceDocument{}, kindError(err)
	}
	doc, err := g.issueDocument(ctx, key)
	return doc, kindError(err)
}
