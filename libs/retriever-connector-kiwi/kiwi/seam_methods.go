package kiwi

import (
	"context"
	"encoding/json"
	"sort"
)

// TestSeamAllowedMethods returns the closed list of Kiwi methods the adapter may call, in sorted order. It reads the table
// that every call checks, so a unit binding can assert the table without reaching into the package.
func TestSeamAllowedMethods() []string {
	return sortedMethods(allowedMethods)
}

// TestSeamWriteMethods returns the allowed methods that change Kiwi, in sorted order.
func TestSeamWriteMethods() []string {
	return sortedMethods(writeMethods)
}

// TestSeamGuardArguments returns the refusal of the argument guard for params, at any depth. Every request passes this
// guard before it is encoded.
func TestSeamGuardArguments(params map[string]any) error {
	return guardArguments(params)
}

// TestSeamCall sends one call through the same path as the adapter's own calls, with the method given as text. The method
// table, the argument guard, and the encoding all apply, so a method outside the table never reaches the network.
func (g *Gateway) TestSeamCall(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	return g.call(ctx, method, params)
}

// sortedMethods returns the members of a method set in sorted order.
func sortedMethods(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for method, ok := range set {
		if ok {
			out = append(out, method)
		}
	}
	sort.Strings(out)
	return out
}
