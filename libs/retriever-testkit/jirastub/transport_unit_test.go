package jirastub

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestStubHostServesLoopbackAddressesAndInvalidNamesOnly(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "127.0.0.1", want: true},
		{host: "127.0.0.2", want: true},
		{host: "::1", want: true},
		{host: "jira-stub.invalid", want: true},
		{host: "198.51.100.7", want: false},
		{host: "203.0.113.9", want: false},
		{host: "jira.example.test", want: false},
		{host: "example.com", want: false},
		{host: "", want: false},
	}
	for _, tt := range tests {
		if got := stubHost(tt.host); got != tt.want {
			t.Errorf("stubHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestRoundTripReportsEachRefusalAsItsOwnError(t *testing.T) {
	s := New("a@example.test", "t")
	s.SetClock(func() time.Time { return fixedNow })
	refused, err := http.NewRequest(http.MethodGet, "https://example.com/rest/api/3/myself", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := s.RoundTripper().RoundTrip(refused); !errors.Is(err, errNotAStubHost) {
		t.Fatalf("a host the stub does not serve returned %v, want errNotAStubHost", err)
	}

	s.SetFault(Fault{Drop: true})
	dropped, err := http.NewRequest(http.MethodGet, "https://jira-stub.invalid/rest/api/3/myself", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := s.RoundTripper().RoundTrip(dropped); !errors.Is(err, errConnectionDropped) {
		t.Fatalf("a dropped request returned %v, want errConnectionDropped", err)
	}
}
