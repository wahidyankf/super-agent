package operations

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// singleParamRow returns a row with one optional parameter named value, of the given kind. A kind that no catalog row
// uses can still be bound through such a row.
func singleParamRow(kind Kind) Row {
	return Row{Params: []Param{{Name: "value", Kind: kind}}}
}

// TestBindRefusesTheWrongTypeForATextKind checks that a number is refused, with the kind's own error, for each text kind
// that no catalog row uses.
func TestBindRefusesTheWrongTypeForATextKind(t *testing.T) {
	cases := []struct {
		kind Kind
		want error
	}{
		{KindReason, domain.ErrInvalidReason},
		{KindIndex, domain.ErrInvalidIndex},
		{KindEmail, domain.ErrInvalidEmail},
	}
	for _, c := range cases {
		_, err := Bind(singleParamRow(c.kind), map[string]any{"value": 5})
		if !errors.Is(err, c.want) {
			t.Fatalf("%s with a number: error %v; want %v", c.kind, err, c.want)
		}
	}
}

// TestBindAcceptsAGoodTextValueAndRefusesABrokenRule checks that a valid value reaches the caller unchanged, and that a
// value that breaks the kind's rule is refused with the domain's error.
func TestBindAcceptsAGoodTextValueAndRefusesABrokenRule(t *testing.T) {
	cases := []struct {
		kind Kind
		good string
		bad  string
		want error
	}{
		{KindReason, "a note", "a\x01note", domain.ErrInvalidReason},
		{KindIndex, "1.5", "", domain.ErrInvalidIndex},
		{KindEmail, "person@example.test", "bad address@example.test", domain.ErrInvalidEmail},
	}
	for _, c := range cases {
		args, err := Bind(singleParamRow(c.kind), map[string]any{"value": c.good})
		if err != nil {
			t.Fatalf("%s with %q: unexpected error %v", c.kind, c.good, err)
		}
		if got := args.String("value"); got != c.good {
			t.Fatalf("%s bound %q; want %q", c.kind, got, c.good)
		}
		_, err = Bind(singleParamRow(c.kind), map[string]any{"value": c.bad})
		if !errors.Is(err, c.want) {
			t.Fatalf("%s with %q: error %v; want %v", c.kind, c.bad, err, c.want)
		}
	}
}

// TestBindChecksEventAndURLListsByTypeAndRule checks the list kinds: a non-list is refused with the kind's error, a list
// that breaks the name or length rule is refused with the domain's error, and a valid list is bound.
func TestBindChecksEventAndURLListsByTypeAndRule(t *testing.T) {
	events := singleParamRow(KindEventList)
	if _, err := Bind(events, map[string]any{"value": []any{"documents.create"}}); err != nil {
		t.Fatalf("a valid event list: unexpected error %v", err)
	}
	if _, err := Bind(events, map[string]any{"value": 5}); !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("an event list that is a number: error %v; want ErrInvalidName", err)
	}
	if _, err := Bind(events, map[string]any{"value": []any{"Not A Name!"}}); !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("an event name with spaces: error %v; want ErrInvalidName", err)
	}

	urls := singleParamRow(KindURLList)
	if _, err := Bind(urls, map[string]any{"value": []any{"https://wiki.example.test/a"}}); err != nil {
		t.Fatalf("a valid URL list: unexpected error %v", err)
	}
	if _, err := Bind(urls, map[string]any{"value": 5}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatalf("a URL list that is a number: error %v; want ErrInvalidURL", err)
	}
	if _, err := Bind(urls, map[string]any{"value": []any{}}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatalf("an empty URL list: error %v; want ErrInvalidURL", err)
	}
}

// TestAsBoolReadsTheTextTrue checks that the text true, like the bool true, reads as true.
func TestAsBoolReadsTheTextTrue(t *testing.T) {
	got, err := asBool("true")
	if err != nil || !got {
		t.Fatalf("asBool(\"true\") = %v, %v; want true, nil", got, err)
	}
}
