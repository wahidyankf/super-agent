package synthetic

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// TestRepliesHoldOneValidJSONFilePerMethod checks the root of the reply file system: every entry is a regular file named
// <method>.json, and each file holds a valid JSON value.
func TestRepliesHoldOneValidJSONFilePerMethod(t *testing.T) {
	entries, err := fs.ReadDir(Replies(), ".")
	if err != nil {
		t.Fatalf("ReadDir(.): %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("the reply file system holds no reply")
	}
	for _, e := range entries {
		method, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || method == "" || e.IsDir() {
			t.Fatalf("entry %q is not a <method>.json reply file", e.Name())
		}
		raw, err := fs.ReadFile(Replies(), e.Name())
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", e.Name(), err)
		}
		if !json.Valid(raw) {
			t.Fatalf("reply %s is not valid JSON", e.Name())
		}
	}
}

// TestRepliesIncludeTheNamedMethods checks that the replies that the command tests read are in the file system, and
// that each one holds a valid JSON value.
func TestRepliesIncludeTheNamedMethods(t *testing.T) {
	for _, name := range []string{"collections.create.json", "collections.info.json", "comments.resolve.json", "auth.info.json"} {
		t.Run(name, func(t *testing.T) {
			raw, err := fs.ReadFile(Replies(), name)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", name, err)
			}
			if !json.Valid(raw) {
				t.Fatalf("reply %s is not valid JSON", name)
			}
		})
	}
}
