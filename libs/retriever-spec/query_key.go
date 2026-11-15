package spec

import (
	"regexp"
	"strconv"
)

// queryKeyPattern is the closed shape of a wire alias: it starts with a letter and holds letters, digits, and
// underscores only, which covers camelCase wire names such as maxResults and startAt.
var queryKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

// The fixed rules of a query alias. Each names the broken rule, never a value.
const (
	// ruleQueryKeyShape refuses an alias outside the closed shape of a wire key.
	ruleQueryKeyShape = "is not a wire key of letters, digits, and underscores starting with a letter"
	// ruleQueryKeyLocation refuses an alias on a parameter that does not travel in the query.
	ruleQueryKeyLocation = "is a query_key on a parameter that is not in the query"
	// ruleQueryKeySame refuses an alias that repeats the parameter name.
	ruleQueryKeySame = "repeats the parameter name"
	// ruleQueryKeyCollision refuses an alias that another query key of the operation already takes.
	ruleQueryKeyCollision = "repeats a query key of the operation"
)

// CheckQueryKeys returns the findings of the wire aliases of a spec model: only an in: query parameter takes one,
// the alias fits the closed shape, it never repeats the parameter name, and no two query keys of one operation share
// a wire key. The schema states the document shape for documents. The validation and the engine constructor both call
// it, so a model built by hand meets the same refusals as a document. Each finding names the field path and a fixed
// rule.
func CheckQueryKeys(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		base := "/operations/" + strconv.Itoa(i)
		for j, p := range op.Params {
			if p.QueryKey == "" {
				continue
			}
			at := paramAt(base, j) + "/query_key"
			if p.In != "query" {
				fs = append(fs, finding(at, ruleQueryKeyLocation))
				continue
			}
			if !queryKeyPattern.MatchString(p.QueryKey) {
				fs = append(fs, finding(at, ruleQueryKeyShape))
				continue
			}
			if p.QueryKey == p.Name {
				fs = append(fs, finding(at, ruleQueryKeySame))
				continue
			}
			seen := false
			for _, q := range op.Params[:j] {
				if q.In != "query" {
					continue
				}
				other := q.Name
				if q.QueryKey != "" {
					other = q.QueryKey
				}
				if other == p.QueryKey {
					seen = true
					break
				}
			}
			if seen {
				fs = append(fs, finding(at, ruleQueryKeyCollision))
			}
		}
	}
	return fs
}
