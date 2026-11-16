// Gateway is the SourceGateway, SourceReader, and SourceWriter of one plugin source: it serves the source by
// calling its pinned plugin once per call. The engine and the plugin never import each other, and both implement
// the same three ports, so the runtime sees one shape (module layout).
//
// The reply shapes below are the plugin protocol of this pull request. Each method answers with a JSON-RPC 2.0
// result object:
//
//	list_collections: {"collections": [{"id", "name"}]}
//	collection_tree:  {"nodes": [{"id", "title", "url", "parent_id" (string or null), "depth"}]}
//	document:         {"id", "title", "url", "markdown", "collection_id"}
//	read, write:      {"fields": {name: value}} or {"items": [{name: value}]}
//
// A null result, a missing object, or a mistyped field fails with ProtocolError: the gateway never guesses a value
// the plugin did not send. A paged shape is not served: the plugin returns the whole answer, and the gateway maps
// it without a page.
package plugin

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

var (
	_ ports.SourceGateway = (*Gateway)(nil)
	_ ports.SourceReader  = (*Gateway)(nil)
	_ ports.SourceWriter  = (*Gateway)(nil)
)

// Gateway serves one plugin source through its pinned plugin: one process per call, one JSON-RPC request per
// process. Build it with the source, the credentials and identities that resolve its secret values, the seams of
// the call, and the bound of each call (zero reads as DefaultTimeout).
type Gateway struct {
	Source      domain.Source
	Credentials ports.Credentials
	Identities  ports.Identities
	Seams       Seams
	Timeout     time.Duration
}

// call runs one plugin method with params and returns the decoded result object. The secret values of the source
// are resolved here and never kept: the call redacts them from the reply.
func (g *Gateway) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	if !g.Source.HasPlugin() {
		return nil, StartError{Rule: "the source has no plugin"}
	}
	key, err := g.Credentials.APIKey(g.Source.Credential())
	if err != nil {
		return nil, err
	}
	cfg := Config{
		Plugin:          g.Source.Plugin(),
		CredentialEnv:   g.Source.Credential().Name(),
		CredentialValue: key.Reveal(),
		Timeout:         g.Timeout,
	}
	if identity := g.Source.Identity(); identity != (domain.IdentityRef{}) {
		address, err := g.Identities.Identity(identity)
		if err != nil {
			return nil, err
		}
		cfg.IdentityEnv = identity.Name()
		cfg.IdentityValue = address.Reveal()
	}
	reply, err := Call(ctx, cfg, method, params, g.Seams)
	if err != nil {
		return nil, err
	}
	return reply.Result, nil
}

// ListCollections returns the collections of the source, through the list_collections method of its plugin.
func (g *Gateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	raw, err := g.call(ctx, "list_collections", map[string]any{"source": g.Source.ID().String()})
	if err != nil {
		return nil, err
	}
	obj, err := asObject(raw)
	if err != nil {
		return nil, err
	}
	list, ok := obj["collections"].([]any)
	if !ok {
		return nil, ProtocolError{Rule: "the plugin reply has no collections list"}
	}
	out := make([]domain.Collection, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		id, hasID := entry["id"].(string)
		name, hasName := entry["name"].(string)
		if !ok || !hasID || !hasName {
			return nil, ProtocolError{Rule: "the plugin collection has no id and name"}
		}
		out = append(out, domain.Collection{ID: id, Name: name})
	}
	return out, nil
}

