package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// gatedRowCount is the number of catalog rows that need --yes (ADR 007 section 3.4): every destructive row and every admin
// row. It is the stage c count of the catalog, so a row that silently stopped needing --yes changes it and fails the test.
const gatedRowCount = 59

// sampleUUID is a canonical UUID that every UUID-kind parameter accepts.
const sampleUUID = "55555555-5555-4555-8555-555555555555"

// sampleValue returns a valid value for one parameter of the given kind, so that Bind accepts the argument and the gates
// of Administer are what the test exercises.
func sampleValue(p operations.Param) any {
	switch p.Kind {
	case operations.KindID, operations.KindCollectionID, operations.KindDocumentID:
		return "doc-1"
	case operations.KindCollectionUUID, operations.KindUUID, operations.KindShareID:
		return sampleUUID
	case operations.KindUUIDList:
		return []any{sampleUUID}
	case operations.KindIDList:
		return []any{"doc-1"}
	case operations.KindQuery, operations.KindName, operations.KindText:
		return "sample"
	case operations.KindLimit:
		return 1
	case operations.KindOffset:
		return 0
	case operations.KindInt:
		return p.Min
	case operations.KindBool:
		return true
	case operations.KindEnum:
		return p.Values[0]
	case operations.KindEnumList:
		return []any{p.Values[0]}
	case operations.KindTime:
		return "2026-01-01T00:00:00Z"
	case operations.KindURL:
		return "https://app.example.com/cb"
	case operations.KindURLList:
		return []any{"https://app.example.com/cb"}
	case operations.KindIndex:
		return "a"
	case operations.KindColor:
		return "#3a7bd5"
	case operations.KindIcon:
		return "star"
	case operations.KindScore:
		return 0
	case operations.KindReason:
		return ""
	case operations.KindEmail:
		return "ann@example.com"
	case operations.KindEventList:
		return []any{"documents.create"}
	}
	return "sample"
}

// sampleArgs returns the raw arguments of one call of the row: every required parameter, and one parameter of each group
// that the row says must take exactly one of its names.
func sampleArgs(row operations.Row) map[string]any {
	raw := map[string]any{}
	for _, p := range row.Params {
		if p.Required {
			raw[p.Name] = sampleValue(p)
		}
	}
	for _, name := range row.OneOf {
		if _, set := raw[name]; set {
			return raw
		}
	}
	if len(row.OneOf) > 0 {
		if p, ok := row.Param(row.OneOf[0]); ok {
			raw[p.Name] = sampleValue(p)
		}
	}
	return raw
}

// TestEveryChangeNeedsYesAndAdminBeforeAnyRead runs one table over every row of the catalog that needs --yes. Without --yes,
// on an admin source, the call is refused as yes_required; on a read source and on a write source, with --yes, it is
// refused as access_level_too_low. Each refusal happens before any read and before any admin call (ADR 007 sections 3.1,
// 3.4, and 3.5), so a row that reads its target or its caller's user first fails here.
func TestEveryChangeNeedsYesAndAdminBeforeAnyRead(t *testing.T) {
	gated := 0
	for _, row := range operations.Rows() {
		if !row.NeedsYes() {
			continue
		}
		gated++
		raw := sampleArgs(row)
		t.Run(string(row.ID), func(t *testing.T) {
			reader := &fakeReader{reply: domain.NewRecord()}
			admin := &fakeAdmin{reply: domain.NewRecord()}
			svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
			if _, err := Administer(context.Background(), svc, row.ID, strPtr("outline"), raw, Confirmation{}); !errors.Is(err, domain.ErrYesRequired) {
				t.Fatalf("without --yes: %v, want ErrYesRequired", err)
			}
			if len(reader.ops) != 0 || len(admin.ops) != 0 {
				t.Fatalf("without --yes the call read %v and reached the admin port %v; the refusal must make no request", reader.ops, admin.ops)
			}

			for _, access := range []domain.Access{domain.AccessRead, domain.AccessWrite} {
				reader := &fakeReader{reply: domain.NewRecord()}
				admin := &fakeAdmin{reply: domain.NewRecord()}
				svc, _ := adminService(t, access, reader, admin)
				if _, err := Administer(context.Background(), svc, row.ID, strPtr("outline"), raw, Confirmation{Yes: true}); !errors.Is(err, domain.ErrAccessLevelTooLow) {
					t.Fatalf("on a %s source with --yes: %v, want ErrAccessLevelTooLow", access, err)
				}
				if len(reader.ops) != 0 || len(admin.ops) != 0 {
					t.Fatalf("on a %s source the call read %v and reached the admin port %v; the refusal must make no request", access, reader.ops, admin.ops)
				}
			}
		})
	}
	if gated != gatedRowCount {
		t.Fatalf("the catalog has %d rows that need --yes, want %d (the stage c rows)", gated, gatedRowCount)
	}
}

