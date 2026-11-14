package engine

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// pageCap returns the most pages one read may take. The built-in cap is 20. A spec may lower it, at the limits or at the
// operation, and never raise it.
func (e *Engine) pageCap(o spec.Operation) int {
	return capInt(capInt(maxPages, e.model.Limits.MaxPages), o.Pagination.MaxPages)
}

// readOffset walks an offset read. It asks for pages of the limit argument, moves the offset on by each page, and stops
// at the first short page. At the page cap it returns the pages it has, with a continuation that names the next offset.
func (e *Engine) readOffset(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	pag := o.Pagination
	limit := args.Int(pag.LimitParam)
	if limit < 1 {
		return domain.Record{}, domain.ErrInvalidLimit
	}
	offset := args.Int(pag.OffsetParam)
	var records []domain.Record
	var last domain.Page
	for i := 0; i < e.pageCap(o); i++ {
		extra := url.Values{
			pag.LimitParam:  {strconv.Itoa(limit)},
			pag.OffsetParam: {strconv.Itoa(offset)},
		}
		root, err := e.fetch(ctx, o, args, extra)
		if err != nil {
			return domain.Record{}, err
		}
		items, err := listItems(o, root, budget)
		if err != nil {
			return domain.Record{}, err
		}
		records = append(records, items...)
		last = domain.NewPage(offset, limit, len(items))
		if len(items) < limit {
			break
		}
		offset += len(items)
	}
	return domain.NewList(records).WithPage(last), nil
}

// readCursor walks a cursor read. It sends the cursor of each page after the first, and it follows the next cursor of
// the reply until that cursor is null or empty. At the page cap it returns the pages it has, with no continuation.
func (e *Engine) readCursor(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	pag := o.Pagination
	var records []domain.Record
	cursor := ""
	for i := 0; i < e.pageCap(o); i++ {
		var extra url.Values
		if cursor != "" {
			extra = url.Values{pag.CursorParam: {cursor}}
		}
		root, err := e.fetch(ctx, o, args, extra)
		if err != nil {
			return domain.Record{}, err
		}
		items, err := listItems(o, root, budget)
		if err != nil {
			return domain.Record{}, err
		}
		records = append(records, items...)
		next, found := pointerValue(root, pag.NextPointer)
		cursor, err = cursorText(next, found)
		if err != nil {
			return domain.Record{}, err
		}
		if cursor == "" {
			break
		}
	}
	return domain.NewList(records), nil
}

// cursorText returns the text of a next cursor. A missing or null cursor is the empty text, which ends the read. A cursor
// is a string or a whole number, and any other value is an unexpected response.
func cursorText(next any, found bool) (string, error) {
	if !found || next == nil {
		return "", nil
	}
	switch x := next.(type) {
	case string:
		return x, nil
	case json.Number:
		return x.String(), nil
	default:
		return "", domain.ErrUnexpectedResponse
	}
}
