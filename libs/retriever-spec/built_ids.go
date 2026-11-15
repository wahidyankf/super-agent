package spec

import (
	"sort"
	"strconv"
)

// The fixed rules of a built ID field. Each names the broken rule, never a value.
const (
	// ruleBuiltSource refuses a field that reads more than one key: one pointer or one built key, never two.
	ruleBuiltSource = "names more than one of pointer, project_key, issue_id, collection_id, and browse_url"
	// ruleBuiltScalar refuses a built field with a scalar keyword of its own: no fallback pointer, no format, and
	// no converter.
	ruleBuiltScalar = "is a built field with a scalar keyword"
)

// CheckBuiltFields returns the findings of the built ID fields of a spec model: a field reads one key only, and a
// built field takes no fallback pointer, format, or converter of its own. The schema states the same shape for
// documents. The validation and the engine constructor both call it, so a model built by hand meets the same
// refusals as a document. Each finding names the field path and a fixed rule. The fields of each list of a merge
// meet the same rules at their own path, and so does each sub-field of a nested record.
func CheckBuiltFields(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		base := "/operations/" + strconv.Itoa(i) + "/response/fields"
		var names []string
		for name := range op.Response.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fs = append(fs, checkBuiltField(base+"/"+safeName(name), op.Response.Fields[name])...)
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
					fs = append(fs, checkBuiltField(lbase+"/"+safeName(name), l.Fields[name])...)
				}
			}
		}
	}
	return fs
}

// checkBuiltField checks one response field for the built ID rules. A scalar field without a built key passes, and
// so do an object field and a bool-map field; only the sub-fields of an object field are checked for the same rules
// at their dotted path.
func checkBuiltField(at string, f Field) []Finding {
	if f.Object != nil {
		var fs []Finding
		var names []string
		for name := range f.Object {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fs = append(fs, checkBuiltField(at+"/object/"+safeName(name), f.Object[name])...)
		}
		return fs
	}
	if f.BoolMap {
		return nil
	}
	sources := 0
	for _, source := range []string{f.Pointer, f.ProjectKey, f.IssueID, f.CollectionID, f.BrowseURL} {
		if source != "" {
			sources++
		}
	}
	built := f.ProjectKey != "" || f.IssueID != "" || f.CollectionID != "" || f.BrowseURL != ""
	if sources > 1 {
		return []Finding{finding(at, ruleBuiltSource)}
	}
	if built && (f.FallbackPointer != "" || f.Format != "" || f.Convert != "") {
		return []Finding{finding(at, ruleBuiltScalar)}
	}
	return nil
}
