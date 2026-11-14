package spec

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// The slot patterns of a spec. A path slot sits inside a path, a header slot is the whole value of a header, and a body
// slot is the whole string of a body value.
var (
	pathSlotPattern   = regexp.MustCompile(`\{([a-z][a-z0-9_]{0,63})\}`)
	headerSlotPattern = regexp.MustCompile(`^\{([a-z][a-z0-9_]{0,63})\}$`)
	bodySlotPattern   = regexp.MustCompile(`^\{\{([a-z][a-z0-9_]{0,63})\}\}$`)
)

// semanticFindings checks the rules that a JSON Schema cannot state, because each one compares two fields of one
// document: the kind against the built-in sources, the operation ids against the kind, the slots against the parameters,
// the pagination and resolve names against the operations, and the examples against the fields. It runs only on a
// document that passed the schema.
func semanticFindings(s *Spec) []Finding {
	var fs []Finding
	if _, err := domain.BuiltinKinds().Kind(s.Kind); err == nil {
		fs = append(fs, finding("/kind", "is the kind of a built-in source"))
	}
	index := make(map[string]int, len(s.Operations))
	for i, op := range s.Operations {
		at := "/operations/" + strconv.Itoa(i) + "/id"
		if !strings.HasPrefix(op.ID, s.Kind+".") {
			fs = append(fs, finding(at, "does not start with the kind of the spec"))
		}
		if _, repeated := index[op.ID]; repeated {
			fs = append(fs, finding(at, "repeats an operation id"))
			continue
		}
		index[op.ID] = i
	}
	for i := range s.Operations {
		fs = append(fs, checkOperation(s, i, index)...)
	}
	return fs
}

// checkOperation checks one operation against its own parameters and the other operations of the spec.
func checkOperation(s *Spec, i int, index map[string]int) []Finding {
	op := s.Operations[i]
	base := "/operations/" + strconv.Itoa(i)
	var fs []Finding
	params := make(map[string]Param, len(op.Params))
	for j, p := range op.Params {
		if _, repeated := params[p.Name]; repeated {
			fs = append(fs, finding(paramAt(base, j)+"/name", "repeats a parameter name"))
			continue
		}
		params[p.Name] = p
	}
	fs = append(fs, checkPath(op, base, params)...)
	fs = append(fs, checkBody(op.Body, base+"/body", params)...)
	for name, value := range op.Headers {
		if reservedHeader(name) {
			fs = append(fs, finding(base+"/headers/"+safeName(name), reservedRule))
		}
		if m := headerSlotPattern.FindStringSubmatch(value); m != nil {
			if _, ok := params[m[1]]; !ok {
				fs = append(fs, finding(base+"/headers/"+safeName(name), "names no parameter of the operation"))
			}
		}
	}
	fs = append(fs, checkPagination(op, base, params)...)
	for j, p := range op.Params {
		if p.Resolve != nil {
			fs = append(fs, checkResolve(s, index, paramAt(base, j)+"/resolve", p.Resolve)...)
		}
	}
	if md := op.Response.Markdown; md != "" {
		if _, ok := op.Response.Fields[md]; !ok {
			fs = append(fs, finding(base+"/response/markdown", "names no field of the operation"))
		}
	}
	return append(fs, checkExamples(op, base, params)...)
}

// paramAt returns the path of parameter j of the operation at base.
func paramAt(base string, j int) string {
	return base + "/params/" + strconv.Itoa(j)
}

// checkPath checks that each {name} slot of the path has a path parameter of that name, and that each path parameter
// is used by the path.
func checkPath(op Operation, base string, params map[string]Param) []Finding {
	var fs []Finding
	used := make(map[string]bool)
	for _, m := range pathSlotPattern.FindAllStringSubmatch(op.Path, -1) {
		used[m[1]] = true
		if p, ok := params[m[1]]; !ok || p.In != "path" {
			fs = append(fs, finding(base+"/path", "has a slot with no path parameter of that name"))
		}
	}
	for j, p := range op.Params {
		if p.In == "path" && !used[p.Name] {
			fs = append(fs, finding(paramAt(base, j)+"/name", "is a path parameter that the path does not use"))
		}
	}
	return fs
}

// checkBody checks each {{name}} slot of a body value against the body parameters, at every depth of the body.
func checkBody(value any, at string, params map[string]Param) []Finding {
	switch v := value.(type) {
	case string:
		m := bodySlotPattern.FindStringSubmatch(v)
		if m == nil {
			return nil
		}
		if p, ok := params[m[1]]; ok && p.In == "body" {
			return nil
		}
		return []Finding{finding(at, "names no body parameter of the operation")}
	case map[string]any:
		var fs []Finding
		for key, child := range v {
			fs = append(fs, checkBody(child, at+"/"+safeName(key), params)...)
		}
		return fs
	case []any:
		var fs []Finding
		for j, child := range v {
			fs = append(fs, checkBody(child, at+"/"+strconv.Itoa(j), params)...)
		}
		return fs
	}
	return nil
}

// checkPagination checks that each named query parameter of a paged read is a query parameter of the operation.
func checkPagination(op Operation, base string, params map[string]Param) []Finding {
	if op.Pagination == nil {
		return nil
	}
	var fs []Finding
	names := map[string]string{
		"limit_param":  op.Pagination.LimitParam,
		"offset_param": op.Pagination.OffsetParam,
		"cursor_param": op.Pagination.CursorParam,
	}
	for field, name := range names {
		if name == "" {
			continue
		}
		if p, ok := params[name]; !ok || p.In != "query" {
			fs = append(fs, finding(base+"/pagination/"+field, "names no query parameter of the operation"))
		}
	}
	return fs
}

// checkResolve checks that a resolve names a read operation of the spec and a field that the read returns.
func checkResolve(s *Spec, index map[string]int, at string, r *Resolve) []Finding {
	k, ok := index[r.Operation]
	if !ok || s.Operations[k].Access != "read" {
		return []Finding{finding(at+"/operation", "names no read operation of the spec")}
	}
	if _, ok := s.Operations[k].Response.Fields[r.Field]; !ok {
		return []Finding{finding(at+"/field", "names no field of the read operation")}
	}
	return nil
}

// checkExamples checks that each example request names a parameter of the operation, and that each expectation names a
// field of its response.
func checkExamples(op Operation, base string, params map[string]Param) []Finding {
	var fs []Finding
	for j, ex := range op.Examples {
		at := base + "/examples/" + strconv.Itoa(j)
		for key := range ex.Request {
			if _, ok := params[key]; !ok {
				fs = append(fs, finding(at+"/request/"+safeName(key), "names no parameter of the operation"))
			}
		}
		for key := range ex.Expect {
			if _, ok := op.Response.Fields[key]; !ok {
				fs = append(fs, finding(at+"/expect/"+safeName(key), "names no field of the operation"))
			}
		}
		for key := range ex.Response.Headers {
			if reservedHeader(key) {
				fs = append(fs, finding(at+"/response/headers/"+safeName(key), reservedRule))
			}
		}
	}
	return fs
}

// reservedRule is the rule of a header that the engine sets itself, so no spec may name it.
const reservedRule = "names a header that the engine sets itself"

// reservedHeaders are the header names that no spec may set. The engine alone sets the key, the host, the length of the
// body, and the cookies, so a spec that names one is refused. The names compare without case.
var reservedHeaders = map[string]bool{
	"authorization":  true,
	"cookie":         true,
	"host":           true,
	"content-length": true,
}

// reservedHeader reports whether a header name is one that no spec may set.
func reservedHeader(name string) bool {
	return reservedHeaders[strings.ToLower(name)]
}
