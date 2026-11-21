package unit_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/endpoints.feature: An HTTPS host name is accepted, and one trailing slash is dropped
func TestEndpointsAcceptHTTPSHostNameAndDropTrailingSlash(t *testing.T) {
	for _, raw := range []string{"https://docs.example.test", "https://docs.example.test/"} {
		t.Run(raw, func(t *testing.T) {
			endpoint, err := domain.NewEndpoint(raw)
			noError(t, err)
			if got := endpoint.Host(); got != "docs.example.test" {
				t.Errorf("host = %q, want %q", got, "docs.example.test")
			}
			if got := endpoint.String(); got != "https://docs.example.test" {
				t.Errorf("text = %q, want %q", got, "https://docs.example.test")
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/endpoints.feature: A plain HTTP address is refused because HTTPS is required
func TestEndpointsRefusePlainHTTPBecauseHTTPSIsRequired(t *testing.T) {
	for _, raw := range []string{"http://docs.example.test", "http://docs.example.test/"} {
		t.Run(raw, func(t *testing.T) {
			endpoint, err := domain.NewEndpoint(raw)
			wantText(t, err, "HTTPS is required for Outline")
			if endpoint != (domain.Endpoint{}) {
				t.Errorf("returned endpoint %q, want none", endpoint.String())
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/endpoints.feature: A non-origin address is refused as an invalid endpoint
func TestEndpointsRefuseNonOriginAddressAsInvalidEndpoint(t *testing.T) {
	for _, raw := range []string{
		"ftp://docs.example.test",
		"docs.example.test",
		"https://docs.example.test/wiki",
		"https://docs.example.test:8443",
		"https://user@docs.example.test",
		"https://localhost",
		"https://example",
		"https://192.0.2.10",
		"https://docs.example.1test",
		"https://Docs.example.test",
	} {
		t.Run(raw, func(t *testing.T) {
			endpoint, err := domain.NewEndpoint(raw)
			wantSentinel(t, err, domain.ErrInvalidEndpoint)
			if endpoint != (domain.Endpoint{}) {
				t.Errorf("returned endpoint %q, want none", endpoint.String())
			}
		})
	}
}
