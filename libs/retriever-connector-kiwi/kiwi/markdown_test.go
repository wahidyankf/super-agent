package kiwi

import "testing"

// TestCleanUnescapesOnceAndStripsControlCharacters checks the text rule: Kiwi escapes its replies once, so Retriever
// unescapes once, and removes control characters except tab and newline.
func TestCleanUnescapesOnceAndStripsControlCharacters(t *testing.T) {
	cases := map[string]string{
		"Login &amp; logout\x00\x07!":      "Login & logout!",
		"&amp;amp;":                        "&amp;",
		"a\tb\nc\rd":                       "a\tb\ncd",
		"<b>Tom &amp; Jerry &lt;b&gt;</b>": "<b>Tom & Jerry <b></b>",
		"x\x7fy‮yz":                        "xyyz",
	}
	for in, want := range cases {
		if got := clean(in); got != want {
			t.Errorf("clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCaseDocumentHasItsMetadataAndText checks the case document layout, with the legacy sections
// shown only when Kiwi returns them.
func TestCaseDocumentHasItsMetadataAndText(t *testing.T) {
	doc := caseDocument{
		Summary:  "Login works",
		Status:   "CONFIRMED",
		Priority: "P1",
		Category: "Functional",
		Text:     "Open the login page.",
	}
	want := "# Login works\n\n- Status: CONFIRMED\n- Priority: P1\n- Category: Functional\n\nOpen the login page.\n"
	if got := caseMarkdown(doc); got != want {
		t.Fatalf("caseMarkdown =\n%q\nwant\n%q", got, want)
	}
}

// TestCaseDocumentShowsLegacyFieldsAsSections checks that the legacy fields become the four sections, in order.
func TestCaseDocumentShowsLegacyFieldsAsSections(t *testing.T) {
	doc := caseDocument{
		Summary:   "Export",
		Setup:     "A logged in user.",
		Action:    "Click Export.",
		Effect:    "A file is downloaded.",
		Breakdown: "Check the file name.",
	}
	want := "# Export\n\n## Setup\n\nA logged in user.\n\n## Action\n\nClick Export.\n\n## Expected result\n\n" +
		"A file is downloaded.\n\n## Breakdown\n\nCheck the file name.\n"
	if got := caseMarkdown(doc); got != want {
		t.Fatalf("caseMarkdown =\n%q\nwant\n%q", got, want)
	}
}

// TestPlanDocumentListsItsCases checks the plan document: metadata, text, and one line per case, in the given order.
func TestPlanDocumentListsItsCases(t *testing.T) {
	doc := planDocument{
		Name:    "Regression",
		Product: "product-1",
		Parent:  "plan-1",
		Text:    "Run before each release.",
		Cases:   []caseLine{{ID: "case-12", Summary: "Login works"}, {ID: "case-13", Summary: "Logout works"}},
	}
	want := "# Regression\n\n- Product: product-1\n- Parent: plan-1\n\nRun before each release.\n\n## Test cases\n\n" +
		"- case-12: Login works\n- case-13: Logout works\n"
	if got := planMarkdown(doc); got != want {
		t.Fatalf("planMarkdown =\n%q\nwant\n%q", got, want)
	}
}

// TestPlanDocumentWithNoCasesSaysSo checks the empty list and the absent parent and text.
func TestPlanDocumentWithNoCasesSaysSo(t *testing.T) {
	doc := planDocument{Name: "Smoke", Product: "product-1"}
	want := "# Smoke\n\n- Product: product-1\n\n## Test cases\n\n(no test cases)\n"
	if got := planMarkdown(doc); got != want {
		t.Fatalf("planMarkdown =\n%q\nwant\n%q", got, want)
	}
}
