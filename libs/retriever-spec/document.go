package spec

import (
	"strconv"
)

// DocumentMarkdownField is the output field that a document layout renders: the Markdown body of the read. A read
// with a document declares no field of that name; the layout computes it, and an example may expect it exactly
// then, like the computed parent and depth fields of a tree read.
const DocumentMarkdownField = "markdown"

// The fixed separator of a joined line. No other separator loads.
const documentJoinSeparator = ", "

// The closed valid check of a document line: the issue-key shape.
const documentValidIssueKey = "issue_key"

// The fixed rules of the document layout. Each names the broken rule, never a value.
const (
	// ruleDocumentShape refuses a document on anything but a read of one object: a list, a first-item, a merge, a
	// tree, a paged, or a write response takes none.
	ruleDocumentShape = "needs a read of one object"
	// ruleDocumentBoth refuses a markdown field with a document: the layout renders into markdown itself, so a read
	// names one body only.
	ruleDocumentBoth = "names markdown with a document"
	// ruleDocumentField refuses a title, line, or body that names no field of the operation.
	ruleDocumentField = "names no field of the operation"
	// ruleDocumentRecord refuses a title, line, or body that names a nested record or bool map: the layout renders
	// text and string lists only.
	ruleDocumentRecord = "names a nested record or bool map"
	// ruleDocumentLine refuses a line with no closed kind: each line names one of field, verbatim, and pair.
	ruleDocumentLine = "needs one of field, verbatim, and pair"
	// ruleDocumentJoinLine refuses a join on a verbatim or pair line: a join rides a field line only.
	ruleDocumentJoinLine = "needs a field line"
	// ruleDocumentJoinList refuses a join that is not the fixed separator of a strings field.
	ruleDocumentJoinList = "is not the fixed separator of a strings field"
	// ruleDocumentStrings refuses a strings field where no join renders it: the title, a verbatim line, a pair, and
	// the body take no list.
	ruleDocumentStrings = "needs a join for a strings field"
	// ruleDocumentScalar refuses a default, skip, or valid check on a pair line: only a field or verbatim line
	// takes one.
	ruleDocumentScalar = "needs a field or verbatim line"
	// ruleDocumentConvert refuses a body whose converter is not the converter of its field: the body declares how
	// the value was converted.
	ruleDocumentConvert = "needs a field with the same converter"
	// ruleDocumentValid refuses a valid check that is not the issue-key check.
	ruleDocumentValid = "is not the issue-key check"
)

// CheckDocument returns the findings of the document layouts of a spec model: only a read of one object declares
// one, never with a markdown field of its own, every title, line, and body names a scalar field of the operation,
// each line names one closed kind, a join renders a strings field with the fixed separator, and the body converter
// matches the converter of its field. The schema states the document shape for documents. The validation and the
// engine constructor both call it, so a model built by hand meets the same refusals as a document. Each finding
// names the field path and a fixed rule.
func CheckDocument(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		if op.Response.Document == nil {
			continue
		}
		fs = append(fs, checkDocument(op, "/operations/"+strconv.Itoa(i)+"/response/document")...)
	}
	return fs
}

// checkDocument checks the document layout of one operation against its own fields.
func checkDocument(op Operation, base string) []Finding {
	if op.Access != "read" || !isObjectRead(op) {
		return []Finding{finding(base, ruleDocumentShape)}
	}
	var fs []Finding
	if op.Response.Markdown != "" {
		fs = append(fs, finding(base, ruleDocumentBoth))
	}
	doc := op.Response.Document
	fs = append(fs, checkDocumentField(op, base+"/title", doc.Title, false)...)
	for i, line := range doc.Lines {
		fs = append(fs, checkDocumentLine(op, base+"/lines/"+strconv.Itoa(i), line)...)
	}
	fs = append(fs, checkDocumentBody(op, base+"/body", doc.Body)...)
	return fs
}

// checkDocumentLine checks one metadata line: exactly one closed kind, field references that name scalar fields of
// the operation, a join on the fixed separator of a strings field, and no default, skip, or valid check on a pair.
func checkDocumentLine(op Operation, base string, line DocLine) []Finding {
	kinds := 0
	for _, kind := range []string{line.Field, line.Verbatim} {
		if kind != "" {
			kinds++
		}
	}
	if line.Pair != nil {
		kinds++
	}
	if kinds != 1 {
		return []Finding{finding(base, ruleDocumentLine)}
	}
	if line.Pair != nil {
		var fs []Finding
		fs = append(fs, checkDocumentField(op, base+"/pair/name", line.Pair.Name, false)...)
		fs = append(fs, checkDocumentField(op, base+"/pair/key", line.Pair.Key, false)...)
		if line.Default != "" {
			fs = append(fs, finding(base+"/default", ruleDocumentScalar))
		}
		if line.SkipEmpty {
			fs = append(fs, finding(base+"/skip_empty", ruleDocumentScalar))
		}
		if line.Valid != "" {
			fs = append(fs, finding(base+"/valid", ruleDocumentScalar))
		}
		if line.Join != "" {
			fs = append(fs, finding(base+"/join", ruleDocumentJoinLine))
		}
		return fs
	}
	name := line.Field
	at := base + "/field"
	join := true
	if line.Verbatim != "" {
		name = line.Verbatim
		at = base + "/verbatim"
		join = false
	}
	var fs []Finding
	fs = append(fs, checkDocumentField(op, at, name, join)...)
	if line.Join != "" {
		if line.Verbatim != "" {
			fs = append(fs, finding(base+"/join", ruleDocumentJoinLine))
		} else if line.Join != documentJoinSeparator {
			fs = append(fs, finding(base+"/join", ruleDocumentJoinList))
		} else if f, ok := op.Response.Fields[name]; ok && f.Format != "strings" {
			fs = append(fs, finding(base+"/join", ruleDocumentJoinList))
		}
	}
	if line.Valid != "" && line.Valid != documentValidIssueKey {
		fs = append(fs, finding(base+"/valid", ruleDocumentValid))
	}
	return fs
}

// checkDocumentField checks that a document reference names a scalar field of the operation: no missing field, no
// nested record or bool map, and no strings field where no join renders it. With join, the caller checks the join
// itself.
func checkDocumentField(op Operation, at, name string, join bool) []Finding {
	f, ok := op.Response.Fields[name]
	if !ok {
		return []Finding{finding(at, ruleDocumentField)}
	}
	if f.Object != nil || f.BoolMap {
		return []Finding{finding(at, ruleDocumentRecord)}
	}
	if f.Format == "strings" && !join {
		return []Finding{finding(at, ruleDocumentStrings)}
	}
	return nil
}

// checkDocumentBody checks the description body: the field names a scalar field of the operation, and the converter
// matches the converter of its field.
func checkDocumentBody(op Operation, base string, body DocBody) []Finding {
	var fs []Finding
	fs = append(fs, checkDocumentField(op, base+"/field", body.Field, false)...)
	if f, ok := op.Response.Fields[body.Field]; ok && f.Convert != body.Convert {
		fs = append(fs, finding(base+"/convert", ruleDocumentConvert))
	}
	return fs
}
