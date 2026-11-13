//go:build integration

package outline

import (
	"context"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// recordingCreds is a Credentials double. It returns the key stored under each credential name, and it records
// every name the gateway asks for, so a test can prove which key each source reads.
type recordingCreds struct {
	mu    sync.Mutex
	keys  map[string]string
	asked []string
}

func (c *recordingCreds) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, ref.Name())
	return domain.NewApiKey(ref.Name(), c.keys[ref.Name()])
}

// sourceWithKey returns an outline source with ID id, whose endpoint is host and whose credential is the variable key.
func sourceWithKey(t *testing.T, id, host, key string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef(key)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return domain.NewSource(sid, domain.KindOutline, tags, domain.AccessRead, endpoint, ref)
}

// TestTwoSourcesEachStubSeesOnlyItsOwnKey serves two outline sources with two credential names from two stubs. Each
// stub must receive its own source's key in every request and never the other source's key, and the recording double
// must have been asked for each name.
func TestTwoSourcesEachStubSeesOnlyItsOwnKey(t *testing.T) {
	const (
		companyKey     = "company-key-3d8f1a"
		engineeringKey = "engineering-key-6b2c9e"
		engineeringH   = "wiki.engineering.example.com"
	)
	companyStub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	engineeringStub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(2)...))
	creds := &recordingCreds{keys: map[string]string{
		"EXAMPLE_OUTLINE_API_KEY":     companyKey,
		"ENGINEERING_OUTLINE_API_KEY": engineeringKey,
	}}

	company, err := NewWithBase(sourceWithKey(t, "outline", testHost, "EXAMPLE_OUTLINE_API_KEY"), companyStub.URL(), companyStub.Client(), creds, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase(company): %v", err)
	}
	engineering, err := NewWithBase(sourceWithKey(t, "outline-engineering", engineeringH, "ENGINEERING_OUTLINE_API_KEY"), engineeringStub.URL(), engineeringStub.Client(), creds, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase(engineering): %v", err)
	}
	for _, gw := range []ports.SourceGateway{company, engineering} {
		if _, err := gw.ListCollections(context.Background()); err != nil {
			t.Fatalf("ListCollections: %v", err)
		}
	}

	assertOnlyBearer(t, "company", companyStub.Requests(), companyKey, engineeringKey)
	assertOnlyBearer(t, "engineering", engineeringStub.Requests(), engineeringKey, companyKey)
	if len(companyStub.Requests()) == 0 || len(engineeringStub.Requests()) == 0 {
		t.Fatalf("each stub must receive at least one request")
	}
	if got := creds.askedNames(); len(got) != 2 || got[0] == got[1] {
		t.Fatalf("the credentials double was asked for %v, want each name once", got)
	}
}

// assertOnlyBearer fails unless every request carries the bearer key and none carries the other key.
func assertOnlyBearer(t *testing.T, label string, reqs []outlinestub.Request, own, other string) {
	t.Helper()
	for i, req := range reqs {
		if req.Authorization != "Bearer "+own {
			t.Fatalf("%s request %d carried the wrong key, want its own source key only", label, i)
		}
		if req.Authorization == "Bearer "+other {
			t.Fatalf("%s request %d carried the other source's key", label, i)
		}
	}
}

// askedNames returns the credential names asked for so far, in order.
func (c *recordingCreds) askedNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.asked...)
}
