package spec

import "testing"

// FuzzValidate checks, for any bytes, that validation returns without a panic, that a valid result holds a model and no
// finding, and that an invalid result holds no model and only findings with the stable code. Under go test it runs the
// seed corpus, which holds the valid and the refused documents of the tests.
func FuzzValidate(f *testing.F) {
	f.Add([]byte(minimalSpec))
	f.Add([]byte(exampleWiki))
	for _, v := range validCorpus {
		f.Add([]byte(v.doc()))
	}
	for _, fault := range faults {
		f.Add([]byte(fault.doc()))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		r := Validate(data)
		if r.Valid() {
			if r.Spec == nil {
				t.Fatal("a valid document has no model")
			}
			if r.Err() != nil {
				t.Fatal("a valid document has an error")
			}
			_ = Review(r.Spec)
			return
		}
		if r.Spec != nil {
			t.Fatal("an invalid document has a model")
		}
		for _, finding := range r.Findings {
			if finding.Code != CodeSpecInvalid {
				t.Fatalf("finding code = %q", finding.Code)
			}
		}
	})
}
