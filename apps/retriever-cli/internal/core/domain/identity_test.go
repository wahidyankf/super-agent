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

// identitySentinel is a unique synthetic address. No output of any kind may contain it.
const identitySentinel = "sentinel-mail-4c1e@example.com"

// identityVariable is the variable name used in the identity fixtures.
const identityVariable = "EXAMPLE_JIRA_EMAIL"

type identityHolder struct {
	Mail Identity
}

func mustIdentity(t *testing.T, value string) Identity {
	t.Helper()
	id, err := NewIdentity(identityVariable, value)
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	return id
}

func TestIdentityNeverShowsTheAddressThroughStandardFormatting(t *testing.T) {
	id := mustIdentity(t, identitySentinel)
	tests := []struct {
		name   string
		render func() string
	}{
		{name: "String method", render: id.String},
		{name: "GoString method", render: id.GoString},
		{name: "%v verb", render: func() string { return fmt.Sprintf("%v", id) }},
		{name: "%+v verb", render: func() string { return fmt.Sprintf("%+v", id) }},
		{name: "%#v verb", render: func() string { return fmt.Sprintf("%#v", id) }},
		{name: "%s verb", render: func() string { return fmt.Sprintf("%s", id) }},
		{name: "%q verb", render: func() string { return fmt.Sprintf("%q", id) }},
		{name: "%x verb", render: func() string { return fmt.Sprintf("%x", id) }},
		{name: "pointer with %v", render: func() string { return fmt.Sprintf("%v", &id) }},
		{name: "struct field with %+v", render: func() string { return fmt.Sprintf("%+v", identityHolder{Mail: id}) }},
		{name: "struct field with %#v", render: func() string { return fmt.Sprintf("%#v", identityHolder{Mail: id}) }},
		{name: "error text built with %v", render: func() string { return fmt.Errorf("auth: %v", id).Error() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.render()
			if strings.Contains(out, identitySentinel) {
				t.Fatalf("address leaked through %s: %q", tt.name, out)
			}
			if !strings.Contains(out, redactedText) {
				t.Fatalf("want %q through %s, got %q", redactedText, tt.name, out)
			}
		})
	}
}

func TestIdentityNeverShowsTheAddressThroughJSONAndText(t *testing.T) {
	id := mustIdentity(t, identitySentinel)
	jsonOut, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	holderOut, err := json.Marshal(identityHolder{Mail: id})
	if err != nil {
		t.Fatalf("json.Marshal holder: %v", err)
	}
	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	for name, out := range map[string]string{
		"json.Marshal":      string(jsonOut),
		"json holder field": string(holderOut),
		"MarshalText":       string(text),
	} {
		if strings.Contains(out, identitySentinel) {
			t.Fatalf("address leaked through %s: %q", name, out)
		}
	}
	if string(jsonOut) != `"`+redactedText+`"` {
		t.Fatalf("want the redacted JSON string, got %s", jsonOut)
	}
	if string(text) != redactedText {
		t.Fatalf("want the redacted text from MarshalText, got %q", text)
	}
}

func TestIdentityNeverShowsTheAddressThroughSlog(t *testing.T) {
	id := mustIdentity(t, identitySentinel)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("calling Jira", "mail", id, "holder", identityHolder{Mail: id})
	logger.Info("calling Jira again", slog.Any("mail", id))
	out := buf.String()
	if strings.Contains(out, identitySentinel) {
		t.Fatalf("address leaked through slog: %q", out)
	}
	if !strings.Contains(out, redactedText) {
		t.Fatalf("want the redacted text in the slog output, got %q", out)
	}
}

func TestIdentityRevealReturnsTheAddressOnly(t *testing.T) {
	id := mustIdentity(t, identitySentinel)
	if id.Reveal() != identitySentinel {
		t.Fatalf("Reveal = %q, want the stored address", id.Reveal())
	}
}

func TestNewIdentityRefusesEachBadValueWithoutEchoingIt(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "an empty value counts as missing", value: ""},
		{name: "no at sign", value: "ann.example.com"},
		{name: "a space", value: "ann @example.com"},
		{name: "a colon", value: "ann:x@example.com"},
		{name: "two at signs", value: "ann@@example.com"},
		{name: "no dot in the host", value: "ann@example"},
		{name: "a control character", value: "ann\x07@example.com"},
		{name: "longer than the address limit", value: strings.Repeat("a", MaxEmailLength) + "@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIdentity(identityVariable, tt.value)
			if err == nil {
				t.Fatalf("NewIdentity(%q) accepted the value", tt.value)
			}
			if id.Reveal() != "" {
				t.Fatalf("a refused identity holds a value")
			}
			if strings.Contains(err.Error(), tt.value) && tt.value != "" {
				t.Fatalf("refusal text echoes the value: %q", err.Error())
			}
			if tt.value == "" {
				var missing MissingKeyError
				if !errors.As(err, &missing) || missing.Variable != identityVariable {
					t.Fatalf("want a MissingKeyError naming %s, got %v", identityVariable, err)
				}
				if err.Error() != identityVariable+" is not set" {
					t.Fatalf("missing text = %q", err.Error())
				}
				return
			}
			var invalid InvalidIdentityError
			if !errors.As(err, &invalid) || invalid.Variable != identityVariable {
				t.Fatalf("want an InvalidIdentityError naming %s, got %v", identityVariable, err)
			}
			if err.Error() != identityVariable+" is not a valid e-mail address" {
				t.Fatalf("invalid text = %q", err.Error())
			}
		})
	}
}

func TestNewIdentityAcceptsASyntacticallyValidAddress(t *testing.T) {
	for _, ok := range []string{identitySentinel, "a.b+c@sub.example.co.id"} {
		if _, err := NewIdentity(identityVariable, ok); err != nil {
			t.Errorf("NewIdentity(%q) = %v, want nil", ok, err)
		}
	}
}

func TestZeroIdentityHoldsNoAddress(t *testing.T) {
	var zero Identity
	if zero.Reveal() != "" {
		t.Fatalf("zero Identity Reveal = %q, want empty", zero.Reveal())
	}
}
