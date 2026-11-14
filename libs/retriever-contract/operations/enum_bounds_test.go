package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// enumListRow is a row with one list-of-enum parameter, for the bounds of that kind.
func enumListRow() Row {
	return Row{
		ID:     domain.OpGroupsList,
		Params: []Param{{Name: "permission", Kind: KindEnumList, Values: []string{"member", "admin"}}},
	}
}

// TestEnumListLengthIsAtMostTheNumberOfValues checks that a list of enum values is refused when it is longer than
// the values it may pick from, so a list cannot repeat one value to any length.
func TestEnumListLengthIsAtMostTheNumberOfValues(t *testing.T) {
	row := enumListRow()
	long := []string{"member", "member", "admin", "member", "admin"}
	if _, err := Bind(row, map[string]any{"permission": long}); !errors.Is(err, domain.ErrInvalidEnum) {
		t.Fatalf("Bind(five enum items, two values) error = %v, want ErrInvalidEnum", err)
	}
}

// TestEnumListDropsRepeatedValuesKeepingTheFirst checks that a repeated value is sent once, in its first position. The
// list is still within its bound, because it has no more items than there are values.
func TestEnumListDropsRepeatedValuesKeepingTheFirst(t *testing.T) {
	row := enumListRow()
	args, err := Bind(row, map[string]any{"permission": []any{"admin", "admin"}})
	if err != nil {
		t.Fatalf("Bind(admin, admin) error = %v", err)
	}
	if got := args.Strings("permission"); !slices.Equal(got, []string{"admin"}) {
		t.Fatalf("permission = %v, want [admin]", got)
	}
	args, err = Bind(row, map[string]any{"permission": []any{"admin", "member"}})
	if err != nil || !slices.Equal(args.Strings("permission"), []string{"admin", "member"}) {
		t.Fatalf("Bind(admin, member) = %v, %v; want [admin member]", args.Strings("permission"), err)
	}
}

// TestValuesReturnsACopyThatCannotChangeTheArgs checks that a caller who changes a list it read from Values does not
// change the Args: the list is copied as well as the map.
func TestValuesReturnsACopyThatCannotChangeTheArgs(t *testing.T) {
	args := domain.NewArgs(map[string]any{"ids": []string{"a", "b"}})
	values := args.Values()
	values["ids"].([]string)[0] = "changed"
	if got := args.Strings("ids"); got[0] != "a" {
		t.Fatalf("Args changed through Values: ids = %v, want [a b]", got)
	}
}

// TestEnumListStillRefusesAnUnknownValue checks that the bound does not hide an unknown value, and that a comma text
// of two known values is accepted.
func TestEnumListStillRefusesAnUnknownValue(t *testing.T) {
	row := enumListRow()
	if _, err := Bind(row, map[string]any{"permission": []any{"member", "owner"}}); !errors.Is(err, domain.ErrInvalidEnum) {
		t.Fatalf("Bind(unknown value) error = %v, want ErrInvalidEnum", err)
	}
	args, err := Bind(row, map[string]any{"permission": "member, admin"})
	if err != nil || !slices.Equal(args.Strings("permission"), []string{"member", "admin"}) {
		t.Fatalf("Bind(comma text) = %v, %v; want [member admin]", args.Strings("permission"), err)
	}
}
