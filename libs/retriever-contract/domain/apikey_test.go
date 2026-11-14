package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// sentinel is a unique key value. No output of any kind may contain it.
const sentinel = "sentinel-key-7f3a9c-do-not-print"

// redactedText is the text every formatting path of ApiKey writes instead of the key.
const redactedText = "[REDACTED]"

type keyHolder struct {
	Key ApiKey
}

func TestApiKeyNeverShowsTheKeyThroughStandardFormatting(t *testing.T) {
	key, err := NewApiKey("EXAMPLE_OUTLINE_API_KEY", sentinel)
	if err != nil {
		t.Fatalf("NewApiKey: %v", err)
	}
	tests := []struct {
		name   string
		render func() string
	}{
		{name: "String method", render: key.String},
		{name: "GoString method", render: key.GoString},
		{name: "%v verb", render: func() string { return fmt.Sprintf("%v", key) }},
		{name: "%+v verb", render: func() string { return fmt.Sprintf("%+v", key) }},
		{name: "%#v verb", render: func() string { return fmt.Sprintf("%#v", key) }},
		{name: "%s verb", render: func() string { return fmt.Sprintf("%s", key) }},
		{name: "%q verb", render: func() string { return fmt.Sprintf("%q", key) }},
		{name: "%x verb", render: func() string { return fmt.Sprintf("%x", key) }},
		{name: "pointer with %v", render: func() string { return fmt.Sprintf("%v", &key) }},
		{name: "exported struct field with %+v", render: func() string { return fmt.Sprintf("%+v", keyHolder{Key: key}) }},
		{name: "exported struct field with %#v", render: func() string { return fmt.Sprintf("%#v", keyHolder{Key: key}) }},
		{name: "error text built with %v", render: func() string { return fmt.Errorf("auth: %v", key).Error() }},
		{name: "error text built with %s", render: func() string { return fmt.Errorf("auth: %s", key).Error() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.render()
			if strings.Contains(out, sentinel) {
				t.Fatalf("key leaked through %s: %q", tt.name, out)
			}
		})
	}
}

func TestApiKeyShowsRedactedTextThroughEachFormattingPath(t *testing.T) {
	key, err := NewApiKey("EXAMPLE_OUTLINE_API_KEY", sentinel)
	if err != nil {
		t.Fatalf("NewApiKey: %v", err)
	}
	tests := []struct {
		name string
		out  string
	}{
		{name: "String", out: key.String()},
		{name: "GoString", out: key.GoString()},
		{name: "%v", out: fmt.Sprintf("%v", key)},
		{name: "%+v", out: fmt.Sprintf("%+v", key)},
		{name: "%#v", out: fmt.Sprintf("%#v", key)},
		{name: "%s", out: fmt.Sprintf("%s", key)},
		{name: "%q", out: fmt.Sprintf("%q", key)},
		{name: "%x", out: fmt.Sprintf("%x", key)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.out, redactedText) {
				t.Fatalf("want redactedText text %q, got %q", redactedText, tt.out)
			}
		})
	}
}

func TestApiKeyNeverShowsTheKeyThroughJSONAndText(t *testing.T) {
	key, err := NewApiKey("EXAMPLE_OUTLINE_API_KEY", sentinel)
	if err != nil {
		t.Fatalf("NewApiKey: %v", err)
	}
	jsonOut, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	holderOut, err := json.Marshal(keyHolder{Key: key})
	if err != nil {
		t.Fatalf("json.Marshal holder: %v", err)
	}
	text, err := key.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	for name, out := range map[string]string{
		"json.Marshal":      string(jsonOut),
		"json holder field": string(holderOut),
		"MarshalText":       string(text),
	} {
		if strings.Contains(out, sentinel) {
			t.Fatalf("key leaked through %s: %q", name, out)
		}
	}
	if string(jsonOut) != `"`+redactedText+`"` {
		t.Fatalf("want the redactedText JSON string, got %s", jsonOut)
	}
	if string(text) != redactedText {
		t.Fatalf("want the redactedText text from MarshalText, got %q", text)
	}
}

func TestApiKeyNeverShowsTheKeyThroughSlog(t *testing.T) {
	key, err := NewApiKey("EXAMPLE_OUTLINE_API_KEY", sentinel)
	if err != nil {
		t.Fatalf("NewApiKey: %v", err)
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("calling Outline", "key", key, "holder", keyHolder{Key: key})
	logger.Info("calling Outline again", slog.Any("key", key))

	out := buf.String()
	if strings.Contains(out, sentinel) {
		t.Fatalf("key leaked through slog: %q", out)
	}
	if !strings.Contains(out, redactedText) {
		t.Fatalf("want redactedText text in the slog output, got %q", out)
	}
}

func TestNewApiKeyRejectsAnEmptyValueAndNamesOnlyTheVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
		wantErr  bool
	}{
		{name: "a set value is accepted", variable: "EXAMPLE_OUTLINE_API_KEY", value: sentinel, wantErr: false},
		{name: "an empty value counts as missing", variable: "EXAMPLE_OUTLINE_API_KEY", value: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewApiKey(tt.variable, tt.value)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("want the key accepted, got %v", err)
				}
				return
			}
			var missing MissingKeyError
			if !errors.As(err, &missing) || missing.Variable != tt.variable {
				t.Fatalf("want a MissingKeyError naming %s, got %v", tt.variable, err)
			}
			if err.Error() != "EXAMPLE_OUTLINE_API_KEY is not set" {
				t.Fatalf("want the variable named and no value, got %q", err.Error())
			}
		})
	}
}

func TestApiKeyRevealReturnsTheValueOnlyForTheAdapter(t *testing.T) {
	key, err := NewApiKey("EXAMPLE_OUTLINE_API_KEY", sentinel)
	if err != nil {
		t.Fatalf("NewApiKey: %v", err)
	}
	if key.Reveal() != sentinel {
		t.Fatalf("Reveal must return the stored value")
	}
}
