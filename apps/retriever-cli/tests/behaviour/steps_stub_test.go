package behaviour

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// registerStubSteps binds the steps that give each stub its data, its faults, and its account, and the steps that check
// the requests each stub received. cur returns the world of the running scenario.
func registerStubSteps(ctx *godog.ScenarioContext, cur func() *world) {
	registerOutlineSteps(ctx, cur)
	registerKiwiSteps(ctx, cur)
	registerJiraSteps(ctx, cur)
	registerTransportSteps(ctx, cur)

	ctx.Step(`^the stub (Outline|Kiwi|Jira) server received no requests$`, func(kind string) error {
		return cur().checkRequestCount(kind, "", 0, exactCount)
	})
	ctx.Step(`^the stub (Outline|Kiwi|Jira) server received (\d+) requests?(?: to "([^"]*)")?$`,
		func(kind, want, method string) error {
			n, err := strconv.Atoi(want)
			if err != nil {
				return err
			}
			return cur().checkRequestCount(kind, method, n, exactCount)
		})
	ctx.Step(`^the stub Outline server received at least (\d+) requests?$`, func(want string) error {
		n, err := strconv.Atoi(want)
		if err != nil {
			return err
		}
		return cur().checkRequestCount("Outline", "", n, atLeastCount)
	})
	ctx.Step(`^no stub Outline server is listening$`, func() error {
		cur().outlineStub().Close()
		return nil
	})
}

// registerOutlineSteps binds the steps about the Outline stub.
func registerOutlineSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the stub Outline server holds these collections:$`, func(tbl *godog.Table) error {
		rows, err := tableRows(tbl)
		if err != nil {
			return err
		}
		w := cur()
		w.stubs.collections = nil
		for _, r := range rows {
			w.stubs.collections = append(w.stubs.collections, outlinestub.Collection{ID: r["id"], Name: r["name"]})
		}
		w.syncOutline()
		return nil
	})
	ctx.Step(`^the stub Outline server holds these documents:$`, func(tbl *godog.Table) error {
		rows, err := tableRows(tbl)
		if err != nil {
			return err
		}
		w := cur()
		w.stubs.documents = nil
		for _, r := range rows {
			w.stubs.documents = append(w.stubs.documents, outlinestub.Document{
				ID:           r["id"],
				CollectionID: r["collection"],
				ParentID:     r["parent"],
				Title:        r["title"],
				Markdown:     r["markdown"],
			})
		}
		w.syncOutline()
		return nil
	})
	ctx.Step(`^the stub Outline server holds the signed-in user "([^"]*)" of team "([^"]*)"$`,
		func(user, team string) error {
			cur().outlineStub().SetAccount(user, team)
			return nil
		})
	ctx.Step(`^the stub Outline server answers "([^"]*)" with the body:$`, func(method string, doc *godog.DocString) error {
		cur().outlineStub().SetReply(method, 200, doc.Content)
		return nil
	})
	ctx.Step(`^the stub Outline server answers "([^"]*)" with status (\d+) and the body:$`,
		func(method, status string, doc *godog.DocString) error {
			n, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Status: n, Body: doc.Content})
			return nil
		})
	ctx.Step(`^the stub Outline server answers "([^"]*)" with status (\d+)$`, func(method, status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Status: n})
		return nil
	})
	ctx.Step(`^the stub Outline server answers "([^"]*)" with status (\d+) and Retry-After "([^"]*)"$`,
		func(method, status, retryAfter string) error {
			n, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Status: n, RetryAfter: retryAfter})
			return nil
		})
	ctx.Step(`^the stub Outline server answers the first (\d+) requests? to "([^"]*)" with status (\d+) and Retry-After "([^"]*)"$`,
		func(first, method, status, retryAfter string) error {
			n, err := strconv.Atoi(first)
			if err != nil {
				return err
			}
			code, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			cur().outlineStub().SetFaultFirst(n, outlinestub.Fault{Method: method, Status: code, RetryAfter: retryAfter})
			return nil
		})
	ctx.Step(`^the stub Outline server answers "([^"]*)" with a body of (\d+) MiB$`, func(method, mib string) error {
		n, err := strconv.Atoi(mib)
		if err != nil {
			return err
		}
		cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Status: 200, Body: strings.Repeat("x", n<<20)})
		return nil
	})
	ctx.Step(`^the stub Outline server redirects "([^"]*)" to "([^"]*)"$`, func(method, location string) error {
		cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Status: 302, Location: location})
		return nil
	})
	ctx.Step(`^the stub Outline server echoes the Authorization header in a non-JSON reply to "([^"]*)"$`,
		func(method string) error {
			cur().outlineStub().SetFault(outlinestub.Fault{Method: method, EchoAuthorization: true})
			return nil
		})
	ctx.Step(`^the stub Outline server never answers "([^"]*)"$`, func(method string) error {
		cur().outlineStub().SetFault(outlinestub.Fault{Method: method, Hang: true})
		return nil
	})
	ctx.Step(`^the stub Outline server's peak concurrency was (\d+)$`, func(want string) error {
		n, err := strconv.Atoi(want)
		if err != nil {
			return err
		}
		if got := cur().outlineStub().PeakInFlight(); got != n {
			return fmt.Errorf("the peak concurrency is %d, want %d", got, n)
		}
		return nil
	})
	ctx.Step(`^the stub Outline server's last request to "([^"]*)" body contains "([^"]*)"$`,
		func(method, text string) error {
			return cur().checkLastBody(method, text)
		})
	ctx.Step(`^every request to the stub Outline server carried the Authorization header "([^"]*)"$`,
		func(header string) error {
			return cur().checkEveryRequest("Outline", func(r outlinestub.Request) error {
				if r.Authorization != header {
					return fmt.Errorf("the request %s carried another Authorization header", r.Method)
				}
				return nil
			})
		})
	ctx.Step(`^every request to the stub Outline server used the HTTP method "([^"]*)"$`, func(method string) error {
		return cur().checkEveryRequest("Outline", func(r outlinestub.Request) error {
			if r.HTTPMethod != method {
				return fmt.Errorf("the request %s used the method %s", r.Method, r.HTTPMethod)
			}
			return nil
		})
	})
	ctx.Step(`^no request to the stub Outline server carried the value of "([^"]*)" in its body or its URL$`,
		func(name string) error {
			return cur().checkNoSecretInRequests(name)
		})
}

