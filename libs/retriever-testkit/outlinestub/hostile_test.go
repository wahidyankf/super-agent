package outlinestub_test

import (
	"encoding/json"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

func TestHostileReplyPutsTheKeyInEveryObjectAndEveryList(t *testing.T) {
	body := []byte(`{"ok":true,"data":{"items":[{"id":"a"},{"id":"b","nested":{"id":"c"}}]}}`)

	out, err := outlinestub.HostileReply(body, "leakedField", "injected-value")
	if err != nil {
		t.Fatalf("HostileReply: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("hostile reply is not JSON: %v", err)
	}
	data := root["data"].(map[string]any)
	items := data["items"].([]any)
	first := items[0].(map[string]any)
	second := items[1].(map[string]any)
	nested := second["nested"].(map[string]any)
	for name, obj := range map[string]map[string]any{
		"the envelope": root, "data": data, "the first item": first, "the second item": second, "the nested object": nested,
	} {
		if obj["leakedField"] != "injected-value" {
			t.Errorf("%s lacks the injected key: %v", name, obj)
		}
	}
	if first["id"] != "a" || nested["id"] != "c" {
		t.Fatalf("the original keys changed: first %v, nested %v", first, nested)
	}
}

func TestHostileReplyLeavesAScalarReplyAsItIs(t *testing.T) {
	out, err := outlinestub.HostileReply([]byte(`"just text"`), "leakedField", "injected-value")
	if err != nil {
		t.Fatalf("HostileReply: %v", err)
	}
	if string(out) != `"just text"` {
		t.Fatalf("a scalar reply became %s, want it unchanged", out)
	}
}

func TestHostileReplyRefusesAReplyThatIsNotJSON(t *testing.T) {
	out, err := outlinestub.HostileReply([]byte(`{"ok":`), "leakedField", "injected-value")
	if err == nil {
		t.Fatalf("HostileReply returned %s for a body that is not JSON, want an error", out)
	}
	if out != nil {
		t.Fatalf("HostileReply returned bytes with its error: %s", out)
	}
}
