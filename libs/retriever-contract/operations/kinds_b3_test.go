package operations

import (
	"errors"
	"math"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// oneParamRow is a row with one parameter of the given kind, for checking how Bind converts one value.
func oneParamRow(kind Kind) Row {
	return Row{Params: []Param{{Name: "value", Kind: kind, Required: true}}}
}

// TestBindConvertsAColorAndRefusesAnythingElse checks the Color kind (inventory notation color: #RGB or #RRGGBB).
func TestBindConvertsAColorAndRefusesAnythingElse(t *testing.T) {
	row := oneParamRow(KindColor)
	if _, err := Bind(row, map[string]any{"value": "#fa0"}); err != nil {
		t.Fatalf("Bind of #fa0 = %v, want nil", err)
	}
	if _, err := Bind(row, map[string]any{"value": "orange"}); !errors.Is(err, domain.ErrInvalidColor) {
		t.Fatalf("Bind of orange = %v, want ErrInvalidColor", err)
	}
	if _, err := Bind(row, map[string]any{"value": 7}); !errors.Is(err, domain.ErrInvalidColor) {
		t.Fatalf("Bind of a number = %v, want ErrInvalidColor", err)
	}
}

// TestBindConvertsAnIconAndRefusesAnythingElse checks the Icon kind: text of the icon shape, and no other type.
func TestBindConvertsAnIconAndRefusesAnythingElse(t *testing.T) {
	row := oneParamRow(KindIcon)
	if _, err := Bind(row, map[string]any{"value": "calendar"}); err != nil {
		t.Fatalf("Bind of calendar = %v, want nil", err)
	}
	if _, err := Bind(row, map[string]any{"value": ""}); !errors.Is(err, domain.ErrInvalidIcon) {
		t.Fatalf("Bind of an empty icon = %v, want ErrInvalidIcon", err)
	}
	if _, err := Bind(row, map[string]any{"value": true}); !errors.Is(err, domain.ErrInvalidIcon) {
		t.Fatalf("Bind of a bool icon = %v, want ErrInvalidIcon", err)
	}
}

// TestBindConvertsAScoreFromANumberOrItsText checks the Score kind: a JSON number, a whole or a decimal number, or its
// text as the command line gives it. A value outside -1 to 1, NaN, or text that is not a number is refused.
func TestBindConvertsAScoreFromANumberOrItsText(t *testing.T) {
	row := oneParamRow(KindScore)
	good := map[string]float64{"0.5": 0.5, " -1 ": -1, "1": 1}
	for raw, want := range good {
		args, err := Bind(row, map[string]any{"value": raw})
		if err != nil {
			t.Fatalf("Bind of %q = %v, want nil", raw, err)
		}
		if got := args.Values()["value"]; got != want {
			t.Fatalf("Bind of %q = %v, want %v", raw, got, want)
		}
	}
	for _, v := range []any{0.5, 1, -1} {
		if _, err := Bind(row, map[string]any{"value": v}); err != nil {
			t.Fatalf("Bind of the number %v = %v, want nil", v, err)
		}
	}
	bad := []any{2, -1.5, "2", "x", "NaN", "Inf", math.NaN(), true}
	for _, raw := range bad {
		if _, err := Bind(row, map[string]any{"value": raw}); !errors.Is(err, domain.ErrInvalidScore) {
			t.Fatalf("Bind of %v = %v, want ErrInvalidScore", raw, err)
		}
	}
}
