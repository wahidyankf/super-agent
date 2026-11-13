package domain

import (
	"regexp"
	"strings"
)

// endpointScheme is the only scheme an endpoint may use.
const endpointScheme = "https://"

// maxHostLength is the longest host name in bytes, per RFC 1035.
const maxHostLength = 253

// hostLabelPattern is one DNS label of a lowercase host name.
var hostLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// Endpoint is the HTTPS origin Retriever sends a source's requests to. It is a host name only, with no userinfo,
// port, path, query, or fragment. Build it with NewEndpoint.
type Endpoint struct {
	host string
}

// NewEndpoint accepts https://<host> with an optional single trailing slash, which it drops. The host must be
// lowercase and made of at least two labels. The last label must start with a letter and must not be
// localhost. This rejects every IP literal, localhost, and single-label name.
func NewEndpoint(raw string) (Endpoint, error) {
	if !strings.HasPrefix(raw, endpointScheme) {
		return Endpoint{}, ErrInvalidEndpoint
	}
	host := strings.TrimSuffix(strings.TrimPrefix(raw, endpointScheme), "/")
	if !validHostName(host) {
		return Endpoint{}, ErrInvalidEndpoint
	}
	return Endpoint{host: host}, nil
}

// validHostName reports whether host meets the endpoint host rules.
func validHostName(host string) bool {
	if len(host) < 1 || len(host) > maxHostLength {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if !hostLabelPattern.MatchString(label) {
			return false
		}
	}
	last := labels[len(labels)-1]
	return last[0] >= 'a' && last[0] <= 'z' && last != "localhost"
}

// Host returns the host name, such as docs.example.test.
func (e Endpoint) Host() string {
	return e.host
}

// String returns the endpoint as https://<host>. The zero value returns "".
func (e Endpoint) String() string {
	if e.host == "" {
		return ""
	}
	return endpointScheme + e.host
}
