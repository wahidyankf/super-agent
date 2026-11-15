package engine

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

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

// pageAllLimitKey and pageAllOffsetKey are the fixed paging keys of a page-all read. The operation declares no page
// parameters, so the engine sends these names itself, in the query or in the JSON body per the rule, mirroring the
// fixed page body of the Go reader.
const (
	pageAllLimitKey  = "limit"
	pageAllOffsetKey = "offset"
)

// pageAllMinDelay is the least time between two requests of a page-all read, mirroring the request pacing of the Go
// gateway the rule replaces. The engine waits the whole delay before every page after the first, and it tops a
// shorter 429 wait up to the delay; a longer wait already covers it. The test clock advances a wait at once.
const pageAllMinDelay = 100 * time.Millisecond

// readPageAll reads every page of a page-all read: pages of the rule's fixed size until a short page, with no
// continuation. At the page cap it fails with the page-cap error, mirroring the Go reader, instead of returning the
// pages it has. With in: body the paging keys join the JSON body; a body the rule declares stays, and the paging
// keys beat a declared key of the same name, so the walk cannot be steered off its pages.
func (e *Engine) readPageAll(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	pag := o.Pagination
	if pag.PageSize < 1 || pag.PageSize > 100 {
		return domain.Record{}, domain.ErrInvalidLimit
	}
	size := pag.PageSize
	var records []domain.Record
	for i := 0; ; i++ {
		if i == e.pageCap(o) {
			return domain.Record{}, domain.ErrPageCapReached
		}
		if i > 0 {
			if err := e.ports.Clock.Sleep(ctx, pageAllMinDelay); err != nil {
				return domain.Record{}, err
			}
		}
		offset := i * size
		var root any
		var err error
		if pag.In == "body" {
			root, err = e.fetch(ctx, withPageBody(o, offset, size), args, nil)
		} else {
			extra := url.Values{
				pageAllLimitKey:  {strconv.Itoa(size)},
				pageAllOffsetKey: {strconv.Itoa(offset)},
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
		if len(items) < size {
			break
		}
	}
	return domain.NewList(records), nil
}

// withPageBody returns the operation with the paging keys of one page-all page joined into its JSON body. The
// declared body stays, and the paging keys beat a declared key of the same name, so the declared document cannot
// move the walk off its pages.
func withPageBody(o spec.Operation, offset, size int) spec.Operation {
	body := map[string]any{
		pageAllOffsetKey: offset,
		pageAllLimitKey:  size,
	}
	for k, v := range o.Body {
		if k != pageAllOffsetKey && k != pageAllLimitKey {
			body[k] = v
		}
	}
	o.Body = body
	return o
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
