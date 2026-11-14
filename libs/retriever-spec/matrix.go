package spec

import (
	"strconv"
	"strings"
)

// The limits of the body of a POST read. The rendered body of a read is at most MaxPostBodyBytes, and it nests at most
// MaxPostBodyDepth levels, counting the top-level object as the first level. No spec field raises or lowers them.
const (
	// MaxPostBodyBytes is the size limit of the JSON body of a POST read: 64 KiB.
	MaxPostBodyBytes = 64 << 10
	// MaxPostBodyDepth is the deepest nesting of the JSON body of a POST read.
	MaxPostBodyDepth = 8
)

// The fixed rules of the access matrix and of a POST read. Each names the broken rule, never a value.
const (
	ruleMatrix      = "is a method that the access label does not allow with this marker"
	ruleBodyMissing = "is missing the body that a POST read sends"
	ruleResolve     = "is not allowed on a POST read, which claims no side effect"
	ruleDepth       = "nests more than 8 levels"
)

// AccessMethodAllowed reports whether a label, an HTTP method, and the POST marker form a pair that the access matrix of
// ADR 002 allows. A GET or HEAD read takes no marker. A POST read takes the marker, and a POST without it is a write or an
// admin operation. PUT, PATCH, and DELETE take a write or an admin label only. Any other triple is refused. The engine calls
// this in its constructor, so a model that breaks the matrix is refused before any request.
func AccessMethodAllowed(access, method string, readOverPost bool) bool {
	switch access {
	case "read":
		switch method {
		case "GET", "HEAD":
			return !readOverPost
		case "POST":
			return readOverPost
		}
		return false
	case "write", "admin":
		switch method {
		case "POST", "PUT", "PATCH", "DELETE":
			return !readOverPost
		}
	}
	return false
}

// CheckAccessMatrix returns the findings of a spec model against the access matrix, and against the rules that make a POST
// read a read. The validation and the engine constructor both call it, so a model built by hand meets the same refusals as
// a document. Each finding names the field path and a fixed rule.
func CheckAccessMatrix(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		base := "/operations/" + strconv.Itoa(i)
		if !AccessMethodAllowed(op.Access, op.Method, op.ReadOverPost) {
			fs = append(fs, finding(base+"/method", ruleMatrix))
		}
		if op.ReadOverPost {
			fs = append(fs, checkPostRead(op, base)...)
		}
	}
	return fs
}

// checkPostRead checks the rules of a POST read that the matrix does not state: the body is present, the spec sets no
// content type, no parameter resolves a name, and the body nests within the depth limit. The schema and the slot rules
// check the rest of the body.
func checkPostRead(op Operation, base string) []Finding {
	var fs []Finding
	if op.Body == nil {
		fs = append(fs, finding(base+"/body", ruleBodyMissing))
	}
	for name := range op.Headers {
		if strings.EqualFold(name, "Content-Type") {
			fs = append(fs, finding(base+"/headers/"+safeName(name), reservedRule))
		}
	}
	for j, p := range op.Params {
		if p.Resolve != nil {
			fs = append(fs, finding(paramAt(base, j)+"/resolve", ruleResolve))
		}
	}
	if BodyDepth(op.Body) > MaxPostBodyDepth {
		fs = append(fs, finding(base+"/body", ruleDepth))
	}
	return fs
}

// BodyDepth returns how many levels a decoded JSON value nests. A scalar nests none, and each object or list adds one level,
// so the top-level object of a body is level one. The engine calls it on the rendered body, where a list that a slot fills
// adds a level the template does not show.
func BodyDepth(v any) int {
	var children []any
	switch x := v.(type) {
	case map[string]any:
		for _, child := range x {
			children = append(children, child)
		}
	case []any:
		children = x
	default:
		return 0
	}
	deepest := 0
	for _, child := range children {
		if d := BodyDepth(child); d > deepest {
			deepest = d
		}
	}
	return deepest + 1
}

// bodySlotNames returns the names of the whole {{name}} slots of a body, at every depth, in sorted order.
func bodySlotNames(v any, out map[string]bool) {
	switch x := v.(type) {
	case string:
		if m := bodySlotPattern.FindStringSubmatch(x); m != nil {
			out[m[1]] = true
		}
	case map[string]any:
		for _, child := range x {
			bodySlotNames(child, out)
		}
	case []any:
		for _, child := range x {
			bodySlotNames(child, out)
		}
	}
}
