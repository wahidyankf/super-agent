package jira

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// endpoint is one row of the closed table of Jira requests. A row names one HTTP method and one path
// template. The request builder takes a row, not a free method and path, so a request outside the table cannot be built.
type endpoint struct {
	method string
	path   string
}

// keyPlaceholder stands for one issue key in a path template. Only a key that passes ParseIssueKey is put in its place.
const keyPlaceholder = "{key}"

// The five rows. Each one is the request of one operation or of one gateway method.
var (
	epMyself        = endpoint{http.MethodGet, "/rest/api/3/myself"}
	epProjectSearch = endpoint{http.MethodGet, "/rest/api/3/project/search"}
	epJQLSearch     = endpoint{http.MethodPost, "/rest/api/3/search/jql"}
	epIssue         = endpoint{http.MethodGet, "/rest/api/3/issue/" + keyPlaceholder}
	epComments      = endpoint{http.MethodGet, "/rest/api/3/issue/" + keyPlaceholder + "/comment"}
)

// allowed is the closed table. It holds one POST, which reads, and no write method. The removed /rest/api/3/search path is
// not here. A request whose row is not in the table is refused before any network call.
var allowed = []endpoint{epMyself, epProjectSearch, epJQLSearch, epIssue, epComments}

// permitted reports whether ep is a row of the table.
func permitted(ep endpoint) bool {
	return slices.Contains(allowed, ep)
}

// newRequest builds one request of a table row, with no network call. A row outside the table returns errMethodNotAllowed.
// An issue path takes key only when it passes ParseIssueKey. The query is encoded by url.Values, and the body is sent as it
// is. The Host header is the endpoint host, so the request never names another host.
func (g *Gateway) newRequest(ctx context.Context, ep endpoint, key string, query url.Values, body []byte) (*http.Request, error) {
	if !permitted(ep) {
		return nil, errMethodNotAllowed
	}
	path := ep.path
	if strings.Contains(path, keyPlaceholder) {
		checked, err := ParseIssueKey(key)
		if err != nil {
			return nil, err
		}
		path = strings.Replace(path, keyPlaceholder, checked, 1)
	}
	target := g.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, ep.method, target, reader)
	if err != nil {
		return nil, errInvalidBaseURL
	}
	req.Host = g.host
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}
