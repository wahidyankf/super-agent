package unit_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// bindRaw binds the raw arguments of one call to the catalog row whose command path is cli.
func bindRaw(t *testing.T, cli string, raw map[string]any) (domain.Args, error) {
	t.Helper()
	return operations.Bind(rowByCLI(t, cli), raw)
}

// wantNoArguments fails the test when the binder returned any bound argument.
func wantNoArguments(t *testing.T, args domain.Args) {
	t.Helper()
	if n := len(args.Values()); n != 0 {
		t.Errorf("returned %d bound arguments, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An argument that the operation does not declare is refused
func TestBindingRefusesArgumentTheOperationDoesNotDeclare(t *testing.T) {
	args, err := bindRaw(t, "outline documents get", map[string]any{"bogus": "x"})
	wantText(t, err, "unknown argument")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: The source argument is accepted by every operation
func TestBindingAcceptsSourceArgumentOnEveryOperation(t *testing.T) {
	args, err := bindRaw(t, "outline documents get", map[string]any{"documentId": "doc-1", "source": "outline"})
	noError(t, err)
	if got := args.String("documentId"); got != "doc-1" {
		t.Errorf("document = %q, want %q", got, "doc-1")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A required argument that is not set is refused
func TestBindingRefusesRequiredArgumentThatIsNotSet(t *testing.T) {
	args, err := bindRaw(t, "outline documents get", map[string]any{})
	wantText(t, err, "a required argument is missing")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An optional argument that is not set takes its default
func TestBindingOptionalArgumentNotSetTakesDefault(t *testing.T) {
	args, err := bindRaw(t, "jira projects list", map[string]any{})
	noError(t, err)
	if got := args.Int("limit"); got != 25 {
		t.Errorf("limit = %d, want 25", got)
	}
	if got := args.Int("offset"); got != 0 {
		t.Errorf("offset = %d, want 0", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A page limit from 1 to 100 is accepted and a limit outside it is refused
func TestBindingPageLimitFromOneToOneHundredAcceptedOtherwiseRefused(t *testing.T) {
	cases := []struct {
		limit  string
		want   int
		refuse bool
	}{
		{"1", 1, false},
		{"100", 100, false},
		{"0", 0, true},
		{"101", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.limit, func(t *testing.T) {
			args, err := bindRaw(t, "jira issues search", map[string]any{
				"jql":   "project = EXAMPLE",
				"limit": tc.limit,
			})
			if tc.refuse {
				wantText(t, err, "limit must be from 1 to 100")
				wantNoArguments(t, args)
				return
			}
			noError(t, err)
			if got := args.Int("limit"); got != tc.want {
				t.Errorf("limit = %d, want %d", got, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An index outside its range is refused and the error names the argument
func TestBindingRefusesIndexOutOfRangeAndNamesArgument(t *testing.T) {
	for _, index := range []string{"-1", "2147483648"} {
		t.Run(index, func(t *testing.T) {
			args, err := bindRaw(t, "outline documents move", map[string]any{
				"documentId":   "doc-1",
				"collectionId": collectionUUID,
				"index":        index,
			})
			wantText(t, err, "index: number is out of range")
			wantNoArguments(t, args)
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An operation with an exactly-one rule refuses a call that names no target
func TestBindingExactlyOneRuleRefusesCallThatNamesNoTarget(t *testing.T) {
	args, err := bindRaw(t, "outline subscriptions get", map[string]any{})
	wantText(t, err, "exactly one of the arguments that select the target is required")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An exactly-one operation refuses a call that names two targets
func TestBindingExactlyOneOperationRefusesCallThatNamesTwoTargets(t *testing.T) {
	args, err := bindRaw(t, "outline subscriptions get", map[string]any{
		"documentId":   "doc-1",
		"collectionId": collectionUUID,
	})
	wantText(t, err, "exactly one of the arguments that select the target is required")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An operation with an exactly-one rule accepts one choice
func TestBindingExactlyOneRuleAcceptsOneChoice(t *testing.T) {
	_, err := bindRaw(t, "outline subscriptions get", map[string]any{"documentId": "doc-1"})
	noError(t, err)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An update with none of its changing arguments is refused
func TestBindingUpdateWithNoneOfItsChangingArgumentsIsRefused(t *testing.T) {
	args, err := bindRaw(t, "kiwi test-cases update", map[string]any{"documentId": "case-1"})
	wantText(t, err, "a required argument is missing")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: An update with one changing argument is accepted
func TestBindingUpdateWithOneChangingArgumentIsAccepted(t *testing.T) {
	_, err := bindRaw(t, "kiwi test-cases update", map[string]any{
		"documentId": "case-1",
		"summary":    "Login works",
	})
	noError(t, err)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A text value of a single dash is refused, as standard input is not read
func TestBindingRefusesTextValueOfSingleDash(t *testing.T) {
	args, err := bindRaw(t, "outline documents create", map[string]any{
		"title":        "Notes",
		"collectionId": collectionUUID,
		"text":         "-",
	})
	wantText(t, err, "standard input is not supported; pass the text as the argument")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A whole number given as a number is read as that number
func TestBindingReadsWholeNumberGivenAsNumber(t *testing.T) {
	args, err := bindRaw(t, "jira issues search", map[string]any{"jql": "project = EXAMPLE", "limit": 10})
	noError(t, err)
	if got := args.Int("limit"); got != 10 {
		t.Errorf("limit = %d, want 10", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A whole number given as text is read as that number
func TestBindingReadsWholeNumberGivenAsText(t *testing.T) {
	args, err := bindRaw(t, "jira issues search", map[string]any{"jql": "project = EXAMPLE", "limit": "10"})
	noError(t, err)
	if got := args.Int("limit"); got != 10 {
		t.Errorf("limit = %d, want 10", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A decimal number with no fraction is read as a whole number
func TestBindingReadsDecimalNumberWithNoFractionAsWholeNumber(t *testing.T) {
	args, err := bindRaw(t, "jira issues search", map[string]any{"jql": "project = EXAMPLE", "limit": 10.0})
	noError(t, err)
	if got := args.Int("limit"); got != 10 {
		t.Errorf("limit = %d, want 10", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A limit that is not a whole number is refused with the limit rule
func TestBindingRefusesLimitThatIsNotWholeNumber(t *testing.T) {
	args, err := bindRaw(t, "jira issues search", map[string]any{"jql": "project = EXAMPLE", "limit": 10.5})
	wantText(t, err, "limit must be from 1 to 100")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A flag accepts only the text true or false
func TestBindingFlagAcceptsOnlyTrueOrFalse(t *testing.T) {
	cases := []struct {
		value  string
		want   bool
		refuse bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"yes", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			args, err := bindRaw(t, "outline documents create", map[string]any{
				"title":        "Notes",
				"collectionId": collectionUUID,
				"text":         "Body",
				"publish":      tc.value,
			})
			if tc.refuse {
				wantText(t, err, "value must be true or false")
				wantNoArguments(t, args)
				return
			}
			noError(t, err)
			if got := args.Bool("publish"); got != tc.want {
				t.Errorf("publish = %v, want %v", got, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A list of allowed values keeps each value once in its first position
func TestBindingAllowedListKeepsEachValueOnceInFirstPosition(t *testing.T) {
	// A list may not be longer than its allowed values, so the repeat is in a two-item list (the binder refuses a
	// longer one, as its own enum bounds test states).
	args, err := bindRaw(t, "outline comments list", map[string]any{
		"documentId":   "doc-1",
		"statusFilter": "resolved,resolved",
	})
	noError(t, err)
	if got, want := args.Strings("statusFilter"), []string{"resolved"}; !sameStrings(got, want) {
		t.Errorf("status filter = %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A value that is not in the allowed list is refused
func TestBindingRefusesValueNotInAllowedList(t *testing.T) {
	args, err := bindRaw(t, "outline comments list", map[string]any{
		"documentId":   "doc-1",
		"statusFilter": "open",
	})
	wantText(t, err, "value is not one of the allowed values")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A document ID outside the ID grammar is refused as an invalid document ID
func TestBindingRefusesDocumentIDOutsideIDGrammar(t *testing.T) {
	args, err := bindRaw(t, "outline documents get", map[string]any{"documentId": "doc 1"})
	wantText(t, err, "invalid document ID")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A collection that is not a UUID is refused as an invalid collection ID
func TestBindingRefusesCollectionThatIsNotUUID(t *testing.T) {
	args, err := bindRaw(t, "outline documents create", map[string]any{
		"title":        "Notes",
		"collectionId": "not-a-uuid",
		"text":         "Body",
	})
	wantText(t, err, "invalid collection ID")
	wantNoArguments(t, args)
}

// @scenario specs/libs/retriever-contract/behaviours/arguments-and-binding.feature: A time without a zone is refused as an invalid time
func TestBindingRefusesTimeWithoutZone(t *testing.T) {
	args, err := bindRaw(t, "outline notifications update", map[string]any{
		"notificationId": collectionUUID,
		"viewedAt":       "2026-11-21T10:00:00",
	})
	wantText(t, err, "invalid time: use RFC 3339 with a zone")
	wantNoArguments(t, args)
}
