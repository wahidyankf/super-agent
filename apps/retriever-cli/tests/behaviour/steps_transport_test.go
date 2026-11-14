package behaviour

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// tolerance is how far above an exact wait the end-to-end level accepts a real gap. The in-process levels read the fake
// clock, so their gaps are exact and get no tolerance. The end-to-end level measures real time on a loaded machine.
const tolerance = 2 * time.Second

// requestFact is what a check reads from one request a stub received, whatever the stub's own record looks like.
type requestFact struct {
	at         time.Time
	inFlight   int
	secure     bool
	host       string
	serverName string
	method     string
	path       string
	// target names the call a request makes: the Outline or JSON-RPC method, or the Jira path. A gap check can name one.
	target string
	cookie bool
}

// stubFacts returns the requests one stub received, in arrival order, as requestFacts. A kind with no stub yet has none.
func (w *world) stubFacts(kind string) ([]requestFact, error) {
	var out []requestFact
	switch kind {
	case "Outline":
		st := w.outlineStub()
		peak := st.PeakInFlight()
		for _, r := range st.Requests() {
			out = append(out, requestFact{at: r.At, inFlight: peak, secure: r.Secure, host: r.Host, method: r.HTTPMethod,
				path: r.Path, target: r.Method})
		}
	case "Kiwi":
		s, err := w.kiwiStub()
		if err != nil {
			return nil, err
		}
		for _, r := range s.Requests() {
			out = append(out, requestFact{at: r.At, secure: r.Secure, host: r.Host, method: r.Verb, path: r.Path, target: r.Method})
		}
		peak := s.PeakInFlight()
		for i := range out {
			out[i].inFlight = peak
		}
	case "Jira":
		s, err := w.jiraStub()
		if err != nil {
			return nil, err
		}
		for _, r := range s.Requests() {
			out = append(out, requestFact{at: r.At, inFlight: r.InFlight, secure: r.Secure, host: r.Host,
				serverName: r.ServerName, method: r.Method, path: r.Path, target: r.Path, cookie: r.Cookie})
		}
	default:
		return nil, fmt.Errorf("there is no stub %q", kind)
	}
	return out, nil
}

// checkEveryFact checks one request after another. It fails when the stub received none, so a check is never true by
// default.
func (w *world) checkEveryFact(kind string, check func(requestFact) error) error {
	facts, err := w.stubFacts(kind)
	if err != nil {
		return err
	}
	if len(facts) == 0 {
		return fmt.Errorf("the stub %s server received no requests to check", kind)
	}
	for _, f := range facts {
		if err := check(f); err != nil {
			return err
		}
	}
	return nil
}

// checkGaps checks the time between each two consecutive requests to one stub. When target is set, only the requests
// that make that call count, so a lookup made in between does not stand in the way. An exact check wants every gap to be
// the given duration, and an at-least check wants every gap to be at least it. At the in-process levels an exact check
// holds to the nanosecond. At the end-to-end level it allows the tolerance above the duration. A stub that received fewer
// than two matching requests has no gap, so the check fails.
func (w *world) checkGaps(kind, target string, ms int, exact bool) error {
	all, err := w.stubFacts(kind)
	if err != nil {
		return err
	}
	var facts []requestFact
	for _, f := range all {
		if target == "" || f.target == targetName(kind, target) {
			facts = append(facts, f)
		}
	}
	if len(facts) < 2 {
		return fmt.Errorf("the stub %s received %d requests, so no gap between requests can be measured", kind, len(facts))
	}
	want := time.Duration(ms) * time.Millisecond
	for i := 1; i < len(facts); i++ {
		gap := facts[i].at.Sub(facts[i-1].at)
		if gap < want {
			return fmt.Errorf("the gap before request %d to the stub %s is %v, want at least %v", i+1, kind, gap, want)
		}
		if !exact {
			continue
		}
		if w.inProcess() && gap != want {
			return fmt.Errorf("the gap before request %d to the stub %s is %v, want exactly %v", i+1, kind, gap, want)
		}
		if !w.inProcess() && gap >= want+tolerance {
			return fmt.Errorf("the gap before request %d to the stub %s is %v, want about %v", i+1, kind, gap, want)
		}
	}
	return nil
}

