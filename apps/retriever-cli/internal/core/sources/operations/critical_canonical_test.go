package operations

import (
	"testing"
)

// TestEveryCriticalRowNamesACanonicalValue pins the catalog rule that makes the critical tier safe (ADR 007 section 3.5): a
// critical row names where its canonical value comes from, either a target read, a URL host argument, or a text argument.
// A row with none of the three has nothing that --confirm can be compared with, so the use case would have no value to
// check (m2). The test also checks that a Host or a Text names a parameter the row declares.
func TestEveryCriticalRowNamesACanonicalValue(t *testing.T) {
	critical := 0
	for _, row := range Rows() {
		if row.Critical == nil {
			continue
		}
		critical++
		if row.Target == nil && row.Critical.Host == "" && row.Critical.Text == "" {
			t.Errorf("%s: a critical row names no target, host, or text, so --confirm has no value to compare", row.ID)
		}
		if h := row.Critical.Host; h != "" {
			if p, ok := row.Param(h); !ok || p.Kind != KindURL {
				t.Errorf("%s: critical host names %q, which is not a URL parameter of the row", row.ID, h)
			}
		}
		if x := row.Critical.Text; x != "" {
			if _, ok := row.Param(x); !ok {
				t.Errorf("%s: critical text names %q, which the row does not declare", row.ID, x)
			}
		}
		if row.Critical.Kind == "" {
			t.Errorf("%s: a critical row names no kind of value for --confirm", row.ID)
		}
	}
	if critical == 0 {
		t.Fatal("the catalog holds no critical row, so this pin checks nothing")
	}
}
