package spec

import "strconv"

// truncatedField is the output field that the truncated block computes on each item of a cursor read. An example may
// expect it exactly when the response declares the block, like the computed parent and depth fields of a tree read.
const truncatedField = "truncated"

// The fixed rules of the page-end block: the last flag, the cut to the limit, and the computed truncated flag. Each
// names the broken rule, never a value.
const (
	// rulePageEndStyle refuses a last flag or a cut on a walk that names no page parameters.
	rulePageEndStyle = "needs an offset or cursor rule"
	// ruleCutToLimitParam refuses a cut that names no parameter of the operation.
	ruleCutToLimitParam = "names no parameter of the operation"
	// ruleCutToLimitKind refuses a cut that names a parameter of another kind.
	ruleCutToLimitKind = "is not a Limit or Int parameter"
	// ruleTruncatedRule refuses a truncated block without the cursor rule that declares the last flag.
	ruleTruncatedRule = "needs a cursor rule that declares last_pointer"
)

// CheckPageEnd returns the findings of the page-end blocks of a spec model: the last flag and the cut ride an offset
// or cursor walk, the cut names a Limit or Int parameter of the operation, and the truncated block rides a cursor
// rule that declares the last flag. The schema states the document shape for documents. The validation and the engine
// constructor both call it, so a model built by hand meets the same refusals as a document. Each finding names the
// field path and a fixed rule.
func CheckPageEnd(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		fs = append(fs, checkPageEnd(op, "/operations/"+strconv.Itoa(i))...)
	}
	return fs
}

// checkPageEnd checks the page-end block of one operation against its own parameters and pagination rule.
func checkPageEnd(op Operation, base string) []Finding {
	var fs []Finding
	pag := op.Pagination
	if pag != nil && pag.Style != "offset" && pag.Style != "cursor" {
		if pag.LastPointer != "" {
			fs = append(fs, finding(base+"/pagination/last_pointer", rulePageEndStyle))
		}
		if pag.CutToLimit != "" {
			fs = append(fs, finding(base+"/pagination/cut_to_limit", rulePageEndStyle))
		}
	}
	if pag != nil && (pag.Style == "offset" || pag.Style == "cursor") && pag.CutToLimit != "" {
		fs = append(fs, checkCutToLimit(op, base, pag)...)
	}
	if op.Response.Truncated != nil {
		if pag == nil || pag.Style != "cursor" || pag.LastPointer == "" {
			fs = append(fs, finding(base+"/response/truncated", ruleTruncatedRule))
		}
	}
	return fs
}

// checkCutToLimit checks that the cut names a Limit or Int parameter of the operation, which the engine reads for the
// page size to keep. The named parameter travels nowhere: its value alone cuts the page.
func checkCutToLimit(op Operation, base string, pag *Pagination) []Finding {
	for j := range op.Params {
		if op.Params[j].Name != pag.CutToLimit {
			continue
		}
		if op.Params[j].Type != "Limit" && op.Params[j].Type != "Int" {
			return []Finding{finding(base+"/pagination/cut_to_limit", ruleCutToLimitKind)}
		}
		return nil
	}
	return []Finding{finding(base+"/pagination/cut_to_limit", ruleCutToLimitParam)}
}
