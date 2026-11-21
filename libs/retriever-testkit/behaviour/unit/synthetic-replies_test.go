package unit_test

import (
	"encoding/json"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/synthetic"
)

// @scenario specs/libs/retriever-testkit/behaviours/synthetic-replies.feature: The package holds one valid JSON reply for each method
func TestPackageHoldsOneValidJSONReplyForEachMethod(t *testing.T) {
	replies := synthetic.Replies()
	entries, err := fs.ReadDir(replies, ".")
	if err != nil {
		t.Fatalf("read the reply files: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("the package holds no reply files")
	}
	for _, entry := range entries {
		if path.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := fs.ReadFile(replies, entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		if !json.Valid(data) {
			t.Fatalf("%s is not one valid JSON value", entry.Name())
		}
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/synthetic-replies.feature: The package holds the reply of each named method
func TestPackageHoldsReplyOfEachNamedMethod(t *testing.T) {
	for _, method := range []string{"collections.list", "collections.info", "documents.info", "documents.export"} {
		t.Run(method, func(t *testing.T) {
			name := method + ".json"
			info, err := fs.Stat(synthetic.Replies(), name)
			if err != nil {
				t.Fatalf("the reply file %s is not found: %v", name, err)
			}
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
				t.Fatalf("%s is not a reply file", name)
			}
		})
	}
}
