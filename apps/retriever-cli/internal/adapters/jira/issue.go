package jira

import (
	"context"
	"net/url"
	"strconv"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// issueDocument runs GET /rest/api/3/issue/{key} for one issue key and returns the document. The key is
// checked before the request is built. The reply must name the same key, and must hold an ID, the fields, and a summary. The
// URL is built from the endpoint, never from the reply.
func (g *Gateway) issueDocument(ctx context.Context, rawKey string) (domain.SourceDocument, error) {
	key, err := ParseIssueKey(rawKey)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	values := url.Values{}
	values.Set("fields", issueFieldsList)
	var reply issueReply
	if err := g.call(ctx, epIssue, key, values, nil, &reply); err != nil {
		return domain.SourceDocument{}, err
	}
	if reply.ID == "" || reply.Key != key || reply.Fields == nil || reply.Fields.Summary == nil {
		return domain.SourceDocument{}, domain.ErrUnexpectedResponse
	}
	title := titleText(*reply.Fields.Summary)
	markdown, err := issueMarkdown(key, title, reply.Fields)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	id, err := IssueDocumentID(key)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	collection, err := ProjectCollectionID(projectOfKey(key))
	if err != nil {
		return domain.SourceDocument{}, err
	}
	return domain.SourceDocument{
		ID:           id.String(),
		Title:        title,
		URL:          g.browse + "/browse/" + key,
		Markdown:     markdown,
		CollectionID: collection.String(),
	}, nil
}

// issueRecord returns the document fields of jira.issues.get for the key of the arguments.
func (g *Gateway) issueRecord(ctx context.Context, args domain.Args) (domain.Record, error) {
	doc, err := g.issueDocument(ctx, args.String("issueKey"))
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		domain.Field{Name: "id", Value: doc.ID},
		domain.Field{Name: "title", Value: doc.Title},
		domain.Field{Name: "url", Value: doc.URL},
		domain.Field{Name: "markdown", Value: doc.Markdown},
		domain.Field{Name: "collectionId", Value: doc.CollectionID},
	), nil
}

// commentsList runs one page of GET /rest/api/3/issue/{key}/comment, in the order Jira returns them. The offset is startAt and the
// limit is maxResults. No order and no expansion is sent. A page that holds more comments than the limit is cut to it.
func (g *Gateway) commentsList(ctx context.Context, args domain.Args) (domain.Record, error) {
	key, err := ParseIssueKey(args.String("issueKey"))
	if err != nil {
		return domain.Record{}, err
	}
	limit := args.Int("limit")
	values := url.Values{}
	values.Set("startAt", strconv.Itoa(args.Int("offset")))
	values.Set("maxResults", strconv.Itoa(limit))
	var reply commentsReply
	if err := g.call(ctx, epComments, key, values, nil, &reply); err != nil {
		return domain.Record{}, err
	}
	comments, err := reply.checkComments()
	if err != nil {
		return domain.Record{}, err
	}
	if len(comments) > limit {
		comments = comments[:limit]
	}
	list := make([]domain.Record, 0, len(comments))
	for _, c := range comments {
		markdown, err := RenderADF(c.Body)
		if err != nil {
			return domain.Record{}, err
		}
		list = append(list, domain.NewRecord(
			domain.Field{Name: "id", Value: c.ID},
			domain.Field{Name: "author", Value: personText(c.Author)},
			domain.Field{Name: "created", Value: timestampText(c.Created)},
			domain.Field{Name: "updated", Value: timestampText(c.Updated)},
			domain.Field{Name: "markdown", Value: markdown},
		))
	}
	return domain.NewList(list), nil
}
