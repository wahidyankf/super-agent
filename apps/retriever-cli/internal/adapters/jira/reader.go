package jira

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Read runs one catalog operation of the Jira source and returns its Record. It binds the arguments with
// the catalog's rules before any request, so a value that reaches a request has passed them. Only the five read operations
// of the Jira kind are served; any other returns domain.ErrUnsupportedOperation. The use case has already checked the access
// level. Every error it returns is named for Jira.
func (g *Gateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	row, ok := operations.Lookup(op)
	if !ok || !operations.Supports(domain.KindJira, row) || row.Effect != domain.EffectRead {
		return domain.Record{}, kindError(domain.ErrUnsupportedOperation)
	}
	bound, err := operations.Bind(row, args.Values())
	if err != nil {
		return domain.Record{}, kindError(err)
	}
	rec, err := g.readBound(ctx, op, bound)
	return rec, kindError(err)
}

// readBound runs a bound operation. The arguments have passed the catalog's rules.
func (g *Gateway) readBound(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	switch op {
	case domain.OpJiraWhoami:
		return g.whoami(ctx)
	case domain.OpJiraProjectsList:
		return g.projectsList(ctx, args)
	case domain.OpJiraIssuesSearch:
		return g.issuesSearch(ctx, args)
	case domain.OpJiraIssuesGet:
		return g.issueRecord(ctx, args)
	case domain.OpJiraCommentsList:
		return g.commentsList(ctx, args)
	}
	return domain.Record{}, domain.ErrUnsupportedOperation
}

// whoami runs GET /rest/api/3/myself and returns the three fields that the command shows.
func (g *Gateway) whoami(ctx context.Context) (domain.Record, error) {
	var reply myselfReply
	if err := g.call(ctx, epMyself, "", nil, nil, &reply); err != nil {
		return domain.Record{}, err
	}
	return reply.record()
}

// projectsList runs one page of GET /rest/api/3/project/search, for the page the arguments ask for.
func (g *Gateway) projectsList(ctx context.Context, args domain.Args) (domain.Record, error) {
	items, _, err := g.projectPage(ctx, args.Int("offset"), args.Int("limit"), args.String("query"))
	if err != nil {
		return domain.Record{}, err
	}
	list := make([]domain.Record, 0, len(items))
	for _, p := range items {
		list = append(list, domain.NewRecord(
			domain.Field{Name: "key", Value: p.Key},
			domain.Field{Name: "name", Value: p.Name},
			domain.Field{Name: "type", Value: p.Type},
			domain.Field{Name: "style", Value: p.Style},
		))
	}
	return domain.NewList(list), nil
}

// projectPage runs one GET /rest/api/3/project/search for the page that starts at startAt and holds at most maxResults projects.
// It returns the checked projects and whether more may follow. A page that holds more than maxResults projects is cut to that
// count, and more is then true, so no project is lost from the next page.
func (g *Gateway) projectPage(ctx context.Context, startAt, maxResults int, query string) ([]project, bool, error) {
	values := url.Values{}
	values.Set("startAt", strconv.Itoa(startAt))
	values.Set("maxResults", strconv.Itoa(maxResults))
	if query != "" {
		values.Set("query", query)
	}
	var reply projectsReply
	if err := g.call(ctx, epProjectSearch, "", values, nil, &reply); err != nil {
		return nil, false, err
	}
	items, isLast, err := reply.projects()
	if err != nil {
		return nil, false, err
	}
	more := !isLast
	if len(items) > maxResults {
		items = items[:maxResults]
		more = true
	}
	return items, more, nil
}

// issuesSearch runs one search for the JQL of the arguments, with at most the limit of issues. The JQL is data: it goes into
// the body as one JSON string and is never changed.
func (g *Gateway) issuesSearch(ctx context.Context, args domain.Args) (domain.Record, error) {
	limit := args.Int("limit")
	reply, err := g.jqlPage(ctx, args.String("jql"), limit, "")
	if err != nil {
		return domain.Record{}, err
	}
	issues, err := reply.issues()
	if err != nil {
		return domain.Record{}, err
	}
	if len(issues) == 0 && reply.NextPageToken != "" {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	truncated := reply.NextPageToken != "" && !reply.IsLast
	if len(issues) > limit {
		issues = issues[:limit]
		truncated = true
	}
	list := make([]domain.Record, 0, len(issues))
	for _, b := range issues {
		list = append(list, searchRecord(b, truncated))
	}
	page := domain.Page{Paged: true, HasMore: truncated, NextOffset: len(list)}
	return domain.NewList(list).WithPage(page), nil
}

// jqlPage posts one search request. The JQL, the limit, and the token are members of the body; the fields list is fixed.
func (g *Gateway) jqlPage(ctx context.Context, jql string, maxResults int, token string) (searchReply, error) {
	body, err := json.Marshal(searchRequest{JQL: jql, MaxResults: maxResults, Fields: searchFields, NextPageToken: token})
	if err != nil {
		return searchReply{}, domain.ErrUnexpectedResponse
	}
	var reply searchReply
	if err := g.call(ctx, epJQLSearch, "", nil, body, &reply); err != nil {
		return searchReply{}, err
	}
	return reply, nil
}

// searchRecord returns the row of one issue in a search: the fields of the jira.issues.search row, with truncated set to the
// page's flag. The flag is on every item because the row declares it as an item field; the page carries it too.
func searchRecord(b issueBody, truncated bool) domain.Record {
	f := b.Fields
	return domain.NewRecord(
		domain.Field{Name: "key", Value: b.Key},
		domain.Field{Name: "summary", Value: titleText(*f.Summary)},
		domain.Field{Name: "status", Value: namedText(f.Status)},
		domain.Field{Name: "type", Value: namedText(f.IssueType)},
		domain.Field{Name: "priority", Value: namedText(f.Priority)},
		domain.Field{Name: "assignee", Value: personText(f.Assignee)},
		domain.Field{Name: "updated", Value: timestampText(f.Updated)},
		domain.Field{Name: "projectKey", Value: projectOfKey(b.Key)},
		domain.Field{Name: "truncated", Value: truncated},
	)
}
