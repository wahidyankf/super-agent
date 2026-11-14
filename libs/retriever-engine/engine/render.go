package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// slotPattern matches a whole path segment or a whole header value that names a parameter, such as {id}.
var slotPattern = regexp.MustCompile(`^\{([a-z][a-z0-9_]*)\}$`)

// bodySlotPattern matches a whole string of a body that names a parameter, such as {{title}}.
var bodySlotPattern = regexp.MustCompile(`^\{\{([a-z][a-z0-9_]*)\}\}$`)

// request is one request ready to send: the method, the target path with its query, the headers that the spec sets,
// and the JSON body when the operation has one. The engine adds the authorization and accept headers when it sends.
type request struct {
	method string
	target string
	header http.Header
	body   []byte
}

// renderRequest builds the request of one operation from its validated arguments. The path slots and the query are
// encoded, a header value is checked, and extra replaces the query values of the same name, which is how the paging
// reads set their offset and cursor.
func renderRequest(basePath string, o spec.Operation, args domain.Args, extra url.Values) (request, error) {
	vals := args.Values()
	path, err := renderPath(o.Path, vals)
	if err != nil {
		return request{}, err
	}
	header, err := renderHeaders(o.Headers, vals)
	if err != nil {
		return request{}, err
	}
	body, err := renderBody(o, vals)
	if err != nil {
		return request{}, err
	}
	q := url.Values{}
	for _, p := range o.Params {
		if p.In != "query" {
			continue
		}
		if v, ok := vals[p.Name]; ok {
			q.Set(p.Name, textOf(v))
		}
	}
	for name, values := range extra {
		q[name] = append([]string(nil), values...)
	}
	target := basePath + path
	if enc := q.Encode(); enc != "" {
		target += "?" + enc
	}
	return request{method: o.Method, target: target, header: header, body: body}, nil
}

// renderPath fills the slots of a path template. A slot is one whole segment, and its value is one segment too: a value
// that would add a segment, leave its segment, or hold a control character is refused.
func renderPath(template string, vals map[string]any) (string, error) {
	segs := strings.Split(template, "/")
	for i, seg := range segs {
		if m := slotPattern.FindStringSubmatch(seg); m != nil {
			v, ok := vals[m[1]]
			if !ok {
				return "", domain.ErrMissingArgument
			}
			text := textOf(v)
			if err := checkSegment(text); err != nil {
				return "", err
			}
			segs[i] = url.PathEscape(text)
			continue
		}
		if strings.ContainsAny(seg, "{}") {
			return "", fmt.Errorf("%w: a path slot must fill one whole segment", spec.ErrInvalid)
		}
	}
	return strings.Join(segs, "/"), nil
}

// checkSegment refuses a value that cannot be one path segment: an empty value, a dot segment, a separator, an escape
// character, a control character, or text that is not UTF-8.
func checkSegment(v string) error {
	if v == "" || v == "." || v == ".." {
		return domain.ErrInvalidID
	}
	if strings.ContainsAny(v, `/\%`) || !utf8.ValidString(v) {
		return domain.ErrInvalidID
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return domain.ErrInvalidID
		}
	}
	return nil
}

// renderHeaders returns the headers that the spec sets. A constant is sent as it is. A whole {name} slot takes the
// value of its parameter, and a slot with no value is left out. A value with a control character is refused.
func renderHeaders(headers map[string]string, vals map[string]any) (http.Header, error) {
	h := http.Header{}
	for name, value := range headers {
		if m := slotPattern.FindStringSubmatch(value); m != nil {
			v, ok := vals[m[1]]
			if !ok {
				continue
			}
			text := textOf(v)
			if !headerSafe(text) {
				return nil, ErrInvalidHeader
			}
			h.Set(name, text)
			continue
		}
		if !headerSafe(value) {
			return nil, ErrInvalidHeader
		}
		h.Set(name, value)
	}
	return h, nil
}

// headerSafe reports whether a header value is UTF-8 and holds no control character except the tab.
func headerSafe(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r != '\t' && unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// renderBody builds the JSON body of an operation. A string that is exactly {{name}} takes the value of its parameter,
// and a parameter with no value leaves its key out. A string with a slot inside other text stays as it is. The
// encoder escapes every value, so a value never changes the shape of the body.
func renderBody(o spec.Operation, vals map[string]any) ([]byte, error) {
	if o.Body == nil {
		return nil, nil
	}
	out, _ := renderValue(o.Body, vals)
	return json.Marshal(out)
}

// renderValue renders one node of a body. It returns false when the node is a slot with no value.
func renderValue(v any, vals map[string]any) (any, bool) {
	switch x := v.(type) {
	case string:
		if m := bodySlotPattern.FindStringSubmatch(x); m != nil {
			val, ok := vals[m[1]]
			return val, ok
		}
		return x, true
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			if val, ok := renderValue(item, vals); ok {
				out[k] = val
			}
		}
		return out, true
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			if val, ok := renderValue(item, vals); ok {
				out = append(out, val)
			}
		}
		return out, true
	default:
		return x, true
	}
}

// textOf returns the text of an argument value, as it travels in a query or a header.
func textOf(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case bool:
		return strconv.FormatBool(x)
	case []string:
		return strings.Join(x, ",")
	default:
		return fmt.Sprint(x)
	}
}

// ExampleTarget returns the method and the target that a read example of the spec sends. The replay stub uses it to
// key each canned reply, so a test sees the same request that the engine would send. It binds the example request with
// the rules of the operation catalog, so an example that the catalog would refuse is refused here too.
func ExampleTarget(s *spec.Spec, o spec.Operation, values map[string]any) (string, string, error) {
	raw := make(map[string]any, len(values))
	for k, v := range values {
		raw[k] = plainValue(v)
	}
	args, err := operations.Bind(rowOf(o), raw)
	if err != nil {
		return "", "", err
	}
	req, err := renderRequest(s.Endpoint.BasePath, o, args, nil)
	if err != nil {
		return "", "", err
	}
	return req.method, req.target, nil
}

// rowOf returns the catalog row that the binder checks an operation of the spec with. A parameter with no minimum or
// maximum takes the widest range, which only an Int parameter reads.
func rowOf(o spec.Operation) operations.Row {
	row := operations.Row{ID: domain.OperationID(o.ID), Summary: o.Summary}
	for _, p := range o.Params {
		param := operations.Param{
			Name:     p.Name,
			Kind:     operations.Kind(p.Type),
			Required: p.Required || p.In == "path",
			Min:      math.MinInt,
			Max:      math.MaxInt,
			Values:   p.Values,
			Help:     p.Help,
		}
		if p.Default != nil {
			param.Default = plainValue(p.Default)
		}
		if p.Min != nil {
			param.Min = *p.Min
		}
		if p.Max != nil {
			param.Max = *p.Max
		}
		row.Params = append(row.Params, param)
	}
	return row
}