// CollectionTree returns the flat document tree of one collection, through the collection_tree method of the
// plugin. A null parent_id reads as a root node.
func (g *Gateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	raw, err := g.call(ctx, "collection_tree", map[string]any{
		"source":        g.Source.ID().String(),
		"collection_id": id.String(),
	})
	if err != nil {
		return nil, err
	}
	obj, err := asObject(raw)
	if err != nil {
		return nil, err
	}
	list, ok := obj["nodes"].([]any)
	if !ok {
		return nil, ProtocolError{Rule: "the plugin reply has no nodes list"}
	}
	out := make([]domain.TreeNode, 0, len(list))
	for _, item := range list {
		node, err := asTreeNode(item)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

// asTreeNode maps one plugin node onto a domain.TreeNode: the id, title, and url are text, the parent_id is text
// or null, and the depth is a whole number at or above zero.
func asTreeNode(item any) (domain.TreeNode, error) {
	entry, ok := item.(map[string]any)
	id, hasID := entry["id"].(string)
	title, hasTitle := entry["title"].(string)
	url, hasURL := entry["url"].(string)
	if !ok || !hasID || !hasTitle || !hasURL {
		return domain.TreeNode{}, ProtocolError{Rule: "the plugin node has no id, title, and url"}
	}
	var parent *string
	if raw, present := entry["parent_id"]; present && raw != nil {
		text, isText := raw.(string)
		if !isText {
			return domain.TreeNode{}, ProtocolError{Rule: "the plugin node parent is not text or null"}
		}
		parent = &text
	}
	depth, isDepth := entry["depth"].(float64)
	if !isDepth || depth != math.Trunc(depth) || depth < 0 {
		return domain.TreeNode{}, ProtocolError{Rule: "the plugin node depth is not a whole number"}
	}
	return domain.TreeNode{ID: id, Title: title, URL: url, ParentID: parent, Depth: int(depth)}, nil
}

// Document returns one document with its Markdown body, through the document method of the plugin.
func (g *Gateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	raw, err := g.call(ctx, "document", map[string]any{
		"source":      g.Source.ID().String(),
		"document_id": id.String(),
	})
	if err != nil {
		return domain.SourceDocument{}, err
	}
	obj, err := asObject(raw)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	docID, hasID := obj["id"].(string)
	title, hasTitle := obj["title"].(string)
	url, hasURL := obj["url"].(string)
	body, hasBody := obj["markdown"].(string)
	collection, hasCollection := obj["collection_id"].(string)
	if !hasID || !hasTitle || !hasURL || !hasBody || !hasCollection {
		return domain.SourceDocument{}, ProtocolError{Rule: "the plugin document has no id, title, url, markdown, and collection"}
	}
	return domain.SourceDocument{ID: docID, Title: title, URL: url, Markdown: body, CollectionID: collection}, nil
}

// Read runs one read operation of the catalog against the plugin, through its read method with the operation and
// the bound arguments. The mapped record holds the fields of the reply, in name order.
func (g *Gateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	raw, err := g.call(ctx, "read", map[string]any{
		"source":    g.Source.ID().String(),
		"operation": string(op),
		"args":      args.Values(),
	})
	if err != nil {
		return domain.Record{}, err
	}
	return asRecord(raw)
}

// Write runs one change operation of the catalog against the plugin, through its write method with the operation
// and the bound arguments. The mapped record holds the fields of the reply, in name order.
func (g *Gateway) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	raw, err := g.call(ctx, "write", map[string]any{
		"source":    g.Source.ID().String(),
		"operation": string(op),
		"args":      args.Values(),
	})
	if err != nil {
		return domain.Record{}, err
	}
	return asRecord(raw)
}

// asRecord maps a read or write reply onto a domain.Record: a fields object maps onto an object record, and an
// items list maps onto a list record of one object record per item. Every object reads its fields in name order, so
// the record reads the same way on every run. A missing or mistyped shape fails with ProtocolError.
func asRecord(raw json.RawMessage) (domain.Record, error) {
	obj, err := asObject(raw)
	if err != nil {
		return domain.Record{}, err
	}
	if items, present := obj["items"]; present {
		list, ok := items.([]any)
		if !ok {
			return domain.Record{}, ProtocolError{Rule: "the plugin reply has no items list"}
		}
		out := make([]domain.Record, 0, len(list))
		for _, item := range list {
			fields, ok := item.(map[string]any)
			if !ok {
				return domain.Record{}, ProtocolError{Rule: "the plugin list item is not an object"}
			}
			out = append(out, domain.NewRecord(recordFields(fields)...))
		}
		return domain.NewList(out), nil
	}
	fields, ok := obj["fields"].(map[string]any)
	if !ok {
		return domain.Record{}, ProtocolError{Rule: "the plugin reply has no fields object"}
	}
	return domain.NewRecord(recordFields(fields)...), nil
}

// recordFields returns the fields of one mapped object in name order.
func recordFields(fields map[string]any) []domain.Field {
	list := make([]domain.Field, 0, len(fields))
	for _, name := range sortedKeys(fields) {
		list = append(list, domain.Field{Name: name, Value: fields[name]})
	}
	return list
}

// asObject decodes a result frame into an object. A null frame, a frame of another shape, and a broken frame each
// fail with ProtocolError: the gateway maps only what the plugin sent as an object.
func asObject(raw json.RawMessage) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, ProtocolError{Rule: "the plugin reply has no result object"}
	}
	return obj, nil
}
