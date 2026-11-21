package unit_test

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: Kiwi text is unescaped once and loses its control characters, except tab and line feed
func TestKiwiTextIsUnescapedOnceAndKeepsTabAndLineFeed(t *testing.T) {
	d := sampleDataset()
	d.Cases[0].Text = "Tom &amp; Jerry\x01 tab\tline\nnext"
	doc, err := docOf(newRig(t, d).gw, "case-12")
	must(t, err)
	mustContain(t, doc.Markdown, "Tom &amp; Jerry tab\tline\nnext")
	if strings.ContainsRune(doc.Markdown, 0x01) {
		t.Errorf("the control character survived:\n%q", doc.Markdown)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: A case document holds its metadata and its text
func TestCaseDocumentHoldsMetadataAndText(t *testing.T) {
	doc, err := docOf(newRig(t, sampleDataset()).gw, "case-12")
	must(t, err)
	mustContain(t, doc.Markdown, "# Login works", "- Status: CONFIRMED", "- Priority: P1", "- Category: Functional", "Open the login page.")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: A case with legacy fields shows each legacy field as a section, in order
func TestLegacyFieldsAppearAsOrderedSections(t *testing.T) {
	d := sampleDataset()
	d.Cases[0].Legacy = kiwistub.Legacy{Setup: "Open the app", Action: "Click login", Effect: "The home page shows", Breakdown: "Smoke only"}
	doc, err := docOf(newRig(t, d).gw, "case-12")
	must(t, err)
	inOrder(t, doc.Markdown, "## Setup", "## Action", "## Expected result", "## Breakdown")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: A case without legacy fields shows no legacy sections
func TestCaseWithoutLegacyFieldsHasNoLegacySections(t *testing.T) {
	doc, err := docOf(newRig(t, sampleDataset()).gw, "case-12")
	must(t, err)
	mustNotContain(t, doc.Markdown, "## Setup", "## Action", "## Expected result", "## Breakdown")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: A plan document holds its metadata and text and one line for each case, in the order given
func TestPlanDocumentHoldsMetadataTextAndOneLinePerCase(t *testing.T) {
	doc, err := docOf(newRig(t, sampleDataset()).gw, "plan-2")
	must(t, err)
	mustContain(t, doc.Markdown, "# Login", "- Product: product-1", "- Parent: plan-1", "Check the login flow.")
	inOrder(t, doc.Markdown, "## Test cases", "- case-12: Login works", "- case-13: Logout works")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/documents-and-text.feature: A plan with no cases and no parent or text says so
func TestPlanWithNoCasesSaysSo(t *testing.T) {
	d := kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}},
		Plans:    []kiwistub.Plan{{ID: 3, Name: "Empty plan", Product: 1}},
	}
	doc, err := docOf(newRig(t, d).gw, "plan-3")
	must(t, err)
	mustContain(t, doc.Markdown, "(no test cases)")
	mustNotContain(t, doc.Markdown, "- Parent:")
}
