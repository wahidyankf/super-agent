package graphql

import (
	"errors"
	"strings"
	"testing"
)

// TestMeasureQuery pins the depth and cost rules of serve-graphql.feature. Depth counts the
// selection sets on the deepest path. A field that calls Outline costs 40 at the operation
// root, every other field costs 1, and a fragment spread or inline fragment costs 1.
func TestMeasureQuery(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantDepth int
		wantCost  int
		wantErr   error
	}{
		{
			name:      "one outline field with one child field",
			query:     `{ collections { id name } }`,
			wantDepth: 2,
			wantCost:  42,
		},
		{
			name:      "two outline fields each with one id field cost 82",
			query:     `{ collections { id } collectionTree(id: "col-1") { id } }`,
			wantDepth: 2,
			wantCost:  82,
		},
		{
			name:    "three outline fields each with one id field cost 123 and are refused",
			query:   `{ collections { id } collectionTree(id: "col-1") { id } document(id: "doc-1") { id } }`,
			wantErr: errComplexityLimit,
		},
		{
			name:      "ten plain fields are within the limit",
			query:     `{ __typename __typename __typename __typename __typename __typename __typename __typename __typename __typename }`,
			wantDepth: 1,
			wantCost:  10,
		},
		{
			name:    "150 plain fields are refused",
			query:   "{" + strings.Repeat(" __typename", 150) + " }",
			wantErr: errComplexityLimit,
		},
		{
			name:      "five levels are within the depth limit",
			query:     `{ a { b { c { d { e } } } } }`,
			wantDepth: 5,
			wantCost:  5,
		},
		{
			name:    "six levels are refused",
			query:   `{ a { b { c { d { e { f } } } } } }`,
			wantErr: errDepthLimit,
		},
		{
			name:    "twelve levels are refused",
			query:   `{ a { b { c { d { e { f { g { h { i { j { k { l } } } } } } } } } } } }`,
			wantErr: errDepthLimit,
		},
		{
			name:    "schema introspection is refused",
			query:   `{ __schema { types { name } } }`,
			wantErr: errIntrospection,
		},
		{
			name:    "type introspection is refused",
			query:   `{ __type(name: "Query") { name } }`,
			wantErr: errIntrospection,
		},
		{
			name:      "typename at the root is allowed",
			query:     `{ __typename }`,
			wantDepth: 1,
			wantCost:  1,
		},
		{
			name:      "a root fragment spread does not hide an outline field",
			query:     `query { ...F } fragment F on Query { collectionTree(id: "c") { id } }`,
			wantDepth: 3,
			wantCost:  42,
		},
		{
			name:      "a root inline fragment does not hide an outline field",
			query:     `{ ... on Query { document(id: "d") { id } } }`,
			wantDepth: 3,
			wantCost:  42,
		},
		{
			name:    "a fragment cycle is stopped by the depth limit",
			query:   `{ ...A } fragment A on Query { ...A }`,
			wantErr: errDepthLimit,
		},
		{
			name:    "a syntax error is passed on to the executor",
			query:   `{ collections {`,
			wantErr: errSyntax,
		},
		{
			name:      "an unknown fragment is left for the executor to report",
			query:     `{ ...Missing }`,
			wantDepth: 1,
			wantCost:  1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			depth, cost, err := measureQuery(tc.query)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("measureQuery error = %v, want none", err)
				}
				if depth != tc.wantDepth {
					t.Errorf("depth = %d, want %d", depth, tc.wantDepth)
				}
				if cost != tc.wantCost {
					t.Errorf("cost = %d, want %d", cost, tc.wantCost)
				}
				return
			}
			// A refused query stops at the first point over the limit, so its partial depth and cost are not pinned.
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("measureQuery error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
