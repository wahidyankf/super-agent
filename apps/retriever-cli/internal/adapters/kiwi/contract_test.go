package kiwi

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/sourcecontract"
)

// kiwiIDs maps the standard dataset's IDs that have no number of their own. The IDs that name missing objects map to
// numbers that the dataset does not hold.
var kiwiIDs = map[string]string{
	"col-tree":    "product-1",
	"col-empty":   "product-2",
	"col-missing": "product-999",
	"doc-missing": "plan-999",
}

// kiwiSourceID maps a standard-dataset ID to the Kiwi ID of the same object: a collection is a product, and a document is
// a test plan. The numbered IDs keep their number.
func kiwiSourceID(id string) string {
	if mapped, ok := kiwiIDs[id]; ok {
		return mapped
	}
	if rest, ok := strings.CutPrefix(id, "col-"); ok {
		if n, err := strconv.Atoi(rest); err == nil {
			return "product-" + strconv.Itoa(n)
		}
	}
	if rest, ok := strings.CutPrefix(id, "doc-"); ok {
		if n, err := strconv.Atoi(rest); err == nil {
			return "plan-" + strconv.Itoa(n)
		}
	}
	return id
}

// kiwiNumber returns the number at the end of a Kiwi ID, such as 3 for product-3, and 0 when there is none.
func kiwiNumber(id string) int {
	n, err := strconv.Atoi(id[strings.LastIndex(id, "-")+1:])
	if err != nil {
		return 0
	}
	return n
}

// contractBody is the text of a document without its heading, which is the text a Kiwi plan holds.
func contractBody(d sourcecontract.Document) string {
	return strings.TrimSpace(strings.TrimPrefix(d.Markdown, "# "+d.Title))
}

// kiwiDataset maps the suite's dataset onto Kiwi's: each collection is a product, and each document is a test plan with
// the same title, parent, and text, and no test cases.
func kiwiDataset(d sourcecontract.Dataset) kiwistub.Dataset {
	var out kiwistub.Dataset
	for _, c := range d.Collections {
		out.Products = append(out.Products, kiwistub.Product{ID: kiwiNumber(kiwiSourceID(c.ID)), Name: c.Name})
	}
	for _, doc := range d.Documents {
		plan := kiwistub.Plan{
			ID:      kiwiNumber(kiwiSourceID(doc.ID)),
			Name:    doc.Title,
			Product: kiwiNumber(kiwiSourceID(doc.CollectionID)),
			Text:    contractBody(doc),
		}
		if doc.ParentID != "" {
			plan.Parent = kiwiNumber(kiwiSourceID(doc.ParentID))
		}
		out.Plans = append(out.Plans, plan)
	}
	return out
}

// kiwiURL is the URL of a plan: the endpoint, then the plan's path.
func kiwiURL(d sourcecontract.Document) string {
	return "https://" + testHost + "/plan/" + strconv.Itoa(kiwiNumber(kiwiSourceID(d.ID))) + "/"
}

// kiwiMarkdown is the Markdown of a plan written out as the expected literal: its title, a list of its product and its
// parent, its text, and the section for its cases, which is empty here.
func kiwiMarkdown(d sourcecontract.Document) string {
	meta := []string{"- Product: " + kiwiSourceID(d.CollectionID)}
	if d.ParentID != "" {
		meta = append(meta, "- Parent: "+kiwiSourceID(d.ParentID))
	}
	return "# " + d.Title + "\n\n" + strings.Join(meta, "\n") + "\n\n" + contractBody(d) + "\n\n## Test cases\n\n(no test cases)\n"
}

// kiwiProfile describes the Kiwi source to the shared contract suite.
func kiwiProfile() sourcecontract.Profile {
	return sourcecontract.Profile{
		MissingKeyVariable: keyVariable,
		SourceID:           kiwiSourceID,
		URL:                kiwiURL,
		Markdown:           kiwiMarkdown,
		Catalog:            false,
	}
}

// kiwiContractGateway builds the Kiwi gateway for the shared contract suite, served by an in-process stub.
func kiwiContractGateway(t *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
	t.Helper()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(kiwiDataset(w.Dataset)))
	switch w.Fault {
	case sourcecontract.FaultKeyRejected:
		stub.SetFault(kiwistub.Fault{Status: http.StatusUnauthorized})
	case sourcecontract.FaultAccessDenied:
		stub.SetFault(kiwistub.Fault{Status: http.StatusForbidden})
	case sourcecontract.FaultUpstream:
		stub.SetFault(kiwistub.Fault{Status: http.StatusInternalServerError})
	}
	var creds ports.Credentials = &staticCreds{}
	if w.Fault == sourcecontract.FaultMissingKey {
		creds = missingCreds{}
	}
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSourceAt(t), testBase, client, creds, WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// TestKiwiPassesTheSharedSourceContract runs the same behavioural contract as the Outline source, with Kiwi's IDs, URLs,
// and Markdown described by kiwiProfile.
func TestKiwiPassesTheSharedSourceContract(t *testing.T) {
	sourcecontract.RunWith(t, kiwiProfile(), kiwiContractGateway)
}
