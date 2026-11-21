package unit_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A row is found by its ID with its command path and effect class
func TestOperationCatalogFindsRowByIDWithPathAndEffect(t *testing.T) {
	cases := []struct{ id, cli, effect string }{
		{"documents.get", "outline documents get", "read"},
		{"documents.create", "outline documents create", "additive"},
		{"documents.move", "outline documents move", "update"},
		{"documents.archive", "outline documents archive", "destructive"},
		{"kiwi.test-cases.update", "kiwi test-cases update", "update"},
		{"jira.issues.get", "jira issues get", "read"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			row := rowByID(t, tc.id)
			if row.CLI != tc.cli {
				t.Errorf("command path = %q, want %q", row.CLI, tc.cli)
			}
			if row.Effect != domain.Effect(tc.effect) {
				t.Errorf("effect = %q, want %q", row.Effect, tc.effect)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: An ID that the catalog does not hold is not found
func TestOperationCatalogDoesNotFindUnknownID(t *testing.T) {
	if _, ok := operations.Lookup(domain.OperationID("documents.vanish")); ok {
		t.Errorf("found a row for %q, want none", "documents.vanish")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A kind runs its own rows and the rows of its closed list only
func TestOperationCatalogKindRunsItsOwnRowsAndClosedListOnly(t *testing.T) {
	cases := []struct {
		kind, id string
		runs     bool
	}{
		{"outline", "documents.get", true},
		{"outline", "kiwi.test-cases.update", false},
		{"kiwi", "kiwi.test-cases.update", true},
		{"kiwi", "documents.get", false},
		{"jira", "jira.issues.get", true},
		{"jira", "kiwi.test-cases.update", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind+" "+tc.id, func(t *testing.T) {
			kind, err := domain.BuiltinKinds().Kind(tc.kind)
			noError(t, err)
			if got := operations.Supports(kind, rowByID(t, tc.id)); got != tc.runs {
				t.Errorf("%s runs %s = %v, want %v", tc.kind, tc.id, got, tc.runs)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A catalog refuses a row that has no operation ID
func TestOperationCatalogRefusesRowWithNoOperationID(t *testing.T) {
	catalog, err := operations.NewCatalog(domain.BuiltinKinds(), []operations.Row{{CLI: "outline documents get"}})
	wantText(t, err, "an operation row needs an operation ID")
	if n := len(catalog.Rows()); n != 0 {
		t.Errorf("returned a catalog of %d rows, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A catalog refuses two rows with one operation ID
func TestOperationCatalogRefusesTwoRowsWithOneOperationID(t *testing.T) {
	row := rowByID(t, "documents.get")
	catalog, err := operations.NewCatalog(domain.BuiltinKinds(), []operations.Row{row, row})
	wantText(t, err, "duplicate operation row")
	if n := len(catalog.Rows()); n != 0 {
		t.Errorf("returned a catalog of %d rows, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A catalog refuses a row whose path starts with an unregistered kind
func TestOperationCatalogRefusesRowWhosePathStartsWithUnregisteredKind(t *testing.T) {
	row := operations.Row{ID: "github.list", CLI: "github list"}
	catalog, err := operations.NewCatalog(domain.BuiltinKinds(), []operations.Row{row})
	wantText(t, err, "a row's command path must start with a registered kind")
	if n := len(catalog.Rows()); n != 0 {
		t.Errorf("returned a catalog of %d rows, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: The Outline resource words are listed sorted, with no Kiwi or Jira word
func TestOperationCatalogListsOutlineResourceWordsSortedWithNoKiwiOrJiraWord(t *testing.T) {
	words := operations.Resources()
	if len(words) == 0 || words[0] != "access-requests" {
		t.Fatalf("resource words start with %v, want %q first", words, "access-requests")
	}
	if !slices.IsSorted(words) {
		t.Errorf("resource words %v are not sorted", words)
	}
	// A word that only Kiwi or Jira rows carry must not be listed. A word that an Outline row also carries, such as
	// comments, is an Outline word and stays listed.
	outlineWords := map[string]bool{}
	for _, row := range operations.Rows() {
		if kind, ok := row.Kind(); ok && kind == domain.KindOutline {
			outlineWords[row.Resource()] = true
		}
	}
	for _, row := range operations.Rows() {
		kind, ok := row.Kind()
		if !ok || kind == domain.KindOutline {
			continue
		}
		if word := row.Resource(); word != "" && !outlineWords[word] && slices.Contains(words, word) {
			t.Errorf("resource words list %q, which only a %s row carries", word, kind)
		}
	}
	for _, word := range words {
		if word == "kiwi" || word == "jira" {
			t.Errorf("resource words list the kind name %q", word)
		}
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A tools value is parsed into resource words, each one kept once
func TestOperationCatalogParsesToolsValueIntoResourceWordsKeptOnce(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"documents,comments", []string{"documents", "comments"}},
		{"documents,documents", []string{"documents"}},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			words, err := operations.ParseResources(tc.raw)
			noError(t, err)
			if !sameStrings(words, tc.want) {
				t.Errorf("words = %v, want %v", words, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: A tools value that is empty, has an empty entry, or is unknown is refused
func TestOperationCatalogRefusesToolsValueThatIsEmptyOrHasEmptyOrUnknownEntry(t *testing.T) {
	cases := []struct{ raw, message string }{
		{"", "the value is empty"},
		{"documents,", "an entry is empty"},
		{"Documents", `unknown resource "Documents"`},
		{"kiwi", `unknown resource "kiwi"`},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			words, err := operations.ParseResources(tc.raw)
			// The scenario states the leading words of the error. The full text adds the list of valid words.
			if err == nil || !strings.HasPrefix(err.Error(), tc.message) {
				t.Fatalf("error = %v, want an error that begins %q", err, tc.message)
			}
			if len(words) != 0 {
				t.Errorf("returned words %v, want none", words)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: The Outline write rows have the effect class of what they change
func TestOperationCatalogOutlineWriteRowsHaveEffectOfWhatTheyChange(t *testing.T) {
	cases := []struct{ id, effect string }{
		{"documents.create", "additive"},
		{"documents.duplicate", "additive"},
		{"documents.update", "update"},
		{"documents.unpublish", "update"},
		{"documents.delete", "destructive"},
		{"documents.empty-trash", "destructive"},
		{"shares.create", "additive"},
		{"shares.revoke", "destructive"},
		{"oauth-clients.rotate-secret", "update"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			if got := rowByID(t, tc.id).Effect; got != domain.Effect(tc.effect) {
				t.Errorf("effect = %q, want %q", got, tc.effect)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: Every Jira row is a read
func TestOperationCatalogEveryJiraRowIsARead(t *testing.T) {
	for _, id := range []string{
		"jira.whoami",
		"jira.projects.list",
		"jira.issues.search",
		"jira.issues.get",
		"jira.comments.list",
	} {
		t.Run(id, func(t *testing.T) {
			if got := rowByID(t, id).Effect; got != domain.EffectRead {
				t.Errorf("effect = %q, want %q", got, domain.EffectRead)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/operation-catalog.feature: Each Kiwi row of the closed list has the effect of its change
func TestOperationCatalogEachKiwiRowOfClosedListHasEffectOfItsChange(t *testing.T) {
	cases := []struct{ id, effect string }{
		{"kiwi.test-cases.create", "additive"},
		{"kiwi.test-cases.update", "update"},
		{"kiwi.plans.add-case", "additive"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			if got := rowByID(t, tc.id).Effect; got != domain.Effect(tc.effect) {
				t.Errorf("effect = %q, want %q", got, tc.effect)
			}
		})
	}
}