// targetName returns the name a stub records for a call a feature names: a Kiwi test-case name maps to its JSON-RPC name,
// and every other name is the same as the record.
func targetName(kind, name string) string {
	if kind == "Kiwi" {
		return kiwiMethod(name)
	}
	return name
}

// checkNeverOverlapped checks that no two requests to one stub were in flight at once.
func (w *world) checkNeverOverlapped(kind string) error {
	return w.checkEveryFact(kind, func(f requestFact) error {
		if f.inFlight > 1 {
			return fmt.Errorf("a request to the stub %s arrived while %d requests were in flight", kind, f.inFlight)
		}
		return nil
	})
}

// stubClockOutline returns the clock option of the Outline stub: the fake clock at the in-process levels, and none at the
// end-to-end level, where the stub stamps real time.
func (w *world) stubClockOutline() []outlinestub.Option {
	if !w.inProcess() {
		return nil
	}
	return []outlinestub.Option{outlinestub.WithClock(w.clock.Now), outlinestub.WithCertificateClock(w.clock.Now)}
}

// registerTransportSteps binds the steps about the transport of each stub: the TLS, host, and name of each request, the
// spacing and overlap of the requests, and the faults the transport rows need.
func registerTransportSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the requests to the stub (Outline|Kiwi|Jira) server were (\d+) ms apart$`, func(kind string, ms int) error {
		return cur().checkGaps(kind, "", ms, true)
	})
	ctx.Step(`^the requests to the stub (Outline|Kiwi|Jira) server were at least (\d+) ms apart$`, func(kind string, ms int) error {
		return cur().checkGaps(kind, "", ms, false)
	})
	ctx.Step(`^the requests to the stub (Outline|Kiwi|Jira) server to "([^"]*)" were (\d+) ms apart$`,
		func(kind, call string, ms int) error {
			return cur().checkGaps(kind, call, ms, true)
		})
	ctx.Step(`^the requests to the stub (Outline|Kiwi|Jira) server never overlapped$`, func(kind string) error {
		return cur().checkNeverOverlapped(kind)
	})
	ctx.Step(`^every request to the stub (Outline|Kiwi|Jira) server arrived over TLS$`, func(kind string) error {
		return cur().checkEveryFact(kind, func(f requestFact) error {
			if !f.secure {
				return fmt.Errorf("a request to the stub %s did not arrive over TLS", kind)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub (Outline|Kiwi|Jira) server had the Host header "([^"]*)"$`, func(kind, host string) error {
		return cur().checkEveryFact(kind, func(f requestFact) error {
			if f.host != host {
				return fmt.Errorf("a request to the stub %s had the Host header %q, want %q", kind, f.host, host)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub Jira server had the TLS name "([^"]*)"$`, func(name string) error {
		return cur().checkEveryFact("Jira", func(f requestFact) error {
			if f.serverName != name {
				return fmt.Errorf("a request to the stub Jira had the TLS name %q, want %q", f.serverName, name)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub Jira server used the HTTP method "([^"]*)"$`, func(method string) error {
		return cur().checkEveryFact("Jira", func(f requestFact) error {
			if f.method != method {
				return fmt.Errorf("a request to the stub Jira used the HTTP method %s", f.method)
			}
			return nil
		})
	})
	ctx.Step(`^every request to the stub Jira server used the path "([^"]*)"$`, func(path string) error {
		return cur().checkEveryFact("Jira", func(f requestFact) error {
			if f.path != path {
				return fmt.Errorf("a request to the stub Jira used the path %s", f.path)
			}
			return nil
		})
	})
	ctx.Step(`^no request to the stub Jira server carried a cookie$`, func() error {
		return cur().checkEveryFact("Jira", func(f requestFact) error {
			if f.cookie {
				return fmt.Errorf("a request to the stub Jira carried a cookie")
			}
			return nil
		})
	})
	ctx.Step(`^the stub Jira server sets the cookie "([^"]*)" on every reply$`, func(name string) error {
		s, err := cur().jiraStub()
		if err != nil {
			return err
		}
		s.SetCookie(name)
		return nil
	})
	ctx.Step(`^the stub Jira server redirects "([^"]*)" to "([^"]*)"$`, func(path, location string) error {
		s, err := cur().jiraStub()
		if err != nil {
			return err
		}
		s.SetFault(jirastub.Fault{Path: path, Status: 302, Location: location})
		return nil
	})
	ctx.Step(`^the stub Jira server answers "([^"]*)" with an oversize body$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, Oversize: true})
	})
	ctx.Step(`^the stub Jira server answers "([^"]*)" with a body that is not JSON$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, NonJSON: true})
	})
	ctx.Step(`^the stub Jira server answers "([^"]*)" with a body that lacks its members$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, MissingMember: true})
	})
	ctx.Step(`^the stub Jira server repeats the page token on "([^"]*)"$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, RepeatToken: true})
	})
	ctx.Step(`^the stub Jira server answers "([^"]*)" with an empty page that still has a page token$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, EmptyPageWithToken: true})
	})
	ctx.Step(`^the stub Jira server never answers "([^"]*)"$`, func(path string) error {
		return setJiraFault(cur(), jirastub.Fault{Path: path, Hang: true})
	})
	ctx.Step(`^the stub Jira server answers every request with status (\d+) and Retry-After "([^"]*)"$`,
		func(status int, retryAfter string) error {
			return setJiraFault(cur(), jirastub.Fault{Status: status, RetryAfter: retryAfter})
		})
	ctx.Step(`^no stub Jira server is listening$`, func() error {
		if _, err := cur().jiraStub(); err != nil {
			return err
		}
		cur().stubs.jiraSrv.Close()
		return nil
	})

	ctx.Step(`^the stub Kiwi server never answers "([^"]*)"$`, func(method string) error {
		return setKiwiFault(cur(), method, kiwistub.Fault{Never: true})
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with an oversize body$`, func(method string) error {
		return setKiwiFault(cur(), method, kiwistub.Fault{Oversize: true})
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with a body that is not JSON$`, func(method string) error {
		return setKiwiFault(cur(), method, kiwistub.Fault{NotJSON: true})
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with a reply that has another request id$`, func(method string) error {
		return setKiwiFault(cur(), method, kiwistub.Fault{WrongID: true})
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with a reply that has neither a result nor an error$`, func(method string) error {
		return setKiwiFault(cur(), method, kiwistub.Fault{NoResultError: true})
	})
	ctx.Step(`^the stub Kiwi server answers "([^"]*)" with status (\d+) and Retry-After "([^"]*)"$`,
		func(method string, status int, retryAfter string) error {
			return setKiwiFault(cur(), method, kiwistub.Fault{Status: status, RetryAfter: retryAfter})
		})
	ctx.Step(`^the stub Kiwi server answers the first request to "([^"]*)" with status (\d+) and Retry-After "([^"]*)"$`,
		func(method string, status int, retryAfter string) error {
			s, err := cur().kiwiStub()
			if err != nil {
				return err
			}
			s.SetMethodFaultFirst(kiwiMethod(method), kiwistub.Fault{Status: status, RetryAfter: retryAfter})
			return nil
		})
}

// setJiraFault starts the Jira stub if needed and gives it one fault.
func setJiraFault(w *world, f jirastub.Fault) error {
	s, err := w.jiraStub()
	if err != nil {
		return err
	}
	s.SetFault(f)
	return nil
}

// setKiwiFault starts the Kiwi stub if needed and gives one of its methods a fault.
func setKiwiFault(w *world, method string, f kiwistub.Fault) error {
	s, err := w.kiwiStub()
	if err != nil {
		return err
	}
	s.SetMethodFault(kiwiMethod(method), f)
	return nil
}
