package engine

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestStringsValueChargingTheBudgetFailsWhenTheListOverflows checks the budget
// side of the strings format: a list whose texts do not fit the shared output
// budget fails, so a reply cannot build without bound.
func TestStringsValueChargingTheBudgetFailsWhenTheListOverflows(t *testing.T) {
	budget := 3
	if _, err := stringsValue([]any{"abcd"}, &budget); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}

// TestCheckStringsRefusesANonList checks the guard of the strings format: a
// value that is not a list of strings fails, so a whole object or scalar is
// never copied as strings.
func TestCheckStringsRefusesANonList(t *testing.T) {
	if err := checkStrings("events", "documents.update"); err == nil {
		t.Fatal("a scalar passed the strings check")
	}
	if err := checkStrings("events", []any{"documents.update", 7}); err == nil {
		t.Fatal("a list with a non-string passed the strings check")
	}
	if err := checkStrings("events", []any{"documents.update"}); err != nil {
		t.Fatalf("a string list failed the strings check: %v", err)
	}
}

// TestHostOfParsesWithoutAHost checks the address side of the hostname
// converter: a text that parses but names no host maps to the empty text.
func TestHostOfParsesWithoutAHost(t *testing.T) {
	if got := hostOf("notaurl"); got != "" {
		t.Fatalf("hostOf(notaurl) = %#v, want empty", got)
	}
	if got := hostOf("https://hooks.example.test:8443/x"); got != "hooks.example.test" {
		t.Fatalf("hostOf with a port = %#v, want hooks.example.test", got)
	}
}
