package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// hostileKey is a field name the server chose, such as an event type of a notification setting. It carries an ESC, a BEL,
// and a C1 control, which a terminal would act on if they were printed as they are.
const hostileKey = "documents.update\x1b]0;pwned\x07\u009b2J"

// TestWriteTextSanitizesFieldNamesLikeValues covers finding M2: the names of the fields are server-controlled in one
// place, the notification settings of a user, whose keys are the event types. Text mode prints each name, so a name
// loses its terminal controls the same way a value does. A nested name is checked too.
func TestWriteTextSanitizesFieldNamesLikeValues(t *testing.T) {
	row, ok := operations.Lookup(domain.OpUsersNotificationsSubscribe)
	if !ok {
		t.Fatal("users.notificationsSubscribe is not in the catalog")
	}
	cases := map[string]domain.Record{
		"a top-level field name": domain.NewRecord(
			domain.Field{Name: "id", Value: "user-1"},
			domain.Field{Name: hostileKey, Value: "yes"},
		),
		"a nested field name": domain.NewRecord(
			domain.Field{Name: "id", Value: "user-1"},
			domain.Field{Name: "notificationSettings", Value: domain.NewRecord(
				domain.Field{Name: hostileKey, Value: true},
			)},
		),
	}
	for name, rec := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			writeText(&buf, row, rec)
			out := buf.String()
			if strings.Contains(out, "\x1b") || hasTerminalControl(out) {
				t.Fatalf("text output holds a terminal control in a field name: %q", out)
			}
			if !strings.Contains(out, "pwned") {
				t.Fatalf("the visible part of the field name was lost: %q", out)
			}
		})
	}
}
