package unit_test

import (
	"fmt"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// modelOf returns the model of the bundled example, which a test then changes in code.
func modelOf(t *testing.T) *spec.Spec {
	t.Helper()
	return requireAccepted(t, exampleWiki).Spec
}

// requireFindingsNamePathAndRule fails the test unless the check returns at least one finding and every finding names a
// field path and a fixed rule.
func requireFindingsNamePathAndRule(t *testing.T, fs []spec.Finding) {
	t.Helper()
	if len(fs) == 0 {
		t.Fatal("the check returns no finding; want at least one")
	}
	for _, f := range fs {
		if f.Path == "" || f.Rule == "" {
			t.Fatalf("finding %+v names no path or no rule", f)
		}
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: The access matrix check names an admin label over GET at the method
func TestAccessMatrixCheckNamesAnAdminLabelOverGETAtTheMethod(t *testing.T) {
	s := modelOf(t)
	s.Operations[0].Access = "admin"
	s.Operations[0].Method = "GET"
	fs := spec.CheckAccessMatrix(s)
	if len(fs) != 1 || fs[0].Path != "/operations/0/method" {
		t.Fatalf("findings = %+v; want one finding at /operations/0/method", fs)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: The access matrix check names a read over POST that has no body at the body
func TestAccessMatrixCheckNamesReadOverPostWithNoBodyAtTheBody(t *testing.T) {
	s := modelOf(t)
	s.Operations[0].Method = "POST"
	s.Operations[0].ReadOverPost = true
	s.Operations[0].Body = nil
	fs := spec.CheckAccessMatrix(s)
	if len(fs) != 1 || fs[0].Path != "/operations/0/body" {
		t.Fatalf("findings = %+v; want one finding at /operations/0/body", fs)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: The object field check names a fallback pointer on a field that has no pointer to fall back from
func TestObjectFieldCheckNamesFallbackPointerWithNoPointer(t *testing.T) {
	s := modelOf(t)
	s.Operations[0].Response.Fields["id"] = spec.Field{FallbackPointer: "/other"}
	fs := spec.CheckObjectFields(s)
	if len(fs) != 1 || fs[0].Path != "/operations/0/response/fields/id/fallback_pointer" {
		t.Fatalf("findings = %+v; want one finding at /operations/0/response/fields/id/fallback_pointer", fs)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: A check of a hand-built model that breaks one rule returns a finding that names a path and a fixed rule
func TestCheckOfHandBuiltModelThatBreaksOneRuleReturnsFindingNamingPathAndRule(t *testing.T) {
	cases := []struct {
		name  string
		build func(s *spec.Spec) []spec.Finding
	}{
		{"a tree with a max_depth above 64", func(s *spec.Spec) []spec.Finding {
			s.Operations[0].Response.Tree = &spec.Tree{Children: "/children", ParentField: "parent", DepthField: "depth", MaxDepth: 65}
			return spec.CheckTree(s)
		}},
		{"a merge with a list count out of its range", func(s *spec.Spec) []spec.Finding {
			s.Operations[0].Response.Merge = &spec.Merge{LabelField: "source", Lists: []spec.MergeList{{Items: "/a", Label: "a"}}}
			return spec.CheckMerge(s)
		}},
		{"a nested record with 33 sub-fields", func(s *spec.Spec) []spec.Finding {
			object := make(map[string]spec.Field, 33)
			for i := 0; i < 33; i++ {
				object[fmt.Sprintf("f%02d", i)] = spec.Field{Pointer: "/x"}
			}
			s.Operations[0].Response.Fields["user"] = spec.Field{Object: object}
			return spec.CheckObjectFields(s)
		}},
		{"a merge pair in a response shape", func(s *spec.Spec) []spec.Finding {
			s.Operations[0].Response.Merge = &spec.Merge{LabelField: "source", Lists: []spec.MergeList{{Items: "/a", Label: "a"}, {Items: "/b", Label: "b"}}}
			return spec.CheckResponseShape(s)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireFindingsNamePathAndRule(t, c.build(modelOf(t)))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: The access matrix check returns no finding for the bundled example model
func TestAccessMatrixCheckReturnsNoFindingForBundledExampleModel(t *testing.T) {
	if fs := spec.CheckAccessMatrix(modelOf(t)); len(fs) != 0 {
		t.Fatalf("findings = %+v; want none", fs)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/hand-built-models.feature: The bundled mapping check returns a finding that names a field path and a fixed rule for a broken catalog mapping
func TestBundledMappingCheckNamesFieldPathAndRuleForBrokenCatalogMapping(t *testing.T) {
	s := spec.ParseBundled([]byte(bundledJiraStub)).Spec
	if s == nil {
		t.Fatal("the bundled stub is refused")
	}
	s.Operations[1].CatalogID = "jira.no-such-operation"
	fs := spec.CheckBundledMapping(s)
	requireFindingsNamePathAndRule(t, fs)
	if !hasFindingPath(fs, "/operations/1/catalog_id") {
		t.Fatalf("findings = %+v; want one at /operations/1/catalog_id", fs)
	}
}

// hasFindingPath reports whether a list of findings holds one at exactly path.
func hasFindingPath(fs []spec.Finding, path string) bool {
	for _, f := range fs {
		if f.Path == path {
			return true
		}
	}
	return false
}
