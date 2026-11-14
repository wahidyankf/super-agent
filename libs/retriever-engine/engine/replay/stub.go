// Package replay is an HTTP round tripper that answers from the examples of a connector spec. A test gives it to the
// engine in place of the network, so the engine runs the requests of a read or a write without a socket or a clock. The
// stub records every request it receives, so a test can check the method, the path, the query, the headers, and the body
// that were sent.
package replay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// ErrNoExchange is the error that the stub returns for a request that matches no script and no read example.
var ErrNoExchange = errors.New("replay: no canned exchange matches this request")

// Reply is a canned reply: the status, the headers, and the body of one response.
type Reply struct {
	Status int
	Header http.Header
	Body   []byte
}

// Request is one request that the stub received: the method, the escaped path, the raw query, the headers, and the body.
type Request struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// step is one scripted outcome: a reply, or a transport error.
type step struct {
	reply Reply
	err   error
}

// Stub answers requests from a script, and then from the read examples of a spec. It is safe for concurrent use.
type Stub struct {
	mu        sync.Mutex
	examples  map[string]Reply
	scripts   map[string][]step
	requests  []Request
	unmatched int
}

// Stub is a round tripper.
var _ http.RoundTripper = (*Stub)(nil)

// New returns a stub that answers the examples of a validated spec, read and write alike. An example whose request the
// operation catalog would refuse fails New, so the stub never serves a request the engine could not send. When two
// examples name the same method and target, the first one answers, because a request is matched by its method and target.
func New(s *spec.Spec) (*Stub, error) {
	stub := &Stub{examples: map[string]Reply{}, scripts: map[string][]step{}}
	for _, o := range s.Operations {
		for _, ex := range o.Examples {
			method, target, err := engine.ExampleTarget(s, o, ex.Request)
			if err != nil {
				return nil, fmt.Errorf("the example of %s does not bind: %w", o.ID, err)
			}
			key := method + " " + target
			if _, seen := stub.examples[key]; seen {
				continue
			}
			body, err := encodeBody(ex.Response.Body)
			if err != nil {
				return nil, fmt.Errorf("the example of %s has a body that does not encode: %w", o.ID, err)
			}
			header := http.Header{}
			for name, value := range ex.Response.Headers {
				header.Set(name, value)
			}
			stub.examples[key] = Reply{Status: ex.Response.Status, Header: header, Body: body}
		}
	}
	return stub, nil
}

// encodeBody returns the JSON of a canned body, or nothing when the example has no body.
func encodeBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return json.Marshal(body)
}

// Script sets the replies for one method and target, and it replaces any earlier script for that key. The replies are
// sent in order, and the last one repeats after the others are used up.
func (s *Stub) Script(method, target string, replies ...Reply) {
	steps := make([]step, 0, len(replies))
	for _, r := range replies {
		steps = append(steps, step{reply: r})
	}
	s.setScript(method, target, steps)
}

// Fail sets a transport error for one method and target, which every later request to that key returns.
func (s *Stub) Fail(method, target string, err error) {
	s.setScript(method, target, []step{{err: err}})
}

// setScript replaces the steps of one key.
func (s *Stub) setScript(method, target string, steps []step) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scripts[method+" "+target] = steps
}

// RoundTrip records the request, drains and closes its body, and returns the next scripted outcome for it. When no script
// names the request, the matching read example answers it. A request with neither is recorded and fails with
// ErrNoExchange.
func (s *Stub) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	target := req.URL.EscapedPath()
	if req.URL.RawQuery != "" {
		target += "?" + req.URL.RawQuery
	}
	key := req.Method + " " + target

	s.mu.Lock()
	s.requests = append(s.requests, Request{
		Method: req.Method,
		Path:   req.URL.Path,
		Query:  req.URL.RawQuery,
		Header: req.Header.Clone(),
		Body:   body,
	})
	if st, scripted := s.nextStep(key); scripted {
		s.mu.Unlock()
		if st.err != nil {
			return nil, st.err
		}
		return response(req, st.reply), nil
	}
	canned, ok := s.examples[key]
	if !ok {
		s.unmatched++
		s.mu.Unlock()
		return nil, ErrNoExchange
	}
	s.mu.Unlock()
	return response(req, canned), nil
}

// nextStep returns the next scripted step of a key. The last step stays in place, so it repeats. The caller holds the
// lock.
func (s *Stub) nextStep(key string) (step, bool) {
	queue := s.scripts[key]
	if len(queue) == 0 {
		return step{}, false
	}
	st := queue[0]
	if len(queue) > 1 {
		s.scripts[key] = queue[1:]
	}
	return st, true
}

// Requests returns a copy of every request the stub received, in order.
func (s *Stub) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Unmatched returns how many requests matched no script and no example.
func (s *Stub) Unmatched() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unmatched
}

// response builds the reply of a request from a canned reply.
func response(req *http.Request, rep Reply) *http.Response {
	header := rep.Header.Clone()
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", rep.Status, http.StatusText(rep.Status)),
		StatusCode:    rep.Status,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(rep.Body)),
		ContentLength: int64(len(rep.Body)),
		Request:       req,
	}
}
