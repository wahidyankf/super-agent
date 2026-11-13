package jira

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// jiraNeverDialed is a dialer that a test sets but never calls. A transport with a dialer keeps its TLS settings unset.
func jiraNeverDialed(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("the test never dials")
}

// jiraReplyRT answers every request with the same 200 body, so a test controls the reply each call decodes.
func jiraReplyRT(body string) jiraFaultRT {
	return func(*http.Request) (*http.Response, error) { return jiraFaultReply(http.StatusOK, body), nil }
}

// jiraFailingRT answers every request with a transport fault.
func jiraFailingRT() jiraFaultRT {
	return func(*http.Request) (*http.Response, error) { return nil, jiraFault }
}

// TestJiraTransportAndBaseHelpers checks the transport and base rules that a request depends on: a base the parser rejects
// is refused, a transport with no TLS settings gets the host's TLS name, and a replaced default transport still yields a
// copy with no proxy.
func TestJiraTransportAndBaseHelpers(t *testing.T) {
	if err := CheckBaseURL("https://[::1"); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("CheckBaseURL error = %v, want errInvalidBaseURL", err)
	}
	tr, ok := hardenTransport(&http.Transport{DialContext: jiraNeverDialed}, "jira.example.test").(*http.Transport)
	if !ok || tr.TLSClientConfig == nil || tr.TLSClientConfig.MinVersion == 0 || tr.Proxy != nil {
		t.Fatalf("hardenTransport = %+v; want a transport with TLS settings and no proxy", tr)
	}
	old := http.DefaultTransport
	http.DefaultTransport = jiraFailingRT()
	t.Cleanup(func() { http.DefaultTransport = old })
	if cp := defaultTransportCopy(); cp.Proxy != nil {
		t.Fatalf("defaultTransportCopy with a replaced default has a proxy")
	}
}

// TestJiraEpochMillisRefusesAValueThatIsNotANumber checks that a time value that is not a number is not read as one.
func TestJiraEpochMillisRefusesAValueThatIsNotANumber(t *testing.T) {
	for _, raw := range []any{"not a number", true, nil} {
		if v, ok := epochMillis(raw); ok || v != 0 {
			t.Fatalf("epochMillis(%v) = %d, %v; want no value", raw, v, ok)
		}
	}
}

// TestJiraSinkIgnoresAnEmptyPiece checks that an empty piece adds no text to the sink.
func TestJiraSinkIgnoresAnEmptyPiece(t *testing.T) {
	var k sink
	k.appendRaw("")
	if k.b.Len() != 0 {
		t.Fatalf("sink holds %q after an empty piece", k.b.String())
	}
}

// TestJiraADFTextThatIsNotAStringIsUnexpected checks a JSON string that cannot be read as text: the ADF is refused.
func TestJiraADFTextThatIsNotAStringIsUnexpected(t *testing.T) {
	if _, err := parseADF([]byte(`"unterminated`)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("parseADF error = %v, want ErrUnexpectedResponse", err)
	}
}

// TestJiraProjectsRefuseAPageWithNoValuesOrNoLastFlag checks that a project page must hold its values and its last flag.
func TestJiraProjectsRefuseAPageWithNoValuesOrNoLastFlag(t *testing.T) {
	if _, _, err := (projectsReply{}).projects(); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("projects error = %v, want ErrUnexpectedResponse", err)
	}
}

// TestJiraLineAndParentHelpersOfAnIssue checks the markdown helpers that read the project and the parent of an issue: a
// project with no name is its key alone, and a parent with a key that does not parse names no parent.
func TestJiraLineAndParentHelpersOfAnIssue(t *testing.T) {
	if got := projectLine("PRJ-1", &issueFields{Project: &keyed{}}); got != "PRJ" {
		t.Fatalf("projectLine with no name = %q, want the key PRJ", got)
	}
	if got := parentKey(&issueFields{Parent: &keyed{Key: "bad"}}); got != "" {
		t.Fatalf("parentKey of a malformed key = %q, want none", got)
	}
}

// TestJiraIssueWithADFThatIsNotTextIsRefused checks an issue whose description is neither an object nor a string: the
// issue's document is refused, and the description is never read as text.
func TestJiraIssueWithADFThatIsNotTextIsRefused(t *testing.T) {
	gw := jiraFaultGateway(t, jiraReplyRT(`{"id":"1","key":"PRJ-1","fields":{"summary":"s","description":123}}`))
	if _, err := gw.issueDocument(context.Background(), "PRJ-1"); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("issueDocument error = %v, want ErrUnexpectedResponse", err)
	}
}

