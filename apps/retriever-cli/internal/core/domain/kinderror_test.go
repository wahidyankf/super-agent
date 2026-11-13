package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestSourceKindDisplayNamesEachKind checks the name a person reads in a failure, per ADR 010 section 7.
func TestSourceKindDisplayNamesEachKind(t *testing.T) {
	cases := map[string]SourceKind{
		"Outline": KindOutline,
		"Kiwi":    KindKiwi,
		"Jira":    KindJira,
	}
	for want, kind := range cases {
		if got := kind.Display(); got != want {
			t.Errorf("%s.Display() = %q, want %q", kind, got, want)
		}
	}
}

// TestKindErrorKeepsTheSentinelReachable checks that errors.Is still finds the wrapped sentinel, so a caller that tests
// for ErrKeyRejected does not change when a kind is added.
func TestKindErrorKeepsTheSentinelReachable(t *testing.T) {
	err := KindError{Kind: KindKiwi, Err: ErrKeyRejected}
	if !errors.Is(err, ErrKeyRejected) {
		t.Fatalf("errors.Is(KindError, ErrKeyRejected) = false, want true")
	}
	if errors.Unwrap(err) != ErrKeyRejected {
		t.Fatalf("Unwrap = %v, want ErrKeyRejected", errors.Unwrap(err))
	}
	if errors.Is(err, ErrAccessDenied) {
		t.Fatalf("errors.Is(KindError, ErrAccessDenied) = true, want false")
	}
}

// TestKindErrorIsMatchesTheSameKindOnly checks the Is method: a KindError matches another of the same kind over the
// same sentinel, and never one of another kind.
func TestKindErrorIsMatchesTheSameKindOnly(t *testing.T) {
	kiwi := KindError{Kind: KindKiwi, Err: ErrKeyRejected}
	if !errors.Is(kiwi, KindError{Kind: KindKiwi, Err: ErrKeyRejected}) {
		t.Fatalf("a Kiwi KindError does not match an equal Kiwi KindError")
	}
	if errors.Is(kiwi, KindError{Kind: KindOutline, Err: ErrKeyRejected}) {
		t.Fatalf("a Kiwi KindError matches an Outline KindError")
	}
}

// TestKindErrorTextNamesTheKind checks the text: an Outline error is unchanged, and any other kind names itself.
func TestKindErrorTextNamesTheKind(t *testing.T) {
	if got, want := (KindError{Kind: KindOutline, Err: ErrKeyRejected}).Error(), ErrKeyRejected.Error(); got != want {
		t.Fatalf("Outline KindError text = %q, want the sentinel text %q", got, want)
	}
	if got, want := (KindError{Kind: KindKiwi, Err: ErrKeyRejected}).Error(), "Kiwi rejected the API key"; got != want {
		t.Fatalf("Kiwi KindError text = %q, want %q", got, want)
	}
}

// TestEveryPublicSentinelIsNamedForItsKind is a table over PublicErrors. A Kiwi-wrapped sentinel never says "Outline",
// an Outline-wrapped one keeps its fixed text byte for byte, and every public sentinel is public through PublicText.
func TestEveryPublicSentinelIsNamedForItsKind(t *testing.T) {
	for _, sentinel := range PublicErrors() {
		t.Run(sentinel.Error(), func(t *testing.T) {
			kiwi := KindError{Kind: KindKiwi, Err: sentinel}
			if strings.Contains(kiwi.Error(), "Outline") {
				t.Fatalf("Kiwi KindError text %q names Outline", kiwi.Error())
			}
			outline := KindError{Kind: KindOutline, Err: sentinel}
			if outline.Error() != sentinel.Error() {
				t.Fatalf("Outline KindError text %q, want the sentinel text %q", outline.Error(), sentinel.Error())
			}
			text, ok := kiwi.PublicText()
			if !ok || text != kiwi.Error() {
				t.Fatalf("PublicText = (%q, %v), want (%q, true)", text, ok, kiwi.Error())
			}
		})
	}
}

// TestPublicTextHidesRawDetail checks that a KindError over a raw error, which is not a public sentinel, is not shown.
func TestPublicTextHidesRawDetail(t *testing.T) {
	raw := KindError{Kind: KindKiwi, Err: errors.New("upstream said Outline secret-detail")}
	if text, ok := raw.PublicText(); ok {
		t.Fatalf("PublicText = %q, want no public text for a raw error", text)
	}
}
