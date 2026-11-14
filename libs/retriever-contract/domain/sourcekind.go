package domain

import (
	"regexp"
	"slices"
	"strings"
)

// SourceKind is a type of external system that Retriever can read, such as "outline". It carries its own name, display
// name, and identity trait, so a kind registered at start-up behaves like a built-in one. The zero value is no kind.
// Build a kind through a KindSet, or use one of the built-in constants.
type SourceKind struct {
	value        string
	display      string
	usesIdentity bool
}

// KindOutline is the Outline wiki kind.
var KindOutline = SourceKind{value: "outline", display: "Outline"}

// KindKiwi is the Kiwi TCMS kind, the test case management system of Example Org.
var KindKiwi = SourceKind{value: "kiwi", display: "Kiwi"}

// KindJira is the Jira Cloud kind, read only, of Example Org.
var KindJira = SourceKind{value: "jira", display: "Jira", usesIdentity: true}

// kindNamePattern is the slug shape of a kind name. A name is the first word of a command path and the area of an error
// code, so it follows the same rule as a source ID.
var kindNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// maxKindNameLength is the longest kind name in bytes. Every accepted byte is ASCII, so bytes are characters.
const maxKindNameLength = 32

// KindDefinition describes one kind that a composition root registers at start-up. Name is the slug that a catalog and a
// command path use. Display is the name a person reads, and it defaults to Name. UsesIdentity is true when a source of
// the kind needs an identity name, the non-secret variable that holds an account e-mail address.
type KindDefinition struct {
	Name         string
	Display      string
	UsesIdentity bool
}

// KindSet is the set of kinds one run accepts. Build it once at start-up with NewKindSet or BuiltinKinds. A set is a
// value: it is never changed after it is built, and a copy holds the same kinds.
type KindSet struct {
	// kinds holds the registered kinds sorted by name, so every listing is stable.
	kinds []SourceKind
}

// BuiltinKinds returns the set of the three built-in kinds: outline, kiwi, and jira.
func BuiltinKinds() KindSet {
	return newKindSet([]SourceKind{KindOutline, KindKiwi, KindJira})
}

// NewKindSet validates the definitions and returns the set that holds them. It refuses an empty list, a name outside the
// slug grammar, and a name that two definitions share. A definition with no display name shows its name.
func NewKindSet(defs ...KindDefinition) (KindSet, error) {
	if len(defs) == 0 {
		return KindSet{}, ErrNoKinds
	}
	kinds := make([]SourceKind, 0, len(defs))
	for _, def := range defs {
		if len(def.Name) > maxKindNameLength || !kindNamePattern.MatchString(def.Name) {
			return KindSet{}, ErrInvalidKindName
		}
		if slices.ContainsFunc(kinds, func(kind SourceKind) bool { return kind.value == def.Name }) {
			return KindSet{}, ErrDuplicateKind
		}
		display := def.Display
		if display == "" {
			display = def.Name
		}
		kinds = append(kinds, SourceKind{value: def.Name, display: display, usesIdentity: def.UsesIdentity})
	}
	return newKindSet(kinds), nil
}

// newKindSet sorts kinds by name and returns the set. The caller passes a slice it owns.
func newKindSet(kinds []SourceKind) KindSet {
	slices.SortFunc(kinds, func(a, b SourceKind) int { return strings.Compare(a.value, b.value) })
	return KindSet{kinds: kinds}
}

// Kind returns the registered kind named raw. Any other text returns an UnknownKindError, which names the registered
// kinds and never the text it was given.
func (s KindSet) Kind(raw string) (SourceKind, error) {
	for _, kind := range s.kinds {
		if kind.value == raw {
			return kind, nil
		}
	}
	return SourceKind{}, UnknownKindError{Known: s.Names()}
}

// Names returns the names of the registered kinds in alphabetical order. The caller may change the returned slice.
func (s KindSet) Names() []string {
	names := make([]string, 0, len(s.kinds))
	for _, kind := range s.kinds {
		names = append(names, kind.value)
	}
	return names
}

// NewSourceKind returns the built-in kind named raw. The flags and the GraphQL arguments take their kind here, so they
// accept the built-in kinds only. Any other text returns ErrUnknownSourceKind, whose text keeps the built-in order.
func NewSourceKind(raw string) (SourceKind, error) {
	kind, err := BuiltinKinds().Kind(raw)
	if err != nil {
		return SourceKind{}, ErrUnknownSourceKind
	}
	return kind, nil
}

// String returns the kind name, such as "outline".
func (k SourceKind) String() string {
	return k.value
}

// UsesIdentity reports whether a source of the kind needs an identity name, the non-secret variable that holds an account
// e-mail address. Jira needs one; Outline and Kiwi refuse one.
func (k SourceKind) UsesIdentity() bool {
	return k.usesIdentity
}

// Display returns the name of the kind as a person reads it in a failure, such as "Outline" or "Kiwi". A kind without a
// display name returns its kind name.
func (k SourceKind) Display() string {
	if k.display == "" {
		return k.value
	}
	return k.display
}

// UnknownKindError reports a kind name that a KindSet does not register. Known holds the registered names in alphabetical
// order. The error names no value that was given, because a catalog may hold a value pasted by mistake. It matches
// ErrUnknownSourceKind.
type UnknownKindError struct {
	Known []string
}

// Error names the registered kinds, such as "unknown source kind (known: jira, kiwi, outline)".
func (e UnknownKindError) Error() string {
	return "unknown source kind (known: " + strings.Join(e.Known, ", ") + ")"
}

// Is matches ErrUnknownSourceKind, so a caller that tests the sentinel still finds it.
func (e UnknownKindError) Is(target error) bool {
	return target == ErrUnknownSourceKind
}
