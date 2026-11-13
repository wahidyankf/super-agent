package jira

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// call sends one row of the table and decodes a 2xx reply into out. The row must be in the table, so a request outside it
// never reaches the network. A 429 reply is retried after its Retry-After wait, at most maxRetries times. Nothing else is
// retried: a 401 or a 403 is final, so a wrong token is never sent again. The request lock is held for the whole call,
// wait included, so no other request goes out while Jira asks for a pause.
func (g *Gateway) call(ctx context.Context, ep endpoint, key string, query url.Values, body []byte, out any) error {
	if !permitted(ep) {
		return errMethodNotAllowed
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for attempt := 0; ; attempt++ {
		wait, retry, err := g.attempt(ctx, ep, key, query, body, out)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
		if attempt >= maxRetries {
			return domain.ErrSourceRateLimited
		}
		if err := g.sleep(ctx, wait); err != nil {
			return err
		}
	}
}

// attempt performs one request. The caller holds the request lock. It builds the request, reads the credentials for this
// request only, paces the request after the previous one, and then classifies the reply. The returned wait and retry flag
// apply only to a 429 reply. The timeout starts when the request is sent, not while it waits for its pace.
func (g *Gateway) attempt(ctx context.Context, ep endpoint, key string, query url.Values, body []byte, out any) (time.Duration, bool, error) {
	req, err := g.newRequest(ctx, ep, key, query, body)
	if err != nil {
		return 0, false, err
	}
	if err := g.authorize(req); err != nil {
		return 0, false, err
	}
	if err := g.pace(ctx); err != nil {
		return 0, false, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	resp, err := g.client.Do(req.WithContext(reqCtx))
	if err != nil {
		g.finish()
		return 0, false, transportError(ctx, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	g.finish()
	if err != nil {
		return 0, false, transportError(ctx, err)
	}
	if len(raw) > maxBodyBytes {
		return 0, false, domain.ErrReplyTooLarge
	}
	return interpret(ep, resp.StatusCode, resp.Header.Get("Retry-After"), raw, out)
}

// pace waits until minDelay has passed since the previous request ended. The caller holds the lock.
func (g *Gateway) pace(ctx context.Context) error {
	if !g.hasLast || g.minDelay <= 0 {
		return nil
	}
	wait := g.minDelay - g.now().Sub(g.lastEnd)
	if wait <= 0 {
		return nil
	}
	return g.sleep(ctx, wait)
}

// finish records when the previous request ended. The caller holds the lock.
func (g *Gateway) finish() {
	g.lastEnd = g.now()
	g.hasLast = true
}

// sleepContext waits for d or until ctx ends, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
