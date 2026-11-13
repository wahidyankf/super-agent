package jira

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestNewWithBaseRefusesEveryNonHTTPSBase proves a connection base that is not HTTPS is refused before any request, with a
// text that names Jira.
func TestNewWithBaseRefusesEveryNonHTTPSBase(t *testing.T) {
	for _, base := range []string{"http://jira-stub.invalid", "ftp://jira-stub.invalid", "jira-stub.invalid", ""} {
		_, err := NewWithBase(testSource(t), base, nil, newSecrets())
		if !errors.Is(err, domain.ErrHTTPSRequired) {
			t.Fatalf("NewWithBase(%q) = %v, want ErrHTTPSRequired", base, err)
		}
		if err.Error() != "HTTPS is required for Jira" {
			t.Fatalf("NewWithBase(%q) text = %q, want the Jira text", base, err.Error())
		}
	}
}

// TestNewWithBaseRefusesABaseWithUserInfoOrNoHost proves a base that holds user information or no host is refused, and the
// refusal never repeats the user information.
func TestNewWithBaseRefusesABaseWithUserInfoOrNoHost(t *testing.T) {
	for _, base := range []string{"https://reader@jira-stub.invalid", "https://"} {
		_, err := NewWithBase(testSource(t), base, nil, newSecrets())
		if err == nil {
			t.Fatalf("NewWithBase(%q) succeeded, want a refusal", base)
		}
		if strings.Contains(err.Error(), "reader@") {
			t.Fatalf("the refusal repeats the user information: %q", err.Error())
		}
	}
}

// TestTheProductionClientHasNoProxyJarOrRedirect proves New builds a client with no environment proxy, no cookie jar, a
// refused redirect, and the endpoint host as the TLS name.
func TestTheProductionClientHasNoProxyJarOrRedirect(t *testing.T) {
	g := New(testSource(t), newSecrets())
	tr, ok := g.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, want *http.Transport", g.client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatal("the production transport reads a proxy")
	}
	if g.client.Jar != nil {
		t.Fatal("the production client holds a cookie jar")
	}
	if g.client.CheckRedirect == nil {
		t.Fatal("the production client follows redirects")
	}
	if err := g.client.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect = %v, want ErrUseLastResponse (the 3xx reply is read, not followed)", err)
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.ServerName != testHost {
		t.Fatal("the TLS name is not the endpoint host")
	}
}

// TestNewWithBaseClearsTheCallersProxyAndJarOnACopy proves the gateway changes a copy of the caller's client, and the
// caller's own client keeps its proxy, jar, and redirect policy.
func TestNewWithBaseClearsTheCallersProxyAndJarOnACopy(t *testing.T) {
	proxyURL, err := url.Parse("http://proxy.invalid:3128")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	callerTransport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	caller := &http.Client{Transport: callerTransport, Jar: jar}

	g, err := NewWithBase(testSource(t), testBase, caller, newSecrets())
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	gatewayTransport, ok := g.client.Transport.(*http.Transport)
	if !ok || gatewayTransport.Proxy != nil {
		t.Fatal("the gateway transport still reads a proxy")
	}
	if g.client.Jar != nil {
		t.Fatal("the gateway client keeps the caller's cookie jar")
	}
	if caller.Jar != jar || callerTransport.Proxy == nil || caller.CheckRedirect != nil {
		t.Fatal("the caller's client was changed; the gateway must work on a copy")
	}
}

// TestTheTLSNameIsTheEndpointHostAndNotTheConnectionBase proves a test seam that changes the connection base keeps the
// endpoint host as the TLS name and the Host header.
func TestTheTLSNameIsTheEndpointHostAndNotTheConnectionBase(t *testing.T) {
	g, err := NewWithBase(testSource(t), "https://127.0.0.1:9443", nil, newSecrets())
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	tr, ok := g.client.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil || tr.TLSClientConfig.ServerName != testHost {
		t.Fatal("the TLS name is not the endpoint host when the base is a loopback address")
	}
	if g.base != "https://127.0.0.1:9443" || g.host != testHost {
		t.Fatalf("base %q host %q, want the loopback base and the endpoint host", g.base, g.host)
	}
}

// TestTheDefaultsMatchTheSafetyRules pins the pacing, the timeout, and the page limits that the client sets.
func TestTheDefaultsMatchTheSafetyRules(t *testing.T) {
	g := New(testSource(t), newSecrets())
	if g.minDelay != 100*time.Millisecond {
		t.Fatalf("minDelay = %v, want 100ms", g.minDelay)
	}
	if g.timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", g.timeout)
	}
	if maxBodyBytes != 10<<20 || maxRetries != 3 || maxRetryAfter != 60 {
		t.Fatal("the reply cap, retry count, or Retry-After limit differs from the client limits")
	}
}
