package envfile

import (
	"fmt"
	"strings"
	"testing"
)

// sentinel stands in for a real key. No error, log line, or formatted output may contain it. The integration
// tests in this package share it.
const sentinel = "sentinel-outline-key-5c1e9d-do-not-print"

func TestLookupReadsOnlyTheNamedVariable(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "a plain value is read", content: outlineVariable + "=plain-value", want: "plain-value"},
		{name: "an export prefix is ignored", content: "export " + outlineVariable + "=exported-value", want: "exported-value"},
		{name: "an export prefix followed by a tab is ignored", content: "export\t" + outlineVariable + "=tab-value", want: "tab-value"},
		{name: "a double-quoted value keeps its spaces", content: outlineVariable + `="value with spaces"`, want: "value with spaces"},
		{name: "a single-quoted value is read literally", content: outlineVariable + `='single $HOME #kept'`, want: "single $HOME #kept"},
		{name: "trailing spaces after a quoted value are dropped", content: outlineVariable + `="quoted"   `, want: "quoted"},
		{name: "a comment after a quoted value is dropped", content: outlineVariable + `="quoted" # note`, want: "quoted"},
		{name: "trailing spaces after an unquoted value are dropped", content: outlineVariable + "=trimmed   ", want: "trimmed"},
		{name: "a comment after an unquoted value is dropped", content: outlineVariable + "=before # comment", want: "before"},
		{name: "a hash inside an unquoted value without a space is kept", content: outlineVariable + "=abc#def", want: "abc#def"},
		{name: "a value may contain equals signs", content: outlineVariable + "=a=b", want: "a=b"},
		{name: "spaces around the equals sign are allowed", content: outlineVariable + " = spaced", want: "spaced"},
		{name: "a full-line comment is skipped", content: "# " + outlineVariable + "=commented\n" + outlineVariable + "=real", want: "real"},
		{name: "an indented comment is skipped", content: "   # " + outlineVariable + "=commented\n" + outlineVariable + "=real", want: "real"},
		{name: "blank lines are skipped", content: "\n\n" + outlineVariable + "=real\n\n", want: "real"},
		{name: "a line of spaces is skipped", content: "   \n" + outlineVariable + "=real", want: "real"},
		{name: "a CRLF line ending is handled", content: outlineVariable + "=crlf\r\n", want: "crlf"},
		{name: "the last assignment wins", content: outlineVariable + "=first\n" + outlineVariable + "=second", want: "second"},
		{name: "other keys are ignored", content: "OTHER=x\n" + outlineVariable + "=real", want: "real"},
		{name: "a variable that is absent reads as empty", content: "OTHER=x", want: ""},
		{name: "an empty value reads as empty", content: outlineVariable + "=", want: ""},
		{name: "an empty file reads as empty", content: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lookup(strings.NewReader(tt.content), outlineVariable)
			if err != nil {
				t.Fatalf("lookup: %v", err)
			}
			if got != tt.want {
				t.Fatalf("lookup = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLookupRefusesMalformedLinesAndNamesTheLineNumber(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine string
	}{
		{name: "a line without an equals sign names its line", content: "OTHER=x\nnot a pair", wantLine: "line 2"},
		{name: "a line with an empty name names its line", content: outlineVariable + "=ok\n=value", wantLine: "line 2"},
		{name: "an unclosed double quote names its line", content: outlineVariable + `="open`, wantLine: "line 1"},
		{name: "an unclosed single quote names its line", content: "OTHER=1\n" + outlineVariable + "='open", wantLine: "line 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := lookup(strings.NewReader(tt.content), outlineVariable)
			if err == nil || !strings.Contains(err.Error(), tt.wantLine) {
				t.Fatalf("lookup error = %v, want an error naming %s", err, tt.wantLine)
			}
		})
	}
}

// TestLookupErrorsNeverEchoTheValue proves a malformed line that holds the sentinel is reported by its number only.
func TestLookupErrorsNeverEchoTheValue(t *testing.T) {
	contents := map[string]string{
		"a line without an equals sign": outlineVariable + "=" + sentinel + "\nbroken " + sentinel,
		"an unclosed quote":             outlineVariable + `="` + sentinel,
	}
	for name, content := range contents {
		t.Run(name, func(t *testing.T) {
			_, err := lookup(strings.NewReader(content), outlineVariable)
			if err == nil {
				t.Fatal("lookup accepted a malformed file, want an error")
			}
			for _, text := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err)} {
				if strings.Contains(text, sentinel) {
					t.Fatalf("error text leaks the value: %q", text)
				}
			}
		})
	}
}