// registerKiwiSteps binds the steps about the Kiwi stub.
func registerKiwiSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the stub Kiwi server holds this dataset:$`, func(doc *godog.DocString) error {
		var ds kiwistub.Dataset
		if err := json.Unmarshal([]byte(doc.Content), &ds); err != nil {
			return fmt.Errorf("the Kiwi dataset is not valid: %w", err)
		}
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetDataset(ds)
		return nil
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with status (\d+)$`, func(method, status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetMethodFault(kiwiMethod(method), kiwistub.Fault{Status: n})
		return nil
	})
	ctx.Step(`^the stub Kiwi server answers every request with status (\d+)$`, func(status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetFault(kiwistub.Fault{Status: n})
		return nil
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with a JSON-RPC error with code (-?\d+) and message "([^"]*)"$`,
		func(method, code, message string) error {
			n, err := strconv.Atoi(code)
			if err != nil {
				return err
			}
			s, err := cur().kiwiStub()
			if err != nil {
				return err
			}
			s.SetMethodError(kiwiMethod(method), n, message)
			return nil
		})
	ctx.Step(`^the stub Kiwi server redirects "([^"]*)" to "([^"]*)"$`, func(method, location string) error {
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetMethodFault(kiwiMethod(method), kiwistub.Fault{Redirect: true, Location: location})
		return nil
	})
	ctx.Step(`^the stub Kiwi server answers every request with status (\d+) and Retry-After "([^"]*)"$`,
		func(status, retryAfter string) error {
			n, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			s, err := cur().kiwiStub()
			if err != nil {
				return err
			}
			s.SetFault(kiwistub.Fault{Status: n, RetryAfter: retryAfter})
			return nil
		})
	ctx.Step(`^the stub Kiwi server answers every request with status (\d+) and echoes the key$`, func(status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetFault(kiwistub.Fault{Status: n, EchoKey: true})
		return nil
	})
	ctx.Step(`^the stub Kiwi server answers every request with a body that is not JSON and echoes the key$`, func() error {
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.SetFault(kiwistub.Fault{NotJSON: true, EchoKey: true})
		return nil
	})
	ctx.Step(`^the stub Kiwi server is not listening$`, func() error {
		s, err := cur().kiwiStub()
		if err != nil {
			return err
		}
		s.Close()
		return nil
	})
	ctx.Step(`^no request to the stub Kiwi server carried the value of "([^"]*)" in its body or its path$`, func(name string) error {
		return cur().checkNoSecretInKiwi(name)
	})
	ctx.Step(`^every request to the stub Kiwi server used the path "([^"]*)"$`, func(path string) error {
		return cur().checkEveryKiwi(func(r kiwistub.Request) error {
			if r.Path != path {
				return fmt.Errorf("a request used the path %s", r.Path)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub Kiwi server used the HTTP method "([^"]*)"$`, func(method string) error {
		return cur().checkEveryKiwi(func(r kiwistub.Request) error {
			if r.Verb != method {
				return fmt.Errorf("a request used the HTTP method %s", r.Verb)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub Kiwi server carried the Authorization header "([^"]*)"$`,
		func(header string) error {
			return cur().checkEveryKiwi(func(r kiwistub.Request) error {
				if r.AuthDigest != sha256Hex(header) {
					return errors.New("a request carried another Authorization header")
				}
				return nil
			})
		})
}

// registerJiraSteps binds the steps about the Jira stub.
func registerJiraSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the stub Jira server holds this configuration:$`, func(doc *godog.DocString) error {
		var cfg jirastub.Config
		if err := json.Unmarshal([]byte(doc.Content), &cfg); err != nil {
			return fmt.Errorf("the Jira configuration is not valid: %w", err)
		}
		w := cur()
		s, err := w.jiraStub()
		if err != nil {
			return err
		}
		s.ApplyConfig(cfg)
		w.stubs.jiraConfigured = true
		return nil
	})
	ctx.Step(`^the stub Jira server answers "([^"]*)" with status (\d+)$`, func(path, status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		s, err := cur().jiraStub()
		if err != nil {
			return err
		}
		s.SetFault(jirastub.Fault{Path: path, Status: n})
		return nil
	})
	ctx.Step(`^the stub Jira server answers every request with status (\d+)$`, func(status string) error {
		n, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		s, err := cur().jiraStub()
		if err != nil {
			return err
		}
		s.SetFault(jirastub.Fault{Status: n})
		return nil
	})
	ctx.Step(`^the stub Jira server answers the first (\d+) requests? to "([^"]*)" with status (\d+) and Retry-After "([^"]*)"$`,
		func(first, path, status, retryAfter string) error {
			n, err := strconv.Atoi(first)
			if err != nil {
				return err
			}
			code, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			s, err := cur().jiraStub()
			if err != nil {
				return err
			}
			s.SetFaultFirst(n, jirastub.Fault{Path: path, Status: code, RetryAfter: retryAfter})
			return nil
		})
	ctx.Step(`^the stub Jira server answers every request with status (\d+) and echoes the token and the e-mail address$`,
		func(status string) error {
			n, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			s, err := cur().jiraStub()
			if err != nil {
				return err
			}
			s.SetFault(jirastub.Fault{Status: n, EchoToken: true, EchoEmail: true})
			return nil
		})
	ctx.Step(`^every request to the stub Jira server carried the credentials of the synthetic e-mail address and token$`,
		func() error {
			return cur().checkEveryJira(func(r jirastub.Request) error {
				if !r.AuthOK {
					return errors.New("a request was not sent with the synthetic credentials")
				}
				return nil
			})
		})
}