// TestJiraIssueKeyIsCheckedBeforeAnyRequest checks that a bare issue key that does not parse is refused before a request.
func TestJiraIssueKeyIsCheckedBeforeAnyRequest(t *testing.T) {
	gw := jiraFaultGateway(t, jiraFailingRT())
	if _, err := gw.issueDocument(context.Background(), "bad"); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("issueDocument error = %v, want ErrInvalidDocumentID", err)
	}
	if _, err := gw.commentsList(context.Background(), domain.NewArgs(map[string]any{"issueKey": "bad"})); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("commentsList error = %v, want ErrInvalidDocumentID", err)
	}
}

// TestJiraReadersReturnTheFaultOfTheirCall checks the readers that make one call: each returns the call's fault, and a
// read operation outside the table is unsupported.
func TestJiraReadersReturnTheFaultOfTheirCall(t *testing.T) {
	gw := jiraFaultGateway(t, jiraFailingRT())
	ctx := context.Background()
	if _, err := gw.whoami(ctx); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("whoami error = %v, want ErrUnreachable", err)
	}
	if _, err := gw.projectsList(ctx, domain.NewArgs(map[string]any{})); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("projectsList error = %v, want ErrUnreachable", err)
	}
	if _, _, err := gw.projectPage(ctx, 0, 1, ""); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("projectPage error = %v, want ErrUnreachable", err)
	}
	if _, err := gw.readBound(ctx, domain.OpKiwiProductsList, domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("readBound error = %v, want ErrUnsupportedOperation", err)
	}
}

// TestJiraProjectPageKeepsOnlyTheRequestedCount checks that a page longer than the requested count is cut to that count,
// and that a cut page reports that more projects follow, even when the server said it was the last page.
func TestJiraProjectPageKeepsOnlyTheRequestedCount(t *testing.T) {
	gw := jiraFaultGateway(t, jiraReplyRT(`{"values":[{"key":"ABC","name":"A"},{"key":"DEF","name":"B"}],"isLast":true}`))
	items, more, err := gw.projectPage(context.Background(), 0, 1, "")
	if err != nil {
		t.Fatalf("projectPage: %v", err)
	}
	if len(items) != 1 || items[0].Key != "ABC" || !more {
		t.Fatalf("projectPage = %+v, more %v; want the first project and more to follow", items, more)
	}
}

// TestJiraTreeIssuesRefuseAMalformedPage checks each step of the tree search: a failed call and a page whose issues are not
// usable are returned, and a page with too many issues and no further page is unexpected.
func TestJiraTreeIssuesRefuseAMalformedPage(t *testing.T) {
	ctx := context.Background()
	if _, err := jiraFaultGateway(t, jiraFailingRT()).treeIssues(ctx, "project = PRJ"); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("treeIssues over a failed call = %v, want ErrUnreachable", err)
	}
	noFields := jiraFaultGateway(t, jiraReplyRT(`{"issues":[{"id":"1","key":"PRJ-1"}],"isLast":true}`))
	if _, err := noFields.treeIssues(ctx, "project = PRJ"); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("treeIssues over an issue with no fields = %v, want ErrUnexpectedResponse", err)
	}
	issue := `{"id":"1","key":"PRJ-1","fields":{"summary":"s"}}`
	many := make([]string, treePageSize+1)
	for i := range many {
		many[i] = issue
	}
	overfull := jiraFaultGateway(t, jiraReplyRT(`{"issues":[`+strings.Join(many, ",")+`],"isLast":true}`))
	if _, err := overfull.treeIssues(ctx, "project = PRJ"); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("treeIssues over a page past the cap = %v, want ErrUnexpectedResponse", err)
	}
}

// adfNested wraps one block in depth blockquotes, so the block sits at the renderer's depth limit.
func adfNested(depth int, block string) string {
	return `{"type":"doc","version":1,"content":[` +
		strings.Repeat(`{"type":"blockquote","content":[`, depth) + block + strings.Repeat(`]}`, depth) + `]}`
}

// TestJiraADFAtTheDepthLimitIsCutNotFailed checks the renderer's depth limit for each block kind: a list, a task list, a
// table, and a paragraph whose children lie past the deepest rendered node are cut, and the document still renders.
func TestJiraADFAtTheDepthLimitIsCutNotFailed(t *testing.T) {
	const text = `{"type":"text","text":"x"}`
	paragraph := `{"type":"paragraph","content":[` + text + `]}`
	tests := []struct {
		name  string
		block string
	}{
		{"a bullet list past the depth limit", `{"type":"bulletList","content":[{"type":"listItem","content":[` + paragraph + `]}]}`},
		{"a task list past the depth limit", `{"type":"taskList","content":[{"type":"taskItem","attrs":{"state":"TODO"},"content":[` + text + `]}]}`},
		{"a table past the depth limit", `{"type":"table","content":[{"type":"tableRow","content":[{"type":"tableCell","content":[` + paragraph + `]}]}]}`},
		{"a paragraph past the depth limit", paragraph},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := RenderADF(json.RawMessage(adfNested(30, tt.block))); err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
		})
	}
}
