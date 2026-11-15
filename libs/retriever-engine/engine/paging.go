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

// maxOffsetValue is one past the largest page offset: the offset is a non-negative integer below 2^31. The binder
// refuses a negative offset for an Offset parameter, and the loop below refuses an offset at or past this cap.
const maxOffsetValue = 1 << 31

// readOffset walks an offset read. It asks for pages of the limit argument, moves the offset on by each page, and stops
// at the first short page. At the page cap it returns the pages it has, with a continuation that names the next offset.
// With in: body the paging values travel in the JSON body instead of the query.
func (e *Engine) readOffset(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	pag := o.Pagination
	limit := args.Int(pag.LimitParam)
	if limit < 1 {
		return domain.Record{}, domain.ErrInvalidLimit
	}
	offset := args.Int(pag.OffsetParam)
	bodyMode := pag.In == "body"
	var records []domain.Record
	var last domain.Page
	for i := 0; i < e.pageCap(o); i++ {
		if offset >= maxOffsetValue {
			return domain.Record{}, domain.ErrInvalidOffset
		}
		var root any
		var err error
		if bodyMode {
			root, err = e.fetch(ctx, o, argsWith(args, map[string]any{
				pag.LimitParam:  limit,
				pag.OffsetParam: offset,
			}), nil)
		} else {
			extra := url.Values{
				pag.LimitParam:  {strconv.Itoa(limit)},
				pag.OffsetParam: {strconv.Itoa(offset)},
			}
			root, err = e.fetch(ctx, o, args, extra)
		}
		if err != nil {
			return domain.Record{}, err
		}
		items, err := pageItems(o, args, root, budget)
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
// the reply until that cursor is null or empty. A next cursor that repeats one already sent ends the read too, so a
// reply that never advances cannot run to the cap. At the page cap it returns the pages it has, with no continuation.
// With in: body the cursor travels in the JSON body instead of the query.
func (e *Engine) readCursor(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	pag := o.Pagination
	bodyMode := pag.In == "body"
	var records []domain.Record
	cursor := ""
	sent := map[string]bool{}
	for i := 0; i < e.pageCap(o); i++ {
		if sent[cursor] {
			break
		}
		sent[cursor] = true
		var root any
		var err error
		if bodyMode {
			vals := args.Values()
			if cursor != "" {
				vals[pag.CursorParam] = cursor
			}
			root, err = e.fetch(ctx, o, domain.NewArgs(vals), nil)
		} else {
			var extra url.Values
			if cursor != "" {
				extra = url.Values{pag.CursorParam: {cursor}}
			}
			root, err = e.fetch(ctx, o, args, extra)
		}
		if err != nil {
			return domain.Record{}, err
		}
		items, err := pageItems(o, args, root, budget)
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

// argsWith returns the args with the given values set, so one page of a body-paged read renders its paging values
// into the JSON body. The query keeps only the declared query parameters, so the values travel in the body alone.
func argsWith(args domain.Args, set map[string]any) domain.Args {
	vals := args.Values()
	for k, v := range set {
		vals[k] = v
	}
	return domain.NewArgs(vals)
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