// TestConfirmCriticalRefusesARowWithNoCanonicalValue checks the guard of a critical row that names no target, no host, and
// no text (m2): the call stops with an error before any admin call, and never dereferences a missing target.
func TestConfirmCriticalRefusesARowWithNoCanonicalValue(t *testing.T) {
	row := operations.Row{ID: domain.OpUsersDelete, Critical: &operations.Critical{Kind: "the user ID"}}
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: domain.NewRecord()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	src, err := Resolve(svc, domain.KindOutline, strPtr("outline"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, confirm := range []*string{nil, strPtr(""), strPtr("anything")} {
		err := confirmCritical(context.Background(), svc, row, src, domain.NewArgs(map[string]any{}),
			Confirmation{Yes: true, Confirm: confirm})
		if !errors.Is(err, domain.ErrUnsupportedOperation) {
			t.Fatalf("confirmCritical error = %v, want ErrUnsupportedOperation for a row with no canonical value", err)
		}
	}
	if len(reader.ops) != 0 || len(admin.ops) != 0 {
		t.Fatalf("the guard read %v and reached the admin port %v", reader.ops, admin.ops)
	}
}

// trashPage returns a reply of documents.deleted with n documents, so the dry run of documents.empty-trash can be counted.
func trashPage(n int) domain.Record {
	items := make([]domain.Record, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, targetDocument())
	}
	return domain.NewList(items)
}

// TestDryRunOfEmptyTrashCountsAFullPageAsAtLeastItsSize checks finding m7: the dry run of documents.empty-trash reads the trash
// with a page of 100, the largest the server gives, and a full page is shown as "at least 100", because the trash may hold
// more. A short page is shown as its exact count.
func TestDryRunOfEmptyTrashCountsAFullPageAsAtLeastItsSize(t *testing.T) {
	cases := []struct {
		name  string
		items int
		want  any
	}{
		{"a short page is counted exactly", 3, 3},
		{"a page of 100 is at least 100", domain.MaxPageLimit, "at least 100"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{reply: trashPage(tc.items)}
			admin := &fakeAdmin{reply: ackReply()}
			svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
			plan, err := Administer(context.Background(), svc, domain.OpDocumentsEmptyTrash, strPtr("outline"), nil, Confirmation{DryRun: true})
			if err != nil {
				t.Fatalf("dry run of empty-trash: %v", err)
			}
			if got := fieldOf(plan, "targetCount"); got != tc.want {
				t.Fatalf("targetCount = %v (%T), want %v", got, got, tc.want)
			}
			if len(reader.ops) != 1 || reader.ops[0] != domain.OpDocumentsDeleted {
				t.Fatalf("the dry run read %v, want documents.deleted once", reader.ops)
			}
			if got := reader.args[0].Values()["limit"]; got != domain.MaxPageLimit {
				t.Fatalf("the trash was read with limit %v, want %d", got, domain.MaxPageLimit)
			}
			if len(admin.ops) != 0 {
				t.Fatalf("a dry run reached the admin port: %v", admin.ops)
			}
		})
	}
}
