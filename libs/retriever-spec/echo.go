package spec

import "strconv"

// The fixed rules of the key echo. Each names the broken rule, never a value.
const (
	// ruleEchoShape refuses an echo on anything but a read of one object: a list, a first-item, a merge, a tree,
	// a paged, or a write response takes none.
	ruleEchoShape = "needs a read of one object"
	// ruleEchoParam refuses an echo that names no parameter of the operation.
	ruleEchoParam = "names no parameter of the operation"
	// ruleEchoPointer refuses an echo with no pointer that starts with a slash to read the echoed value from.
	ruleEchoPointer = "has no pointer that starts with a slash"
)

// CheckEcho returns the findings of the key echo blocks of a spec model: only a read of one object declares one,
// the echo names a parameter of the operation, and the pointer starts with a slash. The schema states the document
// shape for documents. The validation and the engine constructor both call it, so a model built by hand meets the
// same refusals as a document. Each finding names the field path and a fixed rule.
func CheckEcho(s *Spec) []Finding {
	var fs []Finding
	for i, op := range s.Operations {
		if op.Response.Echo == nil {
			continue
		}
		base := "/operations/" + strconv.Itoa(i) + "/response/echo"
		if op.Access != "read" || !isObjectRead(op) {
			fs = append(fs, finding(base, ruleEchoShape))
			continue
		}
		echo := op.Response.Echo
		named := false
		for _, p := range op.Params {
			if p.Name == echo.Param {
				named = true
				break
			}
		}
		if !named {
			fs = append(fs, finding(base+"/param", ruleEchoParam))
		}
		if len(echo.Pointer) == 0 || echo.Pointer[0] != '/' {
			fs = append(fs, finding(base+"/pointer", ruleEchoPointer))
		}
	}
	return fs
}
