package spec

import "testing"

// TestCheckFallbackRefusesHandBuiltModels checks the Go rule on fallback blocks that bypass the schema: a target
// that is missing or not a read, a chained target, a target with a parameter the first operation does not declare,
// a trigger that names no scalar field, a status outside the HTTP range, and a fill that names no field of either
// operation are each refused at their own path with a fixed rule. A sound fallback passes with no finding.
func TestCheckFallbackRefusesHandBuiltModels(t *testing.T) {
	fields := map[string]Field{
		"title":    {Pointer: "/title"},
		"markdown": {Pointer: "/markdown"},
	}
	targetFields := map[string]Field{"markdown": {Pointer: "/markdown"}}
	sound := func() *Fallback {
		return &Fallback{Operation: "example-wiki.docs.export",
			When: FallbackWhen{Empty: "markdown"}, Fill: []string{"markdown"}}
	}
	params := []Param{{Name: "id", Type: "ID", In: "path", Required: true}}
	target := func(op Operation) *Spec {
		return &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields}, Fallback: sound()},
			op,
		}}
	}
	targetOp := func(access string, fb *Fallback, params []Param) Operation {
		return Operation{ID: "example-wiki.docs.export", Access: access, Params: params,
			Response: Response{Fields: targetFields}, Fallback: fb}
	}
	cases := []struct {
		name string
		spec *Spec
		path string
		rule string
	}{
		{"a sound fallback", target(targetOp("read", nil, params)), "", ""},
		{"a missing target", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields}, Fallback: sound()},
			Operation{ID: "example-wiki.docs.gone", Access: "read", Params: params,
				Response: Response{Fields: targetFields}}}}, "/operations/0/fallback/operation", ruleFallbackTarget},
		{"a target that is a write", target(targetOp("write", nil, params)), "/operations/0/fallback/operation", ruleFallbackTarget},
		{"a chained target", target(targetOp("read", sound(), params)), "/operations/0/fallback/operation", ruleFallbackChain},
		{"a target with an extra parameter", target(targetOp("read", nil,
			append(append([]Param(nil), params...), Param{Name: "rev", Type: "ID", In: "query"}))),
			"/operations/0/fallback/operation", ruleFallbackParams},
		{"a trigger on no field", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields},
				Fallback: &Fallback{Operation: "example-wiki.docs.export",
					When: FallbackWhen{Empty: "absent"}, Fill: []string{"markdown"}}},
			targetOp("read", nil, params)}}, "/operations/0/fallback/when/empty", ruleFallbackField},
		{"a trigger on an object field", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: map[string]Field{"user": {Object: map[string]Field{
					"name": {Pointer: "/name"}}}}},
				Fallback: &Fallback{Operation: "example-wiki.docs.export",
					When: FallbackWhen{Empty: "user"}, Fill: []string{"markdown"}}},
			Operation{ID: "example-wiki.docs.export", Access: "read", Params: params,
				Response: Response{Fields: targetFields}}}}, "/operations/0/fallback/when/empty", ruleFallbackObject},
		{"a status outside the HTTP range", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields},
				Fallback: &Fallback{Operation: "example-wiki.docs.export",
					When: FallbackWhen{Status: []int{99}}, Fill: []string{"markdown"}}},
			targetOp("read", nil, params)}}, "/operations/0/fallback/when/status/0", ruleFallbackStatus},
		{"a fill on no field of the read", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields},
				Fallback: &Fallback{Operation: "example-wiki.docs.export",
					When: FallbackWhen{Empty: "markdown"}, Fill: []string{"absent"}}},
			targetOp("read", nil, params)}}, "/operations/0/fallback/fill", ruleFallbackFill},
		{"a fill on no field of the target", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields},
				Fallback: &Fallback{Operation: "example-wiki.docs.export",
					When: FallbackWhen{Empty: "markdown"}, Fill: []string{"title"}}},
			targetOp("read", nil, params)}}, "/operations/0/fallback/fill", ruleFallbackFillTarget},
		{"a fallback on a write", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "write", Params: params,
				Response: Response{Fields: fields}, Fallback: sound()},
			targetOp("read", nil, params)}}, "/operations/0/fallback", ruleFallbackRead},
		{"a fallback on a paged read", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Items: "/items", Fields: fields}, Fallback: sound()},
			targetOp("read", nil, params)}}, "/operations/0/fallback", ruleFallbackShape},
		{"a fallback to a paged read", &Spec{Operations: []Operation{
			{ID: "example-wiki.docs.meta", Access: "read", Params: params,
				Response: Response{Fields: fields}, Fallback: sound()},
			Operation{ID: "example-wiki.docs.export", Access: "read", Params: params,
				Response: Response{Items: "/items", Fields: targetFields}}}}, "/operations/0/fallback/operation", ruleFallbackShape},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := CheckFallback(c.spec)
			if c.path == "" {
				if len(fs) != 0 {
					t.Fatalf("a sound fallback has findings: %v", fs)
				}
				return
			}
			found := false
			for _, f := range fs {
				if f.Code != CodeSpecInvalid {
					t.Fatalf("finding code = %q; want %q", f.Code, CodeSpecInvalid)
				}
				if f.Path == c.path && f.Rule == c.rule {
					found = true
				}
			}
			if !found {
				t.Fatalf("no finding at %q with rule %q; findings: %v", c.path, c.rule, fs)
			}
		})
	}
}
