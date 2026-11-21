package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// wireADFOps are the operations of the ADF bindings: a read with a plain ADF field and a nested ADF field, and a write
// whose reply holds an ADF field. The examples are the replies the fixture would give.
const wireADFOps = `  - id: example-wiki.docs.adf
    summary: Get one document with its ADF body as Markdown.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        body:
          pointer: /body
          convert: markdown-from-adf
        meta:
          object:
            notes:
              pointer: /meta/notes
              convert: markdown-from-adf
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body:
              type: doc
              version: 1
              content:
                - type: paragraph
                  content:
                    - type: text
                      text: Hello
        expect:
          title: First document
          body: Hello
  - id: example-wiki.docs.draft
    summary: Create a draft whose reply holds an ADF body.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Name
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields:
        body:
          pointer: /body
          convert: markdown-from-adf
    examples:
      - request:
          title: First document
        response:
          status: 201
          body:
            body:
              type: doc
              version: 1
              content:
                - type: paragraph
                  content:
                    - type: text
                      text: Draft
        expect:
          body: Draft
`

// wireADFDoc wraps one ADF node in a document, so a rendering binding can name the node it checks.
func wireADFDoc(node string) []byte {
	return []byte(`{"type":"doc","version":1,"content":[` + node + `]}`)
}

// wireParagraph returns a paragraph whose one text node holds text.
func wireParagraph(text string) string {
	return `{"type":"paragraph","content":[{"type":"text","text":` + quoteJSON(text) + `}]}`
}

