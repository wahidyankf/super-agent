package outlinestub

import (
	"net/http"
	"strings"
	"testing"
)

// TestTheInProcessStubAnswersOnlyLoopbackAndInvalidHosts checks the host guard of the in-process transport: a request to
// a loopback address or a .invalid name is answered, and nothing is counted as refused.
func TestTheInProcessStubAnswersOnlyLoopbackAndInvalidHosts(t *testing.T) {
	for _, target := range []string{
		"https://127.0.0.1:8443/api/auth.info",
		"https://[::1]/api/auth.info",
		"https://localhost/api/auth.info",
		"https://outline-stub.invalid/api/auth.info",
	} {
		s := New()
		client := &http.Client{Transport: s.RoundTripper()}
		resp, err := client.Post(target, "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("%s: Post = %v, want the request answered", target, err)
		}
		resp.Body.Close()
		if n := s.Refused(); n != 0 {
			t.Fatalf("%s: refused = %d, want 0", target, n)
		}
	}
}

// TestTheInProcessStubRefusesEveryOtherHost checks that a request to any other host is refused before it is answered:
// the call fails, the refusal is counted, and no request is recorded as received.
func TestTheInProcessStubRefusesEveryOtherHost(t *testing.T) {
	for _, target := range []string{
		"https://docs.example.test/api/documents.update",
		"https://93.184.216.34/api/documents.update",
		"https://example.com/api/documents.update",
	} {
		s := New()
		client := &http.Client{Transport: s.RoundTripper()}
		resp, err := client.Post(target, "application/json", strings.NewReader("{}"))
		if err == nil {
			resp.Body.Close()
			t.Fatalf("%s: the request was answered, want it refused", target)
		}
		if n := s.Refused(); n != 1 {
			t.Fatalf("%s: refused = %d, want 1", target, n)
		}
		if n := len(s.Requests()); n != 0 {
			t.Fatalf("%s: requests received = %d, want 0", target, n)
		}
	}
}
