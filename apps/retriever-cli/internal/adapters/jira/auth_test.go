package jira

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestTheKeyIsReadBeforeTheIdentityForEachRequest proves the two reads happen for every request, in the fixed order, and that
// the gateway keeps neither value between requests.
func TestTheKeyIsReadBeforeTheIdentityForEachRequest(t *testing.T) {
	secrets := newSecrets()
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), secrets)
	for i := 0; i < 2; i++ {
		if err := myself(gw); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	want := []string{"key", "identity", "key", "identity"}
	got := secrets.reads()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("reads = %v, want %v", got, want)
	}
}

// TestTheBasicHeaderCarriesTheEmailAsUserAndTheTokenAsPassword proves the header is the Atlassian form: the account e-mail is
// the user name and the API token is the password.
func TestTheBasicHeaderCarriesTheEmailAsUserAndTheTokenAsPassword(t *testing.T) {
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if err := myself(gw); err != nil {
		t.Fatalf("myself: %v", err)
	}
	auth := p.requests()[0].auth
	raw, ok := strings.CutPrefix(auth, "Basic ")
	if !ok {
		t.Fatalf("Authorization %q is not Basic", auth)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != sentinelEmail+":"+sentinelToken {
		t.Fatalf("Basic credentials are not email:token")
	}
}

// TestAMissingKeyStopsBeforeAnyRequestAndNamesTheVariable proves a missing token sends nothing and names the token variable.
func TestAMissingKeyStopsBeforeAnyRequestAndNamesTheVariable(t *testing.T) {
	secrets := newSecrets()
	secrets.key = ""
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), secrets)
	err := myself(gw)
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != keyVariable {
		t.Fatalf("err = %v, want MissingKeyError for %s", err, keyVariable)
	}
	if p.calls() != 0 {
		t.Fatalf("calls = %d, want 0: a missing key sends nothing", p.calls())
	}
}

// TestAMissingIdentityStopsBeforeAnyRequestAndNamesTheVariable proves a missing e-mail sends nothing and names its variable.
func TestAMissingIdentityStopsBeforeAnyRequestAndNamesTheVariable(t *testing.T) {
	secrets := newSecrets()
	secrets.email = ""
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), secrets)
	err := myself(gw)
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != idVariable {
		t.Fatalf("err = %v, want MissingKeyError for %s", err, idVariable)
	}
	if p.calls() != 0 {
		t.Fatalf("calls = %d, want 0", p.calls())
	}
}

// TestWhenBothAreMissingTheKeyIsNamed proves the order of the checks: the token is the first value read, so it is the one named.
func TestWhenBothAreMissingTheKeyIsNamed(t *testing.T) {
	secrets := newSecrets()
	secrets.key = ""
	secrets.email = ""
	gw := probeGateway(t, &probe{}, newFakeClock(), secrets)
	var missing domain.MissingKeyError
	if err := myself(gw); !errors.As(err, &missing) || missing.Variable != keyVariable {
		t.Fatalf("err = %v, want the key variable named first", err)
	}
}

// TestAnInvalidIdentityNamesTheVariableNotTheValue proves a malformed e-mail is refused before any request, and the text holds
// the variable name only.
func TestAnInvalidIdentityNamesTheVariableNotTheValue(t *testing.T) {
	secrets := newSecrets()
	secrets.email = "not an address with a secret:part"
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), secrets)
	err := myself(gw)
	var invalid domain.InvalidIdentityError
	if !errors.As(err, &invalid) || invalid.Variable != idVariable {
		t.Fatalf("err = %v, want InvalidIdentityError for %s", err, idVariable)
	}
	if strings.Contains(err.Error(), "secret") || p.calls() != 0 {
		t.Fatalf("the refusal shows the value or sent a request: %q, calls %d", err.Error(), p.calls())
	}
}

// TestTheRequestCarriesNoOtherAuthorization proves the only credential on a request is the Basic header, with no Bearer token.
func TestTheRequestCarriesNoOtherAuthorization(t *testing.T) {
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if err := myself(gw); err != nil {
		t.Fatalf("myself: %v", err)
	}
	if !strings.HasPrefix(p.requests()[0].auth, "Basic ") || strings.Contains(p.requests()[0].auth, "Bearer") {
		t.Fatalf("Authorization is %q, want Basic only", p.requests()[0].auth)
	}
	if got := http.MethodGet; p.requests()[0].method != got {
		t.Fatalf("method %s, want GET", p.requests()[0].method)
	}
}