// requestCount is how a count step compares the requests it counts with the number it expects.
const (
	exactCount = iota
	atLeastCount
)

// checkRequestCount counts the requests one stub received, by method (Outline and Kiwi) or by path (Jira), and checks
// the count. An empty method counts every request.
func (w *world) checkRequestCount(kind, method string, want, mode int) error {
	got := 0
	switch kind {
	case "Outline":
		for _, r := range w.outlineStub().Requests() {
			if method == "" || r.Method == method {
				got++
			}
		}
	case "Kiwi":
		s, err := w.kiwiStub()
		if err != nil {
			return err
		}
		name := kiwiMethod(method)
		for _, r := range s.Requests() {
			if method == "" || r.Method == name {
				got++
			}
		}
	case "Jira":
		s, err := w.jiraStub()
		if err != nil {
			return err
		}
		for _, r := range s.Requests() {
			if method == "" || r.Path == method {
				got++
			}
		}
	default:
		return fmt.Errorf("there is no stub %q", kind)
	}
	if mode == atLeastCount {
		if got < want {
			return fmt.Errorf("the stub %s received %d requests, want at least %d", kind, got, want)
		}
		return nil
	}
	if got != want {
		return fmt.Errorf("the stub %s received %d requests to %q, want %d", kind, got, method, want)
	}
	return nil
}

