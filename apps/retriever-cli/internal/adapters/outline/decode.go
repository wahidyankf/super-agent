package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The projections of the endpoint table. Each one reads a decoded reply (encoding/json into any) and returns the
// operation's Record. Decoding is tolerant (ADR 007 section 6.2): a field that is missing, null, or of another type
// becomes the zero value and is left out of nothing, except an ID that is required and invalid, which makes the whole
// reply unexpected. An extra field is ignored. A reply part that is not there is domain.ErrUnexpectedResponse.

// jsonObject returns v as an object.
func jsonObject(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// jsonList returns v as an array.
func jsonList(v any) ([]any, bool) {
	l, ok := v.([]any)
	return l, ok
}

// dataObject returns the data member of the reply as an object.
func dataObject(root any) (map[string]any, error) {
	top, ok := jsonObject(root)
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	data, ok := jsonObject(top["data"])
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	return data, nil
}

// dataList returns the data member of the reply as an array.
func dataList(root any) ([]any, error) {
	top, ok := jsonObject(root)
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	data, ok := jsonList(top["data"])
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	return data, nil
}

// textOf returns the text at key, or "" when it is missing, null, or not text.
func textOf(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

// textOrNil returns the text at key, or nil when it is missing, null, or not text. A nil field means the server did
// not send the value, which a caller may then read another way.
func textOrNil(m map[string]any, key string) any {
	if s, ok := m[key].(string); ok {
		return s
	}
	return nil
}

// requiredID returns the ID at key. A missing ID, or one that valid refuses, makes the reply unexpected.
func requiredID(m map[string]any, key string, valid func(string) error) (string, error) {
	s, ok := m[key].(string)
	if !ok || valid(s) != nil {
		return "", domain.ErrUnexpectedResponse
	}
	return s, nil
}

// validCollection and validDocument are the ID checks of the two object kinds.
func validCollection(raw string) error {
	_, err := domain.NewCollectionID(raw)
	return err
}

func validDocument(raw string) error {
	_, err := domain.NewDocumentID(raw)
	return err
}

// field is a shorthand for one Record field.
func field(name string, value any) domain.Field {
	return domain.Field{Name: name, Value: value}
}

// withField returns rec with the named field set to value, in the same position when the field exists.
func withField(rec domain.Record, name string, value any) domain.Record {
	fields := rec.Fields()
	for i := range fields {
		if fields[i].Name == name {
			fields[i].Value = value
			return domain.NewRecord(fields...)
		}
	}
	return domain.NewRecord(append(fields, field(name, value))...)
}

// projectCollectionList projects collections.list: one list item per collection, with its ID and name.
func projectCollectionList(root any) (domain.Record, error) {
	data, err := dataList(root)
	if err != nil {
		return domain.Record{}, err
	}
	items := make([]domain.Record, 0, len(data))
	for _, raw := range data {
		m, ok := jsonObject(raw)
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		id, err := requiredID(m, "id", validCollection)
		if err != nil {
			return domain.Record{}, err
		}
		items = append(items, domain.NewRecord(field("id", id), field("name", textOf(m, "name"))))
	}
	return domain.NewList(items), nil
}

// projectCollectionInfo projects collections.info: the ID and name of one collection.
func projectCollectionInfo(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	id, err := requiredID(m, "id", validCollection)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(field("id", id), field("name", textOf(m, "name"))), nil
}

// projectTree projects collections.documents: the documents of one collection, flattened depth first. Each node
// names its parent, and a root has a nil parent and depth 0.
func projectTree(root any) (domain.Record, error) {
	data, err := dataList(root)
	if err != nil {
		return domain.Record{}, err
	}
	var items []domain.Record
	if err := flattenTree(data, nil, 0, &items); err != nil {
		return domain.Record{}, err
	}
	return domain.NewList(items), nil
}

// maxTreeDepth is the deepest node of a document tree the answer keeps. A real wiki nests far less, and the cap keeps a
// hostile reply from building an answer of unbounded depth and indentation.
const maxTreeDepth = 64

// flattenTree appends the nodes depth first. Each child carries its own copy of its parent's ID. A node deeper than
// maxTreeDepth, and the nodes below it, are left out.
func flattenTree(nodes []any, parentID *string, depth int, out *[]domain.Record) error {
	if depth > maxTreeDepth {
		return nil
	}
	for _, raw := range nodes {
		m, ok := jsonObject(raw)
		if !ok {
			return domain.ErrUnexpectedResponse
		}
		id, err := requiredID(m, "id", validDocument)
		if err != nil {
			return err
		}
		var parent any
		if parentID != nil {
			parent = *parentID
		}
		*out = append(*out, domain.NewRecord(
			field("id", id),
			field("title", textOf(m, "title")),
			field("url", textOf(m, "url")),
			field("parentId", parent),
			field("depth", depth),
		))
		children, _ := jsonList(m["children"])
		self := id
		if err := flattenTree(children, &self, depth+1, out); err != nil {
			return err
		}
	}
	return nil
}

// projectDocument projects documents.info: the document of the reply, from the wrapped data.document shape or the
// unwrapped data shape. Its markdown is nil when the reply holds no text, and the operation's after step reads it.
func projectDocument(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	if wrapped, ok := jsonObject(m["document"]); ok {
		m = wrapped
	}
	return documentWithText(m)
}

// documentRecord projects one document object of a list, which carries no text.
func documentRecord(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", validDocument)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("title", textOf(m, "title")),
		field("url", textOf(m, "url")),
		field("collectionId", textOf(m, "collectionId")),
	), nil
}

// documentRevisionRecord projects one document with its revision, which documents.info and the write replies carry. The
// revision is the number a later documents.update names as last_revision (ADR 007 section 2, item 9).
func documentRevisionRecord(m map[string]any) (domain.Record, error) {
	rec, err := documentRecord(m)
	if err != nil {
		return domain.Record{}, err
	}
	return withField(rec, "revision", intOf(m, "revision")), nil
}

// documentWithText projects the document of documents.info, which carries its Markdown when the server sends it. The
// markdown is nil when the reply has no text.
func documentWithText(m map[string]any) (domain.Record, error) {
	rec, err := documentRevisionRecord(m)
	if err != nil {
		return domain.Record{}, err
	}
	return withField(rec, "markdown", textOrNil(m, "text")), nil
}

// exportMissingText is the after step of documents.get. When the reply had no text, which a version 3 server sends as
// editor data, it reads the Markdown with documents.export, the version stable call.
func exportMissingText(ctx context.Context, g *Gateway, a domain.Args, rec domain.Record) (domain.Record, error) {
	if v, _ := rec.Get("markdown"); v != nil {
		return rec, nil
	}
	reply, err := g.exportText(ctx, a.String("documentId"))
	if err != nil {
		return domain.Record{}, err
	}
	return withField(rec, "markdown", reply), nil
}

// projectExport projects documents.export: its data is the Markdown string, and nothing else is kept.
func projectExport(root any) (domain.Record, error) {
	top, ok := jsonObject(root)
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	text, ok := top["data"].(string)
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return domain.NewRecord(field("markdown", text)), nil
}

// projectDocumentList projects documents.list: one list item per document of the page.
func projectDocumentList(root any) (domain.Record, error) {
	data, err := dataList(root)
	if err != nil {
		return domain.Record{}, err
	}
	return documentItems(data)
}

// projectSearch projects documents.search: each hit names the document it found, and the item is that document.
func projectSearch(root any) (domain.Record, error) {
	data, err := dataList(root)
	if err != nil {
		return domain.Record{}, err
	}
	docs := make([]any, 0, len(data))
	for _, raw := range data {
		hit, ok := jsonObject(raw)
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		doc, ok := jsonObject(hit["document"])
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		docs = append(docs, doc)
	}
	return documentItems(docs)
}

// documentItems projects a list of document objects.
func documentItems(data []any) (domain.Record, error) {
	items := make([]domain.Record, 0, len(data))
	for _, raw := range data {
		m, ok := jsonObject(raw)
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		rec, err := documentRecord(m)
		if err != nil {
			return domain.Record{}, err
		}
		items = append(items, rec)
	}
	return domain.NewList(items), nil
}

// projectAccount projects auth.info: the user's ID and name, and the team's name. The ID is what the use case compares
// with a change's target (ADR 007 section 3.5, rule 6). The e-mail address and every other field are never read, so they
// never reach a caller.
func projectAccount(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	user, _ := jsonObject(m["user"])
	team, _ := jsonObject(m["team"])
	return domain.NewRecord(
		field("user", domain.NewRecord(field("id", textOrNil(user, "id")), field("name", textOf(user, "name")))),
		field("team", domain.NewRecord(field("name", textOf(team, "name")))),
	), nil
}
