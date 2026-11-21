package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireCapped returns a validated spec of the rebind header with the reply cap set to capBytes, followed by the operations.
// A binding uses it when the test needs a reply budget smaller than the default 10 MiB.
func wireCapped(t *testing.T, capBytes int, operations string) *spec.Spec {
	t.Helper()
	header := wireReplace(t, rebindHeader, "timeout_seconds: 20", fmt.Sprintf("timeout_seconds: 20\n  max_reply_bytes: %d", capBytes))
	return rebindDoc(t, header+operations)
}

// wireEditorOps are the editor-data operations: a POST read whose text field converts the editor data, with a fallback
// to the raw text, and a second field that converts the same editor data, so that a binding can make the shared budget
// run out.
const wireEditorOps = `  - id: example-wiki.comments.get
    summary: Read one comment.
    access: read
    method: POST
    read_over_post: true
    path: /comments/info
    params:
      - name: id
        type: ID
        in: body
        required: true
    body:
      id: "{{id}}"
    response:
      fields:
        id:
          pointer: /data/id
          required: true
        text:
          pointer: /data/data
          fallback_pointer: /data/text
          convert: text-from-editor-data
        again:
          pointer: /data/data
          convert: text-from-editor-data
    examples:
      - request:
          id: c1
        response:
          status: 200
          body:
            data:
              id: c1
              text: Nice
        expect:
          id: c1
          text: Nice
`

// wireHostOps is the hostname operation: a list of webhooks whose address fields keep only the host name.
const wireHostOps = `  - id: example-wiki.hooks.list
    summary: List the webhook subscriptions of the wiki.
    access: read
    method: GET
    path: /hooks
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
          format: id
        urlHost:
          pointer: /url
          convert: hostname
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: h1
                url: https://hooks.example.test/x
        expect:
          id: h1
          urlHost: hooks.example.test
`

// wireJiraOps is the Jira text operation: a read of one issue whose timestamp, date, and title use the Jira converters,
// and whose title is also mapped a second time, so that a binding can make the shared budget run out.
const wireJiraOps = `  - id: example-wiki.issues.get
    summary: Read one issue.
    access: read
    method: GET
    path: /issues/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        updated:
          pointer: /fields/updated
          convert: jira-time
        due:
          pointer: /fields/duedate
          convert: jira-date
        summary:
          pointer: /fields/summary
          convert: jira-title
        summary_copy:
          pointer: /fields/summary
          convert: jira-title
        meta:
          object:
            notes:
              pointer: /fields/notes
              convert: jira-title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            fields:
              updated: "2026-11-02T09:15:00.000+0700"
              duedate: "2026-12-01"
              summary: First issue
        expect:
          updated: "2026-11-02T02:15:00Z"
          due: "2026-12-01"
          summary: First issue
`

// wireEditorBody returns an editor reply whose text is the given words in one paragraph.
func wireEditorBody(words string) string {
	return `{"data":{"id":"c1","data":{"type":"doc","content":[` + wireParagraph(words) + `]}}}`
}

// TestEditorDataMapsToPlainWordsWithMentionLabels maps editor data with two paragraphs and a mention.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: Editor data in a reply maps to the plain words of the data, one block per line, with the mention label
func TestEditorDataMapsToPlainWordsWithMentionLabels(t *testing.T) {
	s := rebindSpec(t, wireEditorOps)
	body := `{"data":{"id":"c1","data":{"type":"doc","content":[` +
		wireParagraph("Hello") + `,` +
		`{"type":"paragraph","content":[{"type":"text","text":"Hi "},{"type":"mention","attrs":{"label":"Ana"}}]}` +
		`]}}}`
	rt := rebindReplies(rebindJSON(200, body))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := rec.Str("text")
	if got != "Hello\nHi @Ana" {
		t.Fatalf("text = %q, want the plain words, one block per line, with the mention label", got)
	}
	if strings.ContainsAny(got, "{}") || strings.Contains(got, "paragraph") {
		t.Fatalf("the editor JSON reached the record: %q", got)
	}
}

// TestEditorRawTextPassesThroughTheFallbackPointer replies with plain text and no editor data.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A reply with no editor data maps the raw text unchanged through the fallback pointer
func TestEditorRawTextPassesThroughTheFallbackPointer(t *testing.T) {
	s := rebindSpec(t, wireEditorOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"id":"c1","text":"Nice"}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("text"); got != "Nice" {
		t.Fatalf("text = %q, want the raw text unchanged", got)
	}
}

