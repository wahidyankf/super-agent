package spec

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// The slot patterns of a spec. A path slot sits inside a path, a header slot is the whole value of a header, and a body
// slot is the whole string of a body value.
var (
	pathSlotPattern   = regexp.MustCompile(`\{([a-z][a-zA-Z0-9_]{0,63})\}`)
	headerSlotPattern = regexp.MustCompile(`^\{([a-z][a-zA-Z0-9_]{0,63})\}$`)
	bodySlotPattern   = regexp.MustCompile(`^\{\{([a-z][a-zA-Z0-9_]{0,63})\}\}$`)
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
	return append(fs, sharedSemanticFindings(s)...)
}

// sharedSemanticFindings checks every rule that both entries share: everything semanticFindings checks except the
// built-in kind, which only the user-folder entry refuses.
func sharedSemanticFindings(s *Spec) []Finding {
	var fs []Finding
	fs = append(fs, CheckAccessMatrix(s)...)
	fs = append(fs, CheckObjectFields(s)...)
	fs = append(fs, CheckResponseShape(s)...)
	fs = append(fs, CheckMerge(s)...)
	fs = append(fs, CheckTree(s)...)
	fs = append(fs, CheckFallback(s)...)
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

// checkPagination checks that each named paging parameter of a paged read is a parameter of the operation in the
// location the rule names: query, or body with in: body. A rule cannot mix locations. A page-all rule names no
// paging parameter at all, because the operation declares none, and its fixed page size is between 1 and 100.
func checkPagination(op Operation, base string, params map[string]Param) []Finding {
	if op.Pagination == nil {
		return nil
	}
	loc := op.Pagination.In
	if loc == "" {
		loc = "query"
	}
	if loc != "query" && loc != "body" {
		return []Finding{finding(base+"/pagination/in", "is neither query nor body")}
	}
	if op.Pagination.Style == "page-all" {
		var fs []Finding
		named := []struct{ field, value string }{
			{"cursor_param", op.Pagination.CursorParam},
			{"limit_param", op.Pagination.LimitParam},
			{"next_pointer", op.Pagination.NextPointer},
			{"offset_param", op.Pagination.OffsetParam},
		}
		for _, p := range named {
			if p.value != "" {
				fs = append(fs, finding(base+"/pagination/"+p.field, "names a paging parameter of a page-all rule"))
			}
		}
		if op.Pagination.PageSize < 1 || op.Pagination.PageSize > 100 {
			fs = append(fs, finding(base+"/pagination/page_size", "is not a page size from 1 to 100"))
		}
		return fs
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
		if p, ok := params[name]; !ok || p.In != loc {
			fs = append(fs, finding(base+"/pagination/"+field, "names no "+loc+" parameter of the operation"))
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
// field of its response. For a merge response the expectation may name the label field or a field of any list. For
// a tree response the expectation may name the computed parent and depth fields too.
func checkExamples(op Operation, base string, params map[string]Param) []Finding {
	var fs []Finding
	var mergeFields map[string]bool
	if op.Response.Merge != nil {
		mergeFields = mergeOutputFields(op)
	}
	var treeFields map[string]bool
	if op.Response.Tree != nil {
		treeFields = treeOutputFields(op)
	}
	for j, ex := range op.Examples {
		at := base + "/examples/" + strconv.Itoa(j)
		for key := range ex.Request {
			if _, ok := params[key]; !ok {
				fs = append(fs, finding(at+"/request/"+safeName(key), "names no parameter of the operation"))
			}
		}
		for key := range ex.Expect {
			if mergeFields != nil {
				if !mergeFields[key] {
					fs = append(fs, finding(at+"/expect/"+safeName(key), "names no field of the operation"))
				}
				continue
			}
			if treeFields != nil {
				if !treeFields[key] {
					fs = append(fs, finding(at+"/expect/"+safeName(key), "names no field of the operation"))
				}
				continue
			}
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

// maxObjectFields is the most sub-fields one nested record field holds.
const maxObjectFields = 32

// The limits of a merge block: two to eight lists, and a label of at most 32 characters.
const (
	minMergeLists = 2
	maxMergeLists = 8
	maxMergeLabel = 32
)

// The fixed rules of a nested record field. Each names the broken rule, never a value.
const (
	ruleObjectScalar = "is an object field with a scalar keyword"
	ruleObjectCount  = "does not hold 1 to 32 sub-fields"
	ruleObjectDepth  = "nests an object inside an object"
	// ruleFallbackPointer is the rule of a scalar field whose fallback pointer has no pointer to fall back from.
	ruleFallbackPointer = "names a fallback pointer with no pointer to fall back from"
	// The fixed rules of an open-key map with closed bool values. Each names the broken rule, never a value.
	ruleBoolMapScalar  = "is a bool map with a scalar keyword"
	ruleBoolMapDepth   = "nests a bool map inside an object"
	ruleBoolMapPointer = "is a bool map with no pointer"
)

// CheckObjectFields returns the findings of the nested record fields of a spec model: an object field takes no scalar
// keyword of its own, it holds 1 to 32 sub-fields, and a sub-field holds no object of its own, so a record nests one
// level only. The schema states the same rules for documents. The validation and the engine constructor both call it,
// so a model built by hand meets the same refusals as a document. Each finding names the field path and a fixed rule.
// The fields of each list of a merge meet the same rules at their own path.
func CheckObjectFields(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		base := "/operations/" + strconv.Itoa(i) + "/response/fields"
		var names []string
		for name := range op.Response.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fs = append(fs, checkObjectField(base+"/"+safeName(name), op.Response.Fields[name])...)
		}
		if op.Response.Merge != nil {
			for li, l := range op.Response.Merge.Lists {
				lbase := "/operations/" + strconv.Itoa(i) + "/response/merge/lists/" + strconv.Itoa(li) + "/fields"
				var lnames []string
				for name := range l.Fields {
					lnames = append(lnames, name)
				}
				sort.Strings(lnames)
				for _, name := range lnames {
					fs = append(fs, checkObjectField(lbase+"/"+safeName(name), l.Fields[name])...)
				}
			}
		}
	}
	return fs
}

// checkObjectField checks one response field. A scalar field passes with no finding, unless its fallback pointer
// has no pointer to fall back from. A bool-map field passes under its own rule.
func checkObjectField(at string, f Field) []Finding {
	if f.Object == nil && !f.BoolMap {
		if f.FallbackPointer != "" && f.Pointer == "" {
			return []Finding{finding(at+"/fallback_pointer", ruleFallbackPointer)}
		}
		return nil
	}
	if f.BoolMap {
		return checkBoolMapField(at, f)
	}
	if f.Pointer != "" || f.FallbackPointer != "" || f.Required || f.Format != "" || f.Convert != "" {
		return []Finding{finding(at, ruleObjectScalar)}
	}
	if len(f.Object) < 1 || len(f.Object) > maxObjectFields {
		return []Finding{finding(at+"/object", ruleObjectCount)}
	}
	var fs []Finding
	var names []string
	for name := range f.Object {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if f.Object[name].Object != nil {
			fs = append(fs, finding(at+"/object/"+safeName(name)+"/object", ruleObjectDepth))
		}
		if f.Object[name].BoolMap {
			fs = append(fs, finding(at+"/object/"+safeName(name)+"/bool_map", ruleBoolMapDepth))
		}
	}
	return fs
}

// checkBoolMapField checks one open-key map with closed bool values: it reads the reply object at its pointer, so
// the pointer is required, and it takes no format, converter, or object of its own.
func checkBoolMapField(at string, f Field) []Finding {
	if f.Pointer == "" {
		return []Finding{finding(at, ruleBoolMapPointer)}
	}
	if f.Format != "" || f.Convert != "" || f.Object != nil {
		return []Finding{finding(at, ruleBoolMapScalar)}
	}
	return nil
}

// ruleResponseShape is the rule of a response that names more than one reply shape. A response holds at most one of
// items, first, and merge, so naming two is refused.
const ruleResponseShape = "names more than one of items, first, and merge"

// The fixed rules of a merge block. Each names the broken rule, never a value.
const (
	ruleMergeCount      = "does not hold 2 to 8 lists"
	ruleMergeLabel      = "is not a label of lowercase letters and hyphens of at most 32 characters"
	ruleMergeLabelField = "is not a field name"
)

// The patterns of a merge block: a label of lowercase letters and hyphens, and a label field of the field-name form.
var (
	mergeLabelPattern = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	mergeFieldPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]{0,63}$`)
)

// CheckMerge returns the findings of the merge blocks of a spec model: two to eight lists, a label of lowercase
// letters and hyphens of at most 32 characters on each list, and a label field of the field-name form. The schema
// states the same rules for documents. The validation and the engine constructor both call it, so a model built by
// hand meets the same refusals as a document. Each finding names the field path and a fixed rule.
func CheckMerge(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		m := op.Response.Merge
		if m == nil {
			continue
		}
		base := "/operations/" + strconv.Itoa(i) + "/response/merge"
		if len(m.Lists) < minMergeLists || len(m.Lists) > maxMergeLists {
			fs = append(fs, finding(base+"/lists", ruleMergeCount))
		}
		if !mergeFieldPattern.MatchString(m.LabelField) {
			fs = append(fs, finding(base+"/label_field", ruleMergeLabelField))
		}
		for li, l := range m.Lists {
			if len(l.Label) > maxMergeLabel || !mergeLabelPattern.MatchString(l.Label) {
				fs = append(fs, finding(base+"/lists/"+strconv.Itoa(li)+"/label", ruleMergeLabel))
			}
		}
	}
	return fs
}

// mergeOutputFields returns the output fields of a merge operation: the label field and every field of every list.
func mergeOutputFields(op Operation) map[string]bool {
	out := map[string]bool{}
	if op.Response.Merge == nil {
		return out
	}
	out[op.Response.Merge.LabelField] = true
	for _, l := range op.Response.Merge.Lists {
		for name := range l.Fields {
			out[name] = true
		}
	}
	return out
}

// CheckResponseShape returns the findings of the responses of a spec model that name more than one reply shape. The
// schema states the same rule for documents. The validation and the engine constructor both call it, so a model built
// by hand meets the same refusal as a document. Each finding names the response path and a fixed rule.
func CheckResponseShape(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		shapes := 0
		if op.Response.Items != "" {
			shapes++
		}
		if op.Response.First != "" {
			shapes++
		}
		if op.Response.Merge != nil {
			shapes++
		}
		if shapes > 1 {
			fs = append(fs, finding("/operations/"+strconv.Itoa(i)+"/response", ruleResponseShape))
		}
	}
	return fs
}

// The fixed rules of a tree block. Each names the broken rule, never a value.
const (
	ruleTreeNeedsItems = "needs items"
	ruleTreeMaxDepth   = "is above the depth cap of 64"
	ruleTreeNeedsID    = "needs a required id field"
)

// CheckTree returns the findings of the tree blocks of a spec model: a tree rides on items, the depth cap is at
// most 64, and the response holds a required id field, because the parent of a node is the id of its parent node.
// The schema states the shape rules for documents. The validation and the engine constructor both call it, so a
// model built by hand meets the same refusals as a document. Each finding names the field path and a fixed rule.
func CheckTree(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		t := op.Response.Tree
		if t == nil {
			continue
		}
		base := "/operations/" + strconv.Itoa(i) + "/response/tree"
		if op.Response.Items == "" {
			fs = append(fs, finding(base, ruleTreeNeedsItems))
		}
		if t.MaxDepth > treeDepthCap {
			fs = append(fs, finding(base+"/max_depth", ruleTreeMaxDepth))
		}
		if id, ok := op.Response.Fields["id"]; !ok || !id.Required {
			fs = append(fs, finding(base, ruleTreeNeedsID))
		}
	}
	return fs
}

// treeOutputFields returns the output fields of a tree operation: every mapped field with the computed parent and
// depth fields, which no pointer reads.
func treeOutputFields(op Operation) map[string]bool {
	out := map[string]bool{}
	for name := range op.Response.Fields {
		out[name] = true
	}
	if op.Response.Tree != nil {
		out[op.Response.Tree.ParentField] = true
		out[op.Response.Tree.DepthField] = true
	}
	return out
}

// The fixed rules of a fallback block. Each names the broken rule, never a value.
const (
	ruleFallbackRead       = "needs a read operation"
	ruleFallbackShape      = "needs a read of one object without pages"
	ruleFallbackTarget     = "names no read operation of the spec"
	ruleFallbackChain      = "names an operation with a fallback of its own"
	ruleFallbackParams     = "names an operation with a parameter the first operation does not declare"
	ruleFallbackWhen       = "needs exactly one closed trigger"
	ruleFallbackField      = "names no field of the operation"
	ruleFallbackObject     = "names an object field"
	ruleFallbackStatus     = "is not an HTTP status"
	ruleFallbackFill       = "names no field of the operation"
	ruleFallbackFillTarget = "names no field of the fallback operation"
	ruleFallbackFillEmpty  = "names no output field"
)

// CheckFallback returns the findings of the fallback blocks of a spec model: the source and the target are reads of
// one object without pages, the target has no fallback of its own so a chain of two fallbacks is refused, the
// target declares only parameters the first operation also declares, when is exactly one closed trigger, and every
// fill field is a field of both operations. The schema states the document shape for documents. The validation and
// the engine constructor both call it, so a model built by hand meets the same refusals as a document. Each finding
// names the field path and a fixed rule. The triggers empty and status are owner proposals that the owner can still
// veto at review.
func CheckFallback(s *Spec) []Finding {
	index := make(map[string]int, len(s.Operations))
	for i, op := range s.Operations {
		if _, repeated := index[op.ID]; !repeated {
			index[op.ID] = i
		}
	}
	var fs []Finding
	for i, op := range s.Operations {
		if op.Fallback == nil {
			continue
		}
		fs = append(fs, checkFallbackOp(s, index, i, op)...)
	}
	return fs
}

// checkFallbackOp checks the fallback block of one operation against its own fields and the target operation.
func checkFallbackOp(s *Spec, index map[string]int, i int, op Operation) []Finding {
	var fs []Finding
	fb := op.Fallback
	base := "/operations/" + strconv.Itoa(i)
	if op.Access != "read" {
		fs = append(fs, finding(base+"/fallback", ruleFallbackRead))
	}
	if !isObjectRead(op) {
		fs = append(fs, finding(base+"/fallback", ruleFallbackShape))
	}
	k, ok := index[fb.Operation]
	if !ok || s.Operations[k].Access != "read" {
		return append(fs, finding(base+"/fallback/operation", ruleFallbackTarget))
	}
	target := s.Operations[k]
	if target.Fallback != nil {
		fs = append(fs, finding(base+"/fallback/operation", ruleFallbackChain))
	}
	if !isObjectRead(target) {
		fs = append(fs, finding(base+"/fallback/operation", ruleFallbackShape))
	}
	names := make(map[string]bool, len(op.Params))
	for _, p := range op.Params {
		names[p.Name] = true
	}
	for _, p := range target.Params {
		if !names[p.Name] {
			fs = append(fs, finding(base+"/fallback/operation", ruleFallbackParams))
			break
		}
	}
	fs = append(fs, checkFallbackWhen(op, base, fb)...)
	if len(fb.Fill) == 0 {
		fs = append(fs, finding(base+"/fallback/fill", ruleFallbackFillEmpty))
	}
	for _, name := range fb.Fill {
		if _, ok := op.Response.Fields[name]; !ok {
			fs = append(fs, finding(base+"/fallback/fill", ruleFallbackFill))
		} else if _, ok := target.Response.Fields[name]; !ok {
			fs = append(fs, finding(base+"/fallback/fill", ruleFallbackFillTarget))
		}
	}
	return fs
}

// checkFallbackWhen checks the trigger of a fallback: exactly one of the closed triggers empty and status is set,
// the empty field is a scalar field of the read, and every status is an HTTP status.
func checkFallbackWhen(op Operation, base string, fb *Fallback) []Finding {
	if (fb.When.Empty == "") == (len(fb.When.Status) == 0) {
		return []Finding{finding(base+"/fallback/when", ruleFallbackWhen)}
	}
	if fb.When.Empty != "" {
		f, ok := op.Response.Fields[fb.When.Empty]
		if !ok {
			return []Finding{finding(base+"/fallback/when/empty", ruleFallbackField)}
		}
		if f.Object != nil {
			return []Finding{finding(base+"/fallback/when/empty", ruleFallbackObject)}
		}
		return nil
	}
	var fs []Finding
	for j, status := range fb.When.Status {
		if status < 100 || status > 599 {
			fs = append(fs, finding(base+"/fallback/when/status/"+strconv.Itoa(j), ruleFallbackStatus))
		}
	}
	return fs
}

// isObjectRead reports whether the operation is an unpaged read of one object: no items list, no first item, no
// merge, no tree, and no pages. Only such a read takes or serves a fallback, so the fill fields of the target map
// onto the fields of the first record one by one.
func isObjectRead(op Operation) bool {
	return op.Pagination == nil && op.Response.Items == "" && op.Response.First == "" &&
		op.Response.Merge == nil && op.Response.Tree == nil
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
