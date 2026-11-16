//go:build ignore

// Command plugintest is the fixture plugin of the retriever-plugin integration tests: one small program whose
// behaviour the request method selects, so the host proves its controls against a real process. It reads one
// JSON-RPC request line and answers one reply frame, except in the modes that test the limits. It never reads the
// network, and it prints its environment to standard error, so the tests see exactly what the host passed.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// request is the request line the host sends.
type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
}

func main() {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	var req request
	id := []byte("1")
	if err := json.Unmarshal([]byte(line), &req); err == nil && len(req.ID) > 0 {
		id = req.ID
	}
	switch req.Method {
	case "silent":
		// Never answer: the host must kill the process group at the limit.
		time.Sleep(time.Hour)
	case "flood":
		// A frame past the 1 MiB cap with no newline: the host must refuse it.
		os.Stdout.Write(bytes.Repeat([]byte("x"), (1<<20)+1))
	case "empty":
		// Exit at once with no output: the host must report that nothing answered.
		return
	case "wrongid":
		answer([]byte("2"), map[string]any{"ok": true})
	case "notify":
		answer(nil, map[string]any{"ok": true})
	case "list_collections":
		answer(id, map[string]any{"collections": []any{map[string]any{"id": "c1", "name": "One"}}})
	case "collection_tree", "document", "read", "write":
		answer(id, map[string]any{"fields": map[string]any{"ok": true}})
	default:
		// Any other method answers with the whole environment, so the test sees exactly what the host passed.
		env(id)
	}
}

// answer writes one JSON-RPC 2.0 reply frame with id (nil omits it) and result.
func answer(id json.RawMessage, result map[string]any) {
	frame := map[string]any{"jsonrpc": "2.0", "result": result}
	if id != nil {
		var raw json.RawMessage
		raw = id
		frame["id"] = raw
	}
	data, err := json.Marshal(frame)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	data = append(data, '\n')
	os.Stdout.Write(data)
}

// env answers with the sorted environment and prints it to standard error too, so the redaction of both is proved.
func env(id json.RawMessage) {
	list := os.Environ()
	sort.Strings(list)
	for _, kv := range list {
		fmt.Fprintln(os.Stderr, kv)
	}
	items := make([]any, 0, len(list))
	for _, kv := range list {
		items = append(items, kv)
	}
	answer(id, map[string]any{"env": items})
}