// TestEditorDataThatIsMissingOrNullMapsToNull reads a comment whose editor field is missing, or present and null.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: An editor-data field that is <state> maps to null with no partial text
func TestEditorDataThatIsMissingOrNullMapsToNull(t *testing.T) {
	cases := []struct {
		state string
		body  string
	}{
		{"missing", `{"data":{"id":"c1"}}`},
		{"null", `{"data":{"id":"c1","data":null}}`},
	}
	for _, c := range cases {
		t.Run(c.state, func(t *testing.T) {
			s := rebindSpec(t, wireEditorOps)
			rt := rebindReplies(rebindJSON(200, c.body))
			eng := rebindEngine(t, s, rt, &rebindClock{})
			rec, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if v, _ := rec.Get("text"); v != nil {
				t.Fatalf("text = %v, want null", v)
			}
		})
	}
}

// TestEditorDataHoldingAScalarFailsAsUnexpected replies with a number or a flag where the editor data belongs.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: An editor-data field holding <value> fails the read as unexpected
func TestEditorDataHoldingAScalarFailsAsUnexpected(t *testing.T) {
	cases := []struct {
		value string
		body  string
	}{
		{"a number", `{"data":{"id":"c1","data":7}}`},
		{"a flag", `{"data":{"id":"c1","data":true}}`},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			s := rebindSpec(t, wireEditorOps)
			rt := rebindReplies(rebindJSON(200, c.body))
			eng := rebindEngine(t, s, rt, &rebindClock{})
			_, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// TestEditorTextOverTheBudgetFailsTheRead maps the same editor data into two fields of a read whose reply cap leaves room
// for only one copy of the text.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A converted editor text past the budget fails the read
func TestEditorTextOverTheBudgetFailsTheRead(t *testing.T) {
	text := strings.Repeat("a", 200)
	body := wireEditorBody(text)
	s := wireCapped(t, len(body), wireEditorOps)
	rt := rebindReplies(rebindJSON(200, body))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}

// TestHostnameConverterKeepsOnlyTheHostName maps an address with a user name, a port, a path, and a query.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: An address field maps to the host name of the address without the path, query, port, or user name
func TestHostnameConverterKeepsOnlyTheHostName(t *testing.T) {
	s := rebindSpec(t, wireHostOps)
	userInfo := "user" + ":" + "secret"
	rt := rebindReplies(rebindJSON(200, `{"items":[{"id":"h1","url":"https://`+userInfo+`@hooks.example.test:8443/x?y=1"}]}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Items()[0].Str("urlHost"); got != "hooks.example.test" {
		t.Fatalf("urlHost = %q, want the host name only", got)
	}
}

// TestHostnameConverterGivesEmptyTextForAGap replies with an address the converter cannot parse, or with no address.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: An address field that is <state> maps to the empty text
func TestHostnameConverterGivesEmptyTextForAGap(t *testing.T) {
	cases := []struct {
		state string
		body  string
	}{
		{"an address the converter cannot parse", `{"items":[{"id":"h1","url":"::not an address"}]}`},
		{"missing", `{"items":[{"id":"h1"}]}`},
	}
	for _, c := range cases {
		t.Run(c.state, func(t *testing.T) {
			s := rebindSpec(t, wireHostOps)
			rt := rebindReplies(rebindJSON(200, c.body))
			eng := rebindEngine(t, s, rt, &rebindClock{})
			rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			v, present := rec.Items()[0].Get("urlHost")
			if !present || v != "" {
				t.Fatalf("urlHost = %v (present %v), want the empty text", v, present)
			}
		})
	}
}

// TestJiraReadMapsTimestampDateAndTitle maps a timestamp, a date, and a title that holds a line break.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A Jira read maps the timestamp in UTC, the date as it is, and the title as one line
func TestJiraReadMapsTimestampDateAndTitle(t *testing.T) {
	s := rebindSpec(t, wireJiraOps)
	body := `{"fields":{"updated":"2026-11-02T09:15:00.000+0700","duedate":"2026-12-01","summary":"First\r\nissue"}}`
	rt := rebindReplies(rebindJSON(200, body))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("updated"); got != "2026-11-02T02:15:00Z" {
		t.Fatalf("updated = %q, want the RFC 3339 instant in UTC", got)
	}
	if got := rec.Str("due"); got != "2026-12-01" {
		t.Fatalf("due = %q, want the date unchanged", got)
	}
	if got := rec.Str("summary"); got != "First issue" {
		t.Fatalf("summary = %q, want one line", got)
	}
}

// TestJiraTitleCountsCharactersAndKeepsVisibleText maps a title longer than 255 characters, and a title with a line
// break, a control character, and a bidirectional override.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A Jira title counts its limit in characters and keeps its visible text
func TestJiraTitleCountsCharactersAndKeepsVisibleText(t *testing.T) {
	s := rebindSpec(t, wireJiraOps)
	long, err := json.Marshal(strings.Repeat("é", 300))
	if err != nil {
		t.Fatal(err)
	}
	rt := rebindReplies(rebindJSON(200, `{"fields":{"summary":`+string(long)+`}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len([]rune(rec.Str("summary"))); got != 255 {
		t.Fatalf("title holds %d characters, want at most 255", got)
	}

	tricky, err := json.Marshal("Line one\nLine two\u0007‮RTL")
	if err != nil {
		t.Fatal(err)
	}
	rt = rebindReplies(rebindJSON(200, `{"fields":{"summary":`+string(tricky)+`}}`))
	eng = rebindEngine(t, s, rt, &rebindClock{})
	rec, err = eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := rec.Str("summary")
	if got != "Line one Line twoRTL" {
		t.Fatalf("summary = %q, want the visible text on one line", got)
	}
}

// TestJiraDateThatIsAYearGivesEmptyTextAndKeepsTheField replies with a bare year as the date.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A Jira date that is a bare year maps to the empty text with the field present
func TestJiraDateThatIsAYearGivesEmptyTextAndKeepsTheField(t *testing.T) {
	s := rebindSpec(t, wireJiraOps)
	rt := rebindReplies(rebindJSON(200, `{"fields":{"duedate":"2026"}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	v, present := rec.Get("due")
	if !present || v != "" {
		t.Fatalf("due = %v (present %v), want the empty text with the field present", v, present)
	}
}

// TestJiraConvertersGiveEmptyTextForAValueOfAnotherShape replies with a number, a flag, an object, and a list.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A Jira converter that receives a value of another shape maps it to the empty text and keeps the row
func TestJiraConvertersGiveEmptyTextForAValueOfAnotherShape(t *testing.T) {
	s := rebindSpec(t, wireJiraOps)
	for _, shape := range []string{"5", "true", `{"a":1}`, `["x"]`} {
		body := `{"fields":{"updated":` + shape + `,"duedate":` + shape + `,"summary":` + shape + `}}`
		eng := rebindEngine(t, s, rebindReplies(rebindJSON(200, body)), &rebindClock{})
		rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
		if err != nil {
			t.Fatalf("Read with %s: %v", shape, err)
		}
		for _, name := range []string{"updated", "due", "summary"} {
			v, present := rec.Get(name)
			if !present || v != "" {
				t.Fatalf("%s with %s = %v (present %v), want the empty text", name, shape, v, present)
			}
		}
	}
}

// TestJiraConvertersGiveEmptyTextForAMissingOrNullValue replies with the Jira fields missing, and then null.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A Jira converter with a missing or null value maps to the empty text
func TestJiraConvertersGiveEmptyTextForAMissingOrNullValue(t *testing.T) {
	s := rebindSpec(t, wireJiraOps)
	for _, body := range []string{`{"fields":{}}`, `{"fields":{"updated":null,"duedate":null,"summary":null}}`} {
		eng := rebindEngine(t, s, rebindReplies(rebindJSON(200, body)), &rebindClock{})
		rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
		if err != nil {
			t.Fatalf("Read with %s: %v", body, err)
		}
		for _, name := range []string{"updated", "due", "summary"} {
			v, present := rec.Get(name)
			if !present || v != "" {
				t.Fatalf("%s with %s = %v (present %v), want the empty text", name, body, v, present)
			}
		}
	}
}

// TestJiraTextOverTheBudgetFailsTheRead maps one Jira title into two fields of a read whose reply cap leaves room for
// only one copy of the title.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-text-and-time.feature: A converted Jira text past the budget fails the read
func TestJiraTextOverTheBudgetFailsTheRead(t *testing.T) {
	title := strings.Repeat("\U0001F600", 255)
	body := `{"fields":{"summary":"` + title + `"}}`
	s := wireCapped(t, len(body)+3, wireJiraOps)
	eng := rebindEngine(t, s, rebindReplies(rebindJSON(200, body)), &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}