// quoteJSON returns text as a JSON string, so a binding can plant markup in an ADF text node.
func quoteJSON(text string) string {
	b, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestADFReadMapsTheDocumentInPlainAndNestedFields reads a document whose body is a plain ADF field and whose notes are
// an ADF field of a nested record.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: An ADF read maps the document to Markdown in a plain field and in a nested record
func TestADFReadMapsTheDocumentInPlainAndNestedFields(t *testing.T) {
	s := rebindSpec(t, wireADFOps)
	rt := rebindReplies(rebindJSON(200, `{"title":"First document",`+
		`"body":{"type":"doc","version":1,"content":[`+wireParagraph("Hello")+`]},`+
		`"meta":{"notes":{"type":"doc","version":1,"content":[`+wireParagraph("Checked")+`]}}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("body"); got != "Hello" {
		t.Fatalf("body = %q, want the Markdown of the document", got)
	}
	if got := rec.Rec("meta").Str("notes"); got != "Checked" {
		t.Fatalf("meta.notes = %q, want the Markdown of the nested document", got)
	}
}

// TestADFWriteMapsTheReplyDocument runs a write whose reply holds an ADF body.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: An ADF write maps the reply document to Markdown
func TestADFWriteMapsTheReplyDocument(t *testing.T) {
	s := rebindSpec(t, wireADFOps)
	rt := rebindReplies(rebindJSON(201, `{"body":{"type":"doc","version":1,"content":[`+wireParagraph("Draft")+`]}}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Write(context.Background(), "example-wiki.docs.draft", domain.NewArgs(map[string]any{"title": "First document"}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("body"); got != "Draft" {
		t.Fatalf("body = %q, want the Markdown of the reply document", got)
	}
}

// TestADFMissingOptionalFieldIsNull reads a document whose optional ADF field is missing from the reply.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: An ADF field missing from the reply maps to null like any other optional field
func TestADFMissingOptionalFieldIsNull(t *testing.T) {
	s := rebindSpec(t, wireADFOps)
	rt := rebindReplies(rebindJSON(200, `{"title":"First document"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	v, present := rec.Get("body")
	if !present || v != nil {
		t.Fatalf("body = %v (present %v), want null", v, present)
	}
}

// TestADFValueThatIsNotADocumentFailsWithoutNamingIt reads a document whose ADF field holds a number.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: A reply value that is not an ADF document fails the read without naming the value
func TestADFValueThatIsNotADocumentFailsWithoutNamingIt(t *testing.T) {
	const planted = "98765"
	s := rebindSpec(t, wireADFOps)
	rt := rebindReplies(rebindJSON(200, `{"title":"First document","body":`+planted+`}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if strings.Contains(err.Error(), planted) {
		t.Fatalf("the failure names the value: %q", err.Error())
	}
}

// TestADFBlockNodesRenderAsMarkdown renders one document of each block node on the allow-list.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: Each block node on the allow-list renders as its Markdown form
func TestADFBlockNodesRenderAsMarkdown(t *testing.T) {
	cases := []struct {
		node string
		json string
		want string
	}{
		{"paragraph", wireParagraph("Plain words"), "Plain words"},
		{"heading", `{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]}`, "### Title"},
		{"bullet list", `{"type":"bulletList","content":[{"type":"listItem","content":[` + wireParagraph("Item one") + `]}]}`, "- Item one"},
		{"ordered list", `{"type":"orderedList","attrs":{"order":1},"content":[{"type":"listItem","content":[` + wireParagraph("First step") + `]}]}`, "1. First step"},
		{"code block", `{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"x := 1"}]}`, "```go\nx := 1\n```"},
		{"blockquote", `{"type":"blockquote","content":[` + wireParagraph("Quoted words") + `]}`, "> Quoted words"},
		{"rule", `{"type":"rule"}`, "---"},
	}
	for _, c := range cases {
		t.Run(c.node, func(t *testing.T) {
			out, err := engine.RenderADF(wireADFDoc(c.json))
			if err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
			if !strings.Contains(out, c.want) {
				t.Fatalf("output %q does not hold the Markdown form %q", out, c.want)
			}
		})
	}
}

// TestADFInlineNodesAndMarksRenderAsMarkdown renders one inline node or mark of each kind on the allow-list.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: Each inline node and mark on the allow-list renders as its Markdown form
func TestADFInlineNodesAndMarksRenderAsMarkdown(t *testing.T) {
	cases := []struct {
		inline string
		json   string
		want   string
	}{
		{"strong mark", `{"type":"text","text":"bold","marks":[{"type":"strong"}]}`, "**bold**"},
		{"em mark", `{"type":"text","text":"it","marks":[{"type":"em"}]}`, "*it*"},
		{"strike mark", `{"type":"text","text":"gone","marks":[{"type":"strike"}]}`, "~~gone~~"},
		{"link mark", `{"type":"text","text":"site","marks":[{"type":"link","attrs":{"href":"https://docs.example.test/x"}}]}`, "[site](<https://docs.example.test/x>)"},
		{"mention", `{"type":"mention","attrs":{"text":"Ana"}}`, "@Ana"},
		{"emoji", `{"type":"emoji","attrs":{"shortName":":smile:"}}`, ":smile:"},
		{"hardBreak", `{"type":"text","text":"line one"},{"type":"hardBreak"},{"type":"text","text":"line two"}`, "line one\\\nline two"},
		{"status", `{"type":"status","attrs":{"text":"In progress"}}`, "[In progress]"},
		{"date", `{"type":"date","attrs":{"timestamp":"1767225600000"}}`, "2026-01-01"},
	}
	for _, c := range cases {
		t.Run(c.inline, func(t *testing.T) {
			doc := `{"type":"paragraph","content":[` + c.json + `]}`
			out, err := engine.RenderADF(wireADFDoc(doc))
			if err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
			if !strings.Contains(out, c.want) {
				t.Fatalf("output %q does not hold the Markdown form %q", out, c.want)
			}
		})
	}
}

// TestADFTextMarkupIsEscapedAndControlsRemoved renders a text that holds markup and a control character.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: Text with markup in an ADF document is escaped and cleaned in the output
func TestADFTextMarkupIsEscapedAndControlsRemoved(t *testing.T) {
	out, err := engine.RenderADF(wireADFDoc(wireParagraph("*bold* [x](y) <tag> \u0001hidden")))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	for _, want := range []string{`\*bold\*`, `\[x\]`, `\<tag\>`} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q is not escaped: want %q", out, want)
		}
	}
	if strings.ContainsRune(out, '\u0001') {
		t.Fatalf("output %q keeps the control character", out)
	}
}

// TestADFInputThatIsNotADocumentNodeFails renders input that is not JSON, and JSON that is not a node.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: Input that is not JSON, or is JSON that is not a node, fails as an unexpected response
func TestADFInputThatIsNotADocumentNodeFails(t *testing.T) {
	for _, raw := range []string{"not json", "42", "[1,2]"} {
		_, err := engine.RenderADF(json.RawMessage(raw))
		if !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("RenderADF(%q) err = %v, want ErrUnexpectedResponse", raw, err)
		}
	}
}

// TestADFTreeDeeperThanTheLimitIsCut renders a document nested past the depth limit.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: A tree deeper than the depth limit renders what fits and ends with the truncation marker
func TestADFTreeDeeperThanTheLimitIsCut(t *testing.T) {
	const deep = "deep-text"
	nested := strings.Repeat(`{"type":"blockquote","content":[`, 40) + wireParagraph(deep) + strings.Repeat("]}", 40)
	out, err := engine.RenderADF(wireADFDoc(wireParagraph("top") + "," + nested))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if !strings.HasSuffix(out, "[truncated]") {
		t.Fatalf("output %q does not end with the truncation marker", out)
	}
	if strings.Contains(out, deep) {
		t.Fatalf("output %q shows text below the depth limit", out)
	}
	if !strings.Contains(out, "top") {
		t.Fatalf("output %q lost the text above the limit", out)
	}
}

// TestADFMoreNodesThanTheLimitSayItStopped renders a document with more nodes than the node limit.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: More nodes than the node limit render what fits and say so
func TestADFMoreNodesThanTheLimitSayItStopped(t *testing.T) {
	const piece = `{"type":"paragraph","content":[{"type":"text","text":"p"}]}`
	doc := `{"type":"doc","content":[` + strings.Repeat(piece+",", 100009) + piece + `]}`
	out, err := engine.RenderADF(json.RawMessage(doc))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if !strings.HasPrefix(out, "p") {
		t.Fatalf("output does not start with the nodes that fit: %q", out[:min(len(out), 20)])
	}
	if !strings.HasSuffix(out, "[truncated]") {
		t.Fatal("output does not say that it stopped")
	}
}

// TestADFTextOverTheOutputCapIsCutAtAWholeCharacter renders a text longer than the output cap.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: A text longer than the output cap is cut at a whole character below the cap with the marker
func TestADFTextOverTheOutputCapIsCutAtAWholeCharacter(t *testing.T) {
	text := strings.Repeat("é", 600000)
	out, err := engine.RenderADF(wireADFDoc(wireParagraph(text)))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if len(out) > engine.MaxTextBytes {
		t.Fatalf("output is %d bytes, over the %d cap", len(out), engine.MaxTextBytes)
	}
	if !utf8.ValidString(out) {
		t.Fatal("output is cut inside a character")
	}
	if !strings.HasSuffix(out, "[truncated]") {
		t.Fatal("output does not end with the marker")
	}
}

// TestADFCodeBlockThatDoesNotFitIsDropped renders a code block that does not fit the output that is left.
//
// @scenario specs/libs/retriever-engine/behaviours/converters-adf.feature: A code block that does not fit the output is dropped whole and never written half-open
func TestADFCodeBlockThatDoesNotFitIsDropped(t *testing.T) {
	const capRoom = 1 << 20
	// The paragraph leaves 100 bytes of room under the output cap, which is less than the code block needs.
	text := strings.Repeat("a", capRoom-13-2-100)
	code := `{"type":"codeBlock","content":[{"type":"text","text":"` + strings.Repeat("x", 200) + `"}]}`
	out, err := engine.RenderADF(wireADFDoc(wireParagraph(text) + "," + code))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if strings.Contains(out, "```") {
		t.Fatal("the code block was written, though it does not fit")
	}
	if !strings.HasSuffix(out, "[truncated]") {
		t.Fatal("output does not end with the marker")
	}
}
