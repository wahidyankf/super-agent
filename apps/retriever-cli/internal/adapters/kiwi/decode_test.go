package kiwi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestDecodeSortkeysReadsTheRealObjectShape checks the shape the live TestCase.sortkeys gives: one object that maps each
// case ID, as a string key, to a sort key number. An empty object is a plan without cases.
func TestDecodeSortkeysReadsTheRealObjectShape(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want map[int]int
	}{
		{"empty object", `{}`, map[int]int{}},
		{"keys by case id", `{"12": 1, "13": 2, "4041": 0}`, map[int]int{12: 1, 13: 2, 4041: 0}},
		{"a null sort key counts as zero", `{"12": null}`, map[int]int{12: 0}},
		{"a key outside the ID range is kept, because the cases decide which IDs count", `{"-3": 1, "1000000000": 2}`, map[int]int{-3: 1, 1000000000: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSortkeys(json.RawMessage(tc.raw))
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decodeSortkeys(%s) = (%v, %v), want (%v, nil)", tc.raw, got, err, tc.want)
			}
		})
	}
}

// TestDecodeSortkeysRefusesAnyOtherShape checks that a reply that is not an object of case IDs to numbers is an unexpected
// response: a list, a scalar, a key that is not a whole number, and a value that is not a number.
func TestDecodeSortkeysRefusesAnyOtherShape(t *testing.T) {
	for _, raw := range []string{
		`[]`, `[{"case_id": 12, "sortkey": 1}]`, `"x"`, `7`, `null`,
		`{"abc": 1}`, `{"": 1}`, `{"1.5": 1}`, `{"12": "first"}`, `{"12": 1.5}`,
	} {
		if _, err := decodeSortkeys(json.RawMessage(raw)); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("decodeSortkeys(%s) error = %v, want ErrUnexpectedResponse", raw, err)
		}
	}
}

// TestRowsKeepTheirFieldsWhenKiwiAddsOrNullsOthers checks the tolerance of the row decoders: an unknown field is ignored,
// and a null where a text or a number is expected leaves the zero value, as a real reply shows for a root plan.
func TestRowsKeepTheirFieldsWhenKiwiAddsOrNullsOthers(t *testing.T) {
	plans, err := decodeList[wirePlan](json.RawMessage(
		`[{"id": 3, "name": "N", "product": 1, "parent": null, "text": null, "is_active": true, "author": 5, "create_date": "x"}]`))
	if err != nil || len(plans) != 1 || plans[0].ID != 3 || plans[0].Parent != nil || plans[0].Text != "" || plans[0].Product != 1 {
		t.Fatalf("plans = (%+v, %v), want plan 3 of product 1 with a null parent", plans, err)
	}
	cases, err := decodeList[wireCase](json.RawMessage(
		`[{"id": 4, "summary": "S", "text": null, "category": 1, "priority": 2, "case_status": 3, "setup": null, "tag": [], "extra": {"a": 1}}]`))
	if err != nil || len(cases) != 1 || cases[0].ID != 4 || cases[0].Text != "" || cases[0].CaseStatus != 3 {
		t.Fatalf("cases = (%+v, %v), want case 4", cases, err)
	}
}
