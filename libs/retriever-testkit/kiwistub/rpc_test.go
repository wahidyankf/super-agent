package kiwistub

import (
	"testing"
)

// TestDispatchFiltersEachLookupTable checks the filter methods over the fixture: each query returns the rows its keys
// select, and a query whose plan is not a number selects nothing.
func TestDispatchFiltersEachLookupTable(t *testing.T) {
	cases := []struct {
		name   string
		method string
		query  map[string]any
		rows   int
		field  string
		want   any
	}{
		{"a priority by value", "Priority.filter", map[string]any{"value": "P1"}, 1, "id", float64(2)},
		{"no priority with that value", "Priority.filter", map[string]any{"value": "P9"}, 0, "", nil},
		{"a status by name", "TestCaseStatus.filter", map[string]any{"name": "CONFIRMED"}, 1, "id", float64(1)},
		{"no status with that name", "TestCaseStatus.filter", map[string]any{"name": "DRAFT"}, 0, "", nil},
		{"a plan by product", "TestPlan.filter", map[string]any{"product": 1}, 2, "", nil},
		{"a case by plan and summary", "TestCase.filter", map[string]any{"plan": 2, "summary": "Logout works"}, 1, "id", float64(13)},
		{"a case of a plan that is not a number", "TestCase.filter", map[string]any{"plan": "two"}, 0, "", nil},
		{"a category by name", "Category.filter", map[string]any{"name": "Functional"}, 1, "id", float64(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(WithDataset(fixture()))
			_, _, raw := call(t, s, tc.method, map[string]any{"query": tc.query})
			rows := result(t, raw).([]any)
			if len(rows) != tc.rows {
				t.Fatalf("%s returned %d rows, want %d", tc.method, len(rows), tc.rows)
			}
			if tc.field != "" && rows[0].(map[string]any)[tc.field] != tc.want {
				t.Fatalf("%s first row %s = %v, want %v", tc.method, tc.field, rows[0].(map[string]any)[tc.field], tc.want)
			}
		})
	}
}

// TestDispatchSortkeysNeedsAPlan checks that TestCase.sortkeys is refused without a plan, and that a plan that is not a
// number has no cases.
func TestDispatchSortkeysNeedsAPlan(t *testing.T) {
	s := newStub(WithDataset(fixture()))
	_, _, raw := call(t, s, "TestCase.sortkeys", map[string]any{"query": map[string]any{}})
	if code, _ := errorOf(t, raw); code != -32602 {
		t.Fatalf("sortkeys without a plan code = %d, want -32602", code)
	}
	_, _, raw = call(t, s, "TestCase.sortkeys", map[string]any{"query": map[string]any{"plan": "two"}})
	if keys, isObject := result(t, raw).(map[string]any); !isObject || len(keys) != 0 {
		t.Fatalf("sortkeys of a plan that is not a number = %v, want an empty object", keys)
	}
}

// TestDispatchAddCaseChecksItsParams checks the three refusals of TestPlan.add_case: a missing or non-numeric ID is an
// invalid parameter, a missing case and a missing plan are Kiwi's not-found messages.
func TestDispatchAddCaseChecksItsParams(t *testing.T) {
	cases := []struct {
		name     string
		params   map[string]any
		wantCode int
		wantMsg  string
	}{
		{"no plan ID", map[string]any{"case_id": 12}, -32602, "Invalid params"},
		{"a plan ID that is not a number", map[string]any{"plan_id": "2", "case_id": 12}, -32602, "Invalid params"},
		{"no such plan", map[string]any{"plan_id": 99, "case_id": 12}, -32603, "Plan does not exist"},
		{"no such case", map[string]any{"plan_id": 1, "case_id": 999}, -32603, "TestCase matching query does not exist."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(WithDataset(fixture()))
			_, _, raw := call(t, s, "TestPlan.add_case", tc.params)
			code, msg := errorOf(t, raw)
			if code != tc.wantCode || msg != tc.wantMsg {
				t.Fatalf("add_case error = %d %q, want %d %q", code, msg, tc.wantCode, tc.wantMsg)
			}
		})
	}
}

// TestDispatchAddCaseKeepsOneMembershipAndAddsTheRest checks that adding a case to a plan it already belongs to changes
// nothing, and that adding it to a new plan gives it that plan's membership and the sort key of the case.
func TestDispatchAddCaseKeepsOneMembershipAndAddsTheRest(t *testing.T) {
	s := newStub(WithDataset(fixture()))
	_, _, raw := call(t, s, "TestPlan.add_case", map[string]any{"plan_id": 2, "case_id": 12})
	if got := result(t, raw).(map[string]any); got["plan_id"] != float64(2) || got["case_id"] != float64(12) {
		t.Fatalf("add_case reply = %v, want the plan and the case", got)
	}
	_, _, raw = call(t, s, "TestCase.filter", map[string]any{"query": map[string]any{"plan": 2}})
	if got := len(result(t, raw).([]any)); got != 2 {
		t.Fatalf("plan 2 has %d cases after a repeated add, want 2", got)
	}

	call(t, s, "TestPlan.add_case", map[string]any{"plan_id": 1, "case_id": 13})
	_, _, raw = call(t, s, "TestCase.sortkeys", map[string]any{"query": map[string]any{"plan": 1}})
	if keys := result(t, raw).(map[string]any); len(keys) != 1 || keys["13"] != float64(2) {
		t.Fatalf("plan 1 sort keys = %v, want {\"13\": 2}", keys)
	}
}

// TestDispatchUpdateCaseNeedsACaseID checks that TestCase.update refuses a missing or non-numeric case ID.
func TestDispatchUpdateCaseNeedsACaseID(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]any
	}{
		{"no case ID", map[string]any{"values": map[string]any{"summary": "x"}}},
		{"a case ID that is not a number", map[string]any{"case_id": "12", "values": map[string]any{"summary": "x"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(WithDataset(fixture()))
			_, _, raw := call(t, s, "TestCase.update", tc.params)
			if code, _ := errorOf(t, raw); code != -32602 {
				t.Fatalf("update code = %d, want -32602", code)
			}
		})
	}
}
