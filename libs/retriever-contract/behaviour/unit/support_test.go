// Package unit_test holds the hermetic unit bindings of the retriever-contract scenarios. Each test names the scenario it
// covers in an @scenario comment and calls only the exported API of the library, with no file, clock, environment,
// process, network, or randomness. The helpers in this file build the values the scenarios give.
package unit_test

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// The hosts, the credential names, and the plugin pin that the scenarios use.
const (
	outlineHost    = "docs.example.test"
	outlineKey     = "OUTLINE_API_KEY"
	kiwiHost       = "kiwi.example.test"
	kiwiKey        = "KIWI_API_KEY"
	collectionUUID = "550e8400-e29b-41d4-a716-446655440000"
	validPluginPin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// noError fails the test when err is not nil.
func noError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// wantText fails the test unless err is not nil and its text is text.
func wantText(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want the error %q", text)
	}
	if got := err.Error(); got != text {
		t.Errorf("error text = %q, want %q", got, text)
	}
}

// wantSentinel fails the test unless err matches the sentinel and its text is the sentinel's text.
func wantSentinel(t *testing.T, err error, sentinel error) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to match %v", err, sentinel)
	}
	if got, want := err.Error(), sentinel.Error(); got != want {
		t.Errorf("error text = %q, want %q", got, want)
	}
}

// rowByCLI returns the built-in catalog row whose command path is cli, such as "outline documents get".
func rowByCLI(t *testing.T, cli string) operations.Row {
	t.Helper()
	for _, row := range operations.Rows() {
		if row.CLI == cli {
			return row
		}
	}
	t.Fatalf("no catalog row has the command path %q", cli)
	return operations.Row{}
}

// rowByID returns the built-in catalog row of the operation id, such as "documents.get".
func rowByID(t *testing.T, id string) operations.Row {
	t.Helper()
	row, ok := operations.Lookup(domain.OperationID(id))
	if !ok {
		t.Fatalf("the catalog holds no operation %q", id)
	}
	return row
}

// sourceIDOf returns the validated source ID of raw.
func sourceIDOf(t *testing.T, raw string) domain.SourceID {
	t.Helper()
	id, err := domain.NewSourceID(raw)
	noError(t, err)
	return id
}

// tagsOf returns the validated tag set of raw.
func tagsOf(t *testing.T, raw ...string) domain.TagSet {
	t.Helper()
	set, err := domain.NewTagSet(raw...)
	noError(t, err)
	return set
}

// endpointOf returns the endpoint of the HTTPS host.
func endpointOf(t *testing.T, host string) domain.Endpoint {
	t.Helper()
	endpoint, err := domain.NewEndpoint("https://" + host)
	noError(t, err)
	return endpoint
}

// credentialOf returns the validated credential name of raw.
func credentialOf(t *testing.T, raw string) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef(raw)
	noError(t, err)
	return ref
}

// identityOf returns the validated identity name of raw.
func identityOf(t *testing.T, raw string) domain.IdentityRef {
	t.Helper()
	ref, err := domain.NewIdentityRef(raw)
	noError(t, err)
	return ref
}

// outlineSource returns a valid outline source with the given ID, host, credential name, and tags.
func outlineSource(t *testing.T, id, host, credential string, tags ...string) domain.Source {
	t.Helper()
	return domain.NewSource(sourceIDOf(t, id), domain.KindOutline, tagsOf(t, tags...), endpointOf(t, host),
		credentialOf(t, credential))
}

// kiwiSource returns a valid Kiwi source with the given ID, host, credential name, and tags.
func kiwiSource(t *testing.T, id, host, credential string, tags ...string) domain.Source {
	t.Helper()
	return domain.NewSource(sourceIDOf(t, id), domain.KindKiwi, tagsOf(t, tags...), endpointOf(t, host),
		credentialOf(t, credential))
}

// jiraSource returns a valid Jira source with the given ID, host, credential name, and identity name.
func jiraSource(t *testing.T, id, host, credential, identity string) domain.Source {
	t.Helper()
	return domain.NewSource(sourceIDOf(t, id), domain.KindJira, tagsOf(t), endpointOf(t, host),
		credentialOf(t, credential)).WithIdentity(identityOf(t, identity))
}

// catalogOf returns the catalog of the sources and fails the test when the catalog is refused.
func catalogOf(t *testing.T, sources ...domain.Source) domain.Catalog {
	t.Helper()
	catalog, err := domain.NewCatalog(sources)
	noError(t, err)
	return catalog
}

// idsOf returns the source IDs of the sources, in the order given.
func idsOf(sources []domain.Source) []string {
	ids := make([]string, 0, len(sources))
	for _, src := range sources {
		ids = append(ids, src.ID().String())
	}
	return ids
}

// sameStrings reports whether two lists hold the same texts in the same order.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
