package behaviour

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// registerReadSteps binds the steps of the read scenarios of every source: the stub switches for an unoffered endpoint,
// a long collection list, full pages, editor-only documents, and a reply that ignores the API version, and the checks
// of one recorded request's JSON fields, API version, and query string. Each check reads the request the stub recorded.
func registerReadSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the stub Outline server holds (\d+) numbered collections$`, func(count string) error {
		n, err := strconv.Atoi(count)
		if err != nil {
			return err
		}
		w := cur()
		w.stubs.collections = outlinestub.NumberedCollections(n)
		w.syncOutline()
		return nil
	})
	ctx.Step(`^the stub Outline server holds no collections$`, func() error {
		w := cur()
		w.stubs.collections = nil
		w.syncOutline()
		return nil
	})
	ctx.Step(`^the stub Outline server returns full pages$`, func() error {
		cur().outlineStub().SetFullPages(true)
		return nil
	})
	ctx.Step(`^the stub Outline server does not offer "([^"]*)"$`, func(method string) error {
		cur().outlineStub().SetUnoffered(method)
		return nil
	})
	ctx.Step(`^the stub Outline server holds documents with editor data only$`, func() error {
		cur().outlineStub().SetEditorOnlyDocuments(true)
		return nil
	})
	ctx.Step(`^the stub Outline server ignores the X-API-Version header$`, func() error {
		cur().outlineStub().SetIgnoreAPIVersion(true)
		return nil
	})
	ctx.Step(`^the stub Outline server's last request to "([^"]*)" carried the API version "([^"]*)"$`,
		func(method, version string) error {
			last, err := lastOutlineRequest(cur(), method)
			if err != nil {
				return err
			}
			if last.APIVersion != version {
				return fmt.Errorf("the last request to %q carried the API version %q, not %q", method, last.APIVersion, version)
			}
			return nil
		})
	ctx.Step(`^the stub Outline server's last request to "([^"]*)" has the field "([^"]*)" equal to (.+)$`,
		func(method, field, want string) error {
			last, err := lastOutlineRequest(cur(), method)
			if err != nil {
				return err
			}
			return checkField(last.Body, field, want)
		})
	ctx.Step(`^the stub Jira server's last request to "([^"]*)" body has the field "([^"]*)" equal to (.+)$`,
		func(path, field, want string) error {
			last, err := lastJiraRequest(cur(), path)
			if err != nil {
				return err
			}
			return checkField(last.Body, field, want)
		})
	ctx.Step(`^the stub Jira server's last request to "([^"]*)" query contains (.+)$`, func(path, text string) error {
		last, err := lastJiraRequest(cur(), path)
		if err != nil {
			return err
		}
		if !strings.Contains(last.Query, text) {
			return fmt.Errorf("the last request to %q has the query %q, which does not contain %q", path, last.Query, text)
		}
		return nil
	})
}

// lastOutlineRequest returns the most recent request the Outline stub recorded for one method.
func lastOutlineRequest(w *world, method string) (outlinestub.Request, error) {
	var last *outlinestub.Request
	for _, r := range w.outlineStub().Requests() {
		if r.Method == method {
			rr := r
			last = &rr
		}
	}
	if last == nil {
		return outlinestub.Request{}, fmt.Errorf("the stub Outline server received no request to %q", method)
	}
	return *last, nil
}

// lastJiraRequest returns the most recent request the Jira stub recorded for one path.
func lastJiraRequest(w *world, path string) (jiraRequestView, error) {
	s, err := w.jiraStub()
	if err != nil {
		return jiraRequestView{}, err
	}
	var last *jiraRequestView
	for _, r := range s.Requests() {
		if r.Path == path {
			last = &jiraRequestView{Body: r.Body, Query: r.Query}
		}
	}
	if last == nil {
		return jiraRequestView{}, fmt.Errorf("the stub Jira server received no request to %q", path)
	}
	return *last, nil
}

// jiraRequestView is the part of a Jira request that the read steps check.
type jiraRequestView struct {
	Body  map[string]any
	Query string
}

// checkField reports an error unless the JSON object of a request holds the field with the wanted value. The wanted
// text is absent, true, false, a number, or a string, and a string is compared without quotes.
func checkField(body any, field, want string) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var fields map[string]any
	if err := dec.Decode(&fields); err != nil {
		return fmt.Errorf("the request body is not a JSON object: %w", err)
	}
	want = strings.TrimSpace(want)
	got, found := fields[field]
	if want == "absent" {
		if found {
			return fmt.Errorf("the request field %q is present, and it must be absent", field)
		}
		return nil
	}
	if !found {
		return fmt.Errorf("the request field %q is missing", field)
	}
	var text string
	switch v := got.(type) {
	case json.Number:
		text = v.String()
	case bool:
		text = strconv.FormatBool(v)
	case string:
		text = v
	default:
		return fmt.Errorf("the request field %q has a value that is not a scalar", field)
	}
	if text != want {
		return fmt.Errorf("the request field %q is %q, not %q", field, text, want)
	}
	return nil
}
