package outline

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// testHost is the endpoint host of the built-in source, which is the Host header of every request. testBase is the
// origin the in-process stub answers, a .invalid name that never resolves: a request to any other host is refused by
// the stub, so no test write can reach a real host (see TestNoWriteReachesANonLoopbackHost).
const (
	testHost = "docs.example.test"
	testBase = "https://outline-stub.invalid"
)

// testRef returns the credential reference the outline tests pass to their gateways. The fakes ignore the name.
func testRef(t *testing.T) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return ref
}

// testSource returns an outline read source whose endpoint is host, with the key name the outline tests use.
func testSource(t *testing.T, host string) domain.Source {
	t.Helper()
	return testSourceAt(t, host, domain.AccessRead)
}

// testSourceAt returns an outline source at the given access level, whose endpoint is host.
func testSourceAt(t *testing.T, host string, access domain.Access) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return domain.NewSource(id, domain.KindOutline, tags, access, endpoint, testRef(t))
}
