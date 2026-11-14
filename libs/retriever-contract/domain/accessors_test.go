package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateURLListAcceptsOneToTenAddresses checks the count and length rules of a redirect list: one to MaxURLList
// addresses, each within MaxURLLength characters.
func TestValidateURLListAcceptsOneToTenAddresses(t *testing.T) {
	ok := []string{"https://app.example.com/callback"}
	if err := ValidateURLList(ok); err != nil {
		t.Fatalf("one valid address = %v, want nil", err)
	}
	if err := ValidateURLList(nil); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("an empty list = %v, want ErrInvalidURL", err)
	}
	tooMany := make([]string, MaxURLList+1)
	for i := range tooMany {
		tooMany[i] = "https://app.example.com/callback"
	}
	if err := ValidateURLList(tooMany); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("%d addresses = %v, want ErrInvalidURL", len(tooMany), err)
	}
}

// TestValidateURLListRefusesALongOrMalformedAddress checks that one bad address refuses the whole list.
func TestValidateURLListRefusesALongOrMalformedAddress(t *testing.T) {
	long := "https://app.example.com/" + strings.Repeat("a", MaxURLLength)
	if err := ValidateURLList([]string{long}); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("an address of %d characters = %v, want ErrInvalidURL", len(long), err)
	}
	if err := ValidateURLList([]string{"https://app.example.com/ok", "not a url"}); err == nil {
		t.Fatal("a malformed address in the list was accepted")
	}
}

// TestValidateEventListAcceptsNamesAndTheWildcard checks the webhook event rule: at most MaxIDList names, each a wildcard
// or a dotted name.
func TestValidateEventListAcceptsNamesAndTheWildcard(t *testing.T) {
	if err := ValidateEventList([]string{"*", "documents.create", "authenticationProviders.update"}); err != nil {
		t.Fatalf("valid event names = %v, want nil", err)
	}
	if err := ValidateEventList([]string{"documents create"}); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("a name with a space = %v, want ErrInvalidName", err)
	}
	tooMany := make([]string, MaxIDList+1)
	for i := range tooMany {
		tooMany[i] = "documents.create"
	}
	if err := ValidateEventList(tooMany); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("%d event names = %v, want ErrInvalidName", len(tooMany), err)
	}
}

// TestArgsReturnCopiesAndTypedGetters checks that Values returns a copy, that a list is copied on the way in and out, and
// that each typed getter returns the zero value for an absent parameter.
func TestArgsReturnCopiesAndTypedGetters(t *testing.T) {
	list := []string{"a", "b"}
	args := NewArgs(map[string]any{"name": "doc", "count": 3, "flag": true, "tags": list})
	list[0] = "changed"
	if got := args.Values()["tags"].([]string); got[0] != "a" {
		t.Fatalf("the list changed through the caller's slice: %v", got)
	}
	out := args.Values()
	out["name"] = "other"
	if args.String("name") != "doc" {
		t.Fatal("a change to the returned map reached the Args")
	}
	if !args.Has("name") || args.Has("absent") {
		t.Fatal("Has must report set parameters only")
	}
	if args.String("name") != "doc" || args.Int("count") != 3 || !args.Bool("flag") {
		t.Fatal("the typed getters returned the wrong values")
	}
	if args.String("absent") != "" || args.Int("absent") != 0 || args.Bool("absent") {
		t.Fatal("an absent parameter must read as the zero value")
	}
	if list, ok := NewArgs(map[string]any{"tags": []string(nil)}).Values()["tags"].([]string); !ok || list != nil {
		t.Fatal("a nil list must stay nil")
	}
}

// TestRecordAccessorsReadFieldsAndItems checks the object and list records: fields in order, typed reads of one field,
// nested records, and the copies that Fields returns.
func TestRecordAccessorsReadFieldsAndItems(t *testing.T) {
	nested := NewRecord(Field{Name: "inner", Value: "x"})
	rec := NewRecord(Field{Name: "id", Value: "d1"}, Field{Name: "count", Value: 2}, Field{Name: "child", Value: nested})
	if rec.IsList() || len(rec.Items()) != 0 {
		t.Fatal("an object record must not be a list and has no items")
	}
	fields := rec.Fields()
	if len(fields) != 3 || fields[0].Name != "id" {
		t.Fatalf("Fields = %v, want three fields in order", fields)
	}
	fields[0].Name = "changed"
	if rec.Fields()[0].Name != "id" {
		t.Fatal("a change to the returned fields reached the record")
	}
	if value, ok := rec.Get("id"); !ok || value != "d1" {
		t.Fatalf("Get(id) = %v, %v", value, ok)
	}
	if _, ok := rec.Get("absent"); ok {
		t.Fatal("Get reported an absent field as present")
	}
	if rec.Str("id") != "d1" || rec.Str("count") != "" || rec.Str("absent") != "" {
		t.Fatal("Str must return text only for a string field")
	}
	if rec.Rec("child").Str("inner") != "x" || rec.Rec("id").Str("inner") != "" || rec.Rec("absent").Str("inner") != "" {
		t.Fatal("Rec must return the nested record of a record field, and the zero record otherwise")
	}
}

// TestNewListKeepsItemsAndNilIsEmpty checks that a list record holds its items in order, and that a nil list is an
// empty list record.
func TestNewListKeepsItemsAndNilIsEmpty(t *testing.T) {
	list := NewList([]Record{NewRecord(Field{Name: "id", Value: "1"})})
	if !list.IsList() || len(list.Items()) != 1 || list.Items()[0].Str("id") != "1" {
		t.Fatal("NewList must keep the items in order")
	}
	empty := NewList(nil)
	if !empty.IsList() || empty.Items() == nil || len(empty.Items()) != 0 {
		t.Fatal("a nil list must be an empty list record")
	}
}

// TestCatalogErrorReturnsItsSafeMessage checks that a catalog error prints its message and nothing more.
func TestCatalogErrorReturnsItsSafeMessage(t *testing.T) {
	if got := (CatalogError{Message: "catalog: not found"}).Error(); got != "catalog: not found" {
		t.Fatalf("Error = %q", got)
	}
}