// checkLastBody checks that the body of the last request to a method contains the text.
func (w *world) checkLastBody(method, text string) error {
	var last *outlinestub.Request
	for _, r := range w.outlineStub().Requests() {
		if r.Method == method {
			rr := r
			last = &rr
		}
	}
	if last == nil {
		return fmt.Errorf("the stub Outline server received no request to %q", method)
	}
	body, err := json.Marshal(last.Body)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), text) {
		return fmt.Errorf("the last request to %q does not carry %q", method, text)
	}
	return nil
}

// checkEveryRequest checks one Outline request after another. It fails when the stub received none, so the check is
// never true by default.
func (w *world) checkEveryRequest(kind string, check func(outlinestub.Request) error) error {
	if kind != "Outline" {
		return fmt.Errorf("there is no request check for %q", kind)
	}
	reqs := w.outlineStub().Requests()
	if len(reqs) == 0 {
		return errors.New("the stub Outline server received no requests to check")
	}
	for _, r := range reqs {
		if err := check(r); err != nil {
			return err
		}
	}
	return nil
}

// checkNoSecretInRequests checks that no Outline request carried a secret's value in its body or its URL.
func (w *world) checkNoSecretInRequests(name string) error {
	value, err := w.valueOf(name)
	if err != nil {
		return err
	}
	for _, r := range w.outlineStub().Requests() {
		body, err := json.Marshal(r.Body)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), value) || strings.Contains(r.Path, value) {
			return fmt.Errorf("the request %s carried the value of %s", r.Method, name)
		}
	}
	return nil
}

// checkNoSecretInKiwi checks that no Kiwi request carried a secret's value in its body or its path. It fails when the stub
// received no request, so the check is never true by default.
func (w *world) checkNoSecretInKiwi(name string) error {
	value, err := w.valueOf(name)
	if err != nil {
		return err
	}
	s, err := w.kiwiStub()
	if err != nil {
		return err
	}
	reqs := s.Requests()
	if len(reqs) == 0 {
		return errors.New("the stub Kiwi server received no requests to check")
	}
	for _, r := range reqs {
		if strings.Contains(string(r.Body), value) || strings.Contains(r.Path, value) {
			return fmt.Errorf("a request %s carried the value of %s", r.Verb, name)
		}
	}
	return nil
}

// checkEveryKiwi checks one Kiwi request after another, and fails when the stub received none.
func (w *world) checkEveryKiwi(check func(kiwistub.Request) error) error {
	s, err := w.kiwiStub()
	if err != nil {
		return err
	}
	reqs := s.Requests()
	if len(reqs) == 0 {
		return errors.New("the stub Kiwi server received no requests to check")
	}
	for _, r := range reqs {
		if err := check(r); err != nil {
			return err
		}
	}
	return nil
}

// checkEveryJira checks one Jira request after another, and fails when the stub received none.
func (w *world) checkEveryJira(check func(jirastub.Request) error) error {
	s, err := w.jiraStub()
	if err != nil {
		return err
	}
	reqs := s.Requests()
	if len(reqs) == 0 {
		return errors.New("the stub Jira server received no requests to check")
	}
	for _, r := range reqs {
		if err := check(r); err != nil {
			return err
		}
	}
	return nil
}
