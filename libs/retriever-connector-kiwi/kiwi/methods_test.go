package kiwi

import (
	"errors"
	"sort"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestAllowlistHoldsExactlyTheTenMethods checks the closed method table. A method outside it never
// reaches the network, and no remove, delete, or Auth method is in it.
func TestAllowlistHoldsExactlyTheTenMethods(t *testing.T) {
	want := []string{
		"Category.filter", "Priority.filter", "Product.filter", "TestCase.create", "TestCase.filter",
		"TestCase.sortkeys", "TestCase.update", "TestCaseStatus.filter", "TestPlan.add_case", "TestPlan.filter",
	}
	var got []string
	for name := range allowedMethods {
		got = append(got, name)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("allowlist = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allowlist = %v, want %v", got, want)
		}
	}
}

// TestOnlyThreeAllowedMethodsChangeKiwi checks that the failure rule for writes covers exactly the three methods that
// change Kiwi, and that each of them is on the allowlist.
func TestOnlyThreeAllowedMethodsChangeKiwi(t *testing.T) {
	want := []string{methodTestCaseCreate, methodTestCaseUpdate, methodTestPlanAddCase}
	if len(writeMethods) != len(want) {
		t.Fatalf("write methods = %v, want %v", writeMethods, want)
	}
	for _, name := range want {
		if !writeMethods[name] || !allowedMethods[name] {
			t.Errorf("%s must be an allowed write method", name)
		}
	}
	if allowedMethods["TestCase.remove"] || allowedMethods["Auth.login"] {
		t.Errorf("a method outside the allowlist is allowed")
	}
}

// TestArgumentGuardRefusesForbiddenKeysAtAnyDepth checks the guard. Kiwi refuses these keys, so
// the adapter refuses to send them.
func TestArgumentGuardRefusesForbiddenKeysAtAnyDepth(t *testing.T) {
	bad := []map[string]any{
		{"query": map[string]any{"password": "x"}},
		{"values": map[string]any{"token": "x"}},
		{"query": map[string]any{"api_key": "x"}},
		{"query": map[string]any{"apikey": "x"}},
		{"values": map[string]any{"secret": "x"}},
		{"query": map[string]any{"Password": "x"}},
		{"query": map[string]any{"nested": []any{map[string]any{"TOKEN": "x"}}}},
	}
	for _, params := range bad {
		if err := guardArguments(params); !errors.Is(err, domain.ErrBadRequest) {
			t.Errorf("guardArguments(%v) = %v, want ErrBadRequest", params, err)
		}
	}
	good := map[string]any{"query": map[string]any{"id": 1, "name": "password policy"}, "values": map[string]any{"summary": "token rules"}}
	if err := guardArguments(good); err != nil {
		t.Errorf("guardArguments(good) = %v, want nil: a forbidden word in a value is allowed", err)
	}
}
