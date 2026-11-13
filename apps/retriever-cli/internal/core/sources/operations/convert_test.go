package operations

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestBindChecksEachKindOfParameter checks the rule of every parameter kind. One row is bound with one
// value; a value that passes returns its Go value, and a value that fails returns the kind's sentinel.
func TestBindChecksEachKindOfParameter(t *testing.T) {
	cases := []struct {
		name string
		kind Kind
		opts Param // Values, Min, Max
		in   any
		want any
		err  error // nil when the value must pass
	}{
		{name: "ID passes an ID", kind: KindID, in: "doc-1", want: "doc-1"},
		{name: "ID refuses a dotted ID", kind: KindID, in: "doc.1", err: domain.ErrInvalidID},
		{name: "ID refuses a number", kind: KindID, in: 7, err: domain.ErrInvalidID},
		{name: "collection ID passes an ID", kind: KindCollectionID, in: "col-1", want: "col-1"},
		{name: "collection ID refuses a dotted ID", kind: KindCollectionID, in: "col.1", err: domain.ErrInvalidCollectionID},
		{name: "collection ID refuses a number", kind: KindCollectionID, in: 7, err: domain.ErrInvalidCollectionID},
		{name: "document ID passes an ID", kind: KindDocumentID, in: "doc-1", want: "doc-1"},
		{name: "document ID refuses a dotted ID", kind: KindDocumentID, in: "doc.1", err: domain.ErrInvalidDocumentID},
		{name: "document ID refuses a number", kind: KindDocumentID, in: 7, err: domain.ErrInvalidDocumentID},
		{name: "collection UUID passes a UUID", kind: KindCollectionUUID,
			in: "123e4567-e89b-12d3-a456-426614174000", want: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "collection UUID refuses an ID", kind: KindCollectionUUID, in: "col-1", err: domain.ErrInvalidCollectionID},
		{name: "collection UUID refuses a number", kind: KindCollectionUUID, in: 7, err: domain.ErrInvalidCollectionID},
		{name: "query passes a text", kind: KindQuery, in: "install", want: "install"},
		{name: "query refuses an empty text", kind: KindQuery, in: "", err: domain.ErrInvalidQuery},
		{name: "query refuses a number", kind: KindQuery, in: 7, err: domain.ErrInvalidQuery},
		{name: "query refuses more than 1000 characters", kind: KindQuery, in: strings.Repeat("a", domain.MaxQueryLength+1), err: domain.ErrInvalidQuery},
		{name: "limit passes a text number", kind: KindLimit, in: "10", want: 10},
		{name: "limit passes a whole float", kind: KindLimit, in: float64(10), want: 10},
		{name: "limit refuses zero", kind: KindLimit, in: 0, err: domain.ErrInvalidLimit},
		{name: "limit refuses 101", kind: KindLimit, in: 101, err: domain.ErrInvalidLimit},
		{name: "limit refuses a fraction", kind: KindLimit, in: 1.5, err: domain.ErrInvalidLimit},
		{name: "limit refuses a word", kind: KindLimit, in: "x", err: domain.ErrInvalidLimit},
		{name: "limit refuses a bool", kind: KindLimit, in: true, err: domain.ErrInvalidLimit},
		{name: "offset passes zero", kind: KindOffset, in: 0, want: 0},
		{name: "offset passes a text number", kind: KindOffset, in: "3", want: 3},
		{name: "offset refuses a negative number", kind: KindOffset, in: -1, err: domain.ErrInvalidOffset},
		{name: "offset refuses a word", kind: KindOffset, in: "x", err: domain.ErrInvalidOffset},
		{name: "int passes a value in range", kind: KindInt, opts: Param{Min: 1, Max: 5}, in: 3, want: 3},
		{name: "int passes a text number in range", kind: KindInt, opts: Param{Min: 1, Max: 5}, in: "2", want: 2},
		{name: "int refuses a value above the range", kind: KindInt, opts: Param{Min: 1, Max: 5}, in: 9, err: domain.ErrInvalidInt},
		{name: "int refuses a word", kind: KindInt, opts: Param{Min: 1, Max: 5}, in: "x", err: domain.ErrInvalidInt},
		{name: "int refuses a number too large for 32 bits", kind: KindInt, opts: Param{Min: 1, Max: 5}, in: float64(1 << 40), err: domain.ErrInvalidInt},
		{name: "bool passes a bool", kind: KindBool, in: true, want: true},
		{name: "bool passes the text false", kind: KindBool, in: "false", want: false},
		{name: "bool refuses yes", kind: KindBool, in: "yes", err: domain.ErrInvalidBool},
		{name: "bool refuses a number", kind: KindBool, in: 1, err: domain.ErrInvalidBool},
		{name: "name passes a name", kind: KindName, in: "Engineering", want: "Engineering"},
		{name: "name refuses an empty name", kind: KindName, in: "", err: domain.ErrInvalidName},
		{name: "name refuses a control character", kind: KindName, in: "a\nb", err: domain.ErrInvalidName},
		{name: "name refuses a number", kind: KindName, in: 7, err: domain.ErrInvalidName},
		{name: "text passes a text", kind: KindText, in: "hello", want: "hello"},
		{name: "text refuses a number", kind: KindText, in: 7, err: domain.ErrInvalidText},
		{name: "text refuses more than 1 MiB", kind: KindText, in: strings.Repeat("a", domain.MaxTextBytes+1), err: domain.ErrInvalidText},
		{name: "enum passes a listed value", kind: KindEnum, opts: Param{Values: []string{"a", "b"}}, in: "a", want: "a"},
		{name: "enum refuses an unlisted value", kind: KindEnum, opts: Param{Values: []string{"a", "b"}}, in: "c", err: domain.ErrInvalidEnum},
		{name: "enum refuses a number", kind: KindEnum, opts: Param{Values: []string{"a", "b"}}, in: 7, err: domain.ErrInvalidEnum},
		{name: "ID list passes a list of text", kind: KindIDList, in: []any{"a", "b"}, want: []string{"a", "b"}},
		{name: "ID list passes a list of strings", kind: KindIDList, in: []string{"a"}, want: []string{"a"}},
		{name: "ID list splits comma-separated text", kind: KindIDList, in: "a, b", want: []string{"a", "b"}},
		{name: "ID list refuses a non-text item", kind: KindIDList, in: []any{7}, err: domain.ErrInvalidIDList},
		{name: "ID list refuses a number", kind: KindIDList, in: 7, err: domain.ErrInvalidIDList},
		{name: "ID list refuses a dotted ID", kind: KindIDList, in: []any{"a.b"}, err: domain.ErrInvalidIDList},
		{name: "time passes RFC 3339 with a zone", kind: KindTime, in: "2026-11-04T10:00:00Z", want: "2026-11-04T10:00:00Z"},
		{name: "time refuses a word", kind: KindTime, in: "tomorrow", err: domain.ErrInvalidTime},
		{name: "time refuses a number", kind: KindTime, in: 7, err: domain.ErrInvalidTime},
		{name: "URL passes an HTTPS URL", kind: KindURL, in: "https://docs.example.com/doc", want: "https://docs.example.com/doc"},
		{name: "URL refuses plain HTTP", kind: KindURL, in: "http://docs.example.com", err: domain.ErrInvalidURL},
		{name: "URL refuses a number", kind: KindURL, in: 7, err: domain.ErrInvalidURL},
		{name: "UUID passes a canonical UUID", kind: KindUUID,
			in: "123e4567-e89b-12d3-a456-426614174000", want: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "UUID refuses a urlId slug", kind: KindUUID, in: "fixture0001", err: domain.ErrInvalidUUID},
		{name: "UUID refuses a number", kind: KindUUID, in: 7, err: domain.ErrInvalidUUID},
		{name: "share ID passes a UUID", kind: KindShareID,
			in: "123e4567-e89b-12d3-a456-426614174000", want: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "share ID passes a lower-case slug", kind: KindShareID, in: "share-2026-a", want: "share-2026-a"},
		{name: "share ID refuses an upper-case slug", kind: KindShareID, in: "Share-A", err: domain.ErrInvalidShareID},
		{name: "share ID refuses an underscore", kind: KindShareID, in: "share_a", err: domain.ErrInvalidShareID},
		{name: "share ID refuses a number", kind: KindShareID, in: 7, err: domain.ErrInvalidShareID},
		{name: "UUID list passes a list of UUIDs", kind: KindUUIDList,
			in: []any{"123e4567-e89b-12d3-a456-426614174000"}, want: []string{"123e4567-e89b-12d3-a456-426614174000"}},
		{name: "UUID list splits comma-separated text", kind: KindUUIDList,
			in:   "123e4567-e89b-12d3-a456-426614174000, 123e4567-e89b-12d3-a456-426614174001",
			want: []string{"123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174001"}},
		{name: "UUID list refuses a urlId slug", kind: KindUUIDList, in: []any{"fixture0001"}, err: domain.ErrInvalidUUID},
		{name: "UUID list refuses a non-text item", kind: KindUUIDList, in: []any{7}, err: domain.ErrInvalidUUID},
		{name: "UUID list refuses a number", kind: KindUUIDList, in: 7, err: domain.ErrInvalidUUID},
		{name: "UUID list refuses more than 50 items", kind: KindUUIDList,
			in: repeatedUUIDs(domain.MaxIDList + 1), err: domain.ErrInvalidUUID},
		{name: "enum list passes a list of listed values", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: []any{"a", "b"}, want: []string{"a", "b"}},
		{name: "enum list passes comma-separated text", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: "a,b", want: []string{"a", "b"}},
		{name: "enum list passes an empty list", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: []any{}, want: []string{}},
		{name: "enum list refuses an unlisted value", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: []any{"c"}, err: domain.ErrInvalidEnum},
		{name: "enum list refuses a non-text item", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: []any{7}, err: domain.ErrInvalidEnum},
		{name: "enum list refuses a number", kind: KindEnumList,
			opts: Param{Values: []string{"a", "b"}}, in: 7, err: domain.ErrInvalidEnum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.opts
			p.Name = "x"
			p.Kind = tc.kind
			p.Required = true
			got, err := bindValue(p, tc.in)
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("Bind error = %v, want %v", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Bind error = %v, want the value to pass", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Bind value = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestBindRefusesAKindWithNoRule checks that a parameter kind the binder does not know is refused, never passed through.
func TestBindRefusesAKindWithNoRule(t *testing.T) {
	if _, err := bindValue(Param{Name: "x", Kind: Kind("Nope"), Required: true}, "anything"); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Bind error = %v, want ErrUnsupportedOperation", err)
	}
}

// TestSourceKindOfAnotherKindIsNotSupported checks the kind rule of a read: only the Outline kind is served.
func TestSourceKindOfAnotherKindIsNotSupported(t *testing.T) {
	if Supports(domain.SourceKind{}, rows[0]) {
		t.Fatalf("Supports(another kind) = true, want false")
	}
	if !Supports(domain.KindOutline, rows[0]) {
		t.Fatalf("Supports(outline) = false, want true")
	}
}

// bindValue binds one value of one parameter, and returns the bound value.
func bindValue(p Param, v any) (any, error) {
	row := Row{ID: domain.OpCollectionsInfo, Params: []Param{p}}
	args, err := Bind(row, map[string]any{p.Name: v})
	if err != nil {
		return nil, err
	}
	return args.Values()[p.Name], nil
}

// repeatedUUIDs returns n copies of one canonical UUID, for the list limit cases.
func repeatedUUIDs(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = "123e4567-e89b-12d3-a456-426614174000"
	}
	return out
}
