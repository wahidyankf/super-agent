package jira

import (
	"fmt"
	"net/http"
)

// authorize sets the HTTP Basic authorization of one request. It is the only place in this package that
// calls SetBasicAuth. The token is read first, then the account e-mail address, both for this request only. They are locals
// of this call: nothing is stored in the gateway, a closure, a struct, or an error. A missing or invalid value stops the
// request before the network, and its error names the variable, never the value.
func (g *Gateway) authorize(req *http.Request) error {
	key, err := g.secrets.APIKey(g.credential)
	if err != nil {
		return fmt.Errorf("jira credentials: %w", err)
	}
	identity, err := g.secrets.Identity(g.identity)
	if err != nil {
		return fmt.Errorf("jira identity: %w", err)
	}
	req.SetBasicAuth(identity.Reveal(), key.Reveal())
	return nil
}
