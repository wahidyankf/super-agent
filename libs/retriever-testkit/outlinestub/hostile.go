package outlinestub

import (
	"encoding/json"
	"fmt"
)

// HostileReply returns a reply in which every object, the envelope included, holds key with the value. A key that the
// object did not have is added. It is for redaction tests only: a hostile reply carries keys the router never sends, so
// it is not checked against the router key table (CheckReplyKeys), and a scenario that uses it must say so by name. The
// projection must drop every such key, so a value that reaches an output is a leak.
func HostileReply(body []byte, key string, value any) ([]byte, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("the reply is not JSON: %w", err)
	}
	inject(root, key, value)
	return json.Marshal(root)
}

// inject sets key in every object under v, and walks into every list.
func inject(v any, key string, value any) {
	switch x := v.(type) {
	case map[string]any:
		x[key] = value
		for _, child := range x {
			inject(child, key, value)
		}
	case []any:
		for _, item := range x {
			inject(item, key, value)
		}
	}
}
