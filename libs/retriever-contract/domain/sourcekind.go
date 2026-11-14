package domain

import "strings"

// SourceKind is a type of external system that Retriever can read. It is a closed set in code.
type SourceKind struct {
	value string
}

// KindOutline is the Outline wiki kind.
var KindOutline = SourceKind{value: "outline"}

// KindKiwi is the Kiwi TCMS kind, the test case management system of Example Org.
var KindKiwi = SourceKind{value: "kiwi"}

// KindJira is the Jira Cloud kind, read only, of Example Org.
var KindJira = SourceKind{value: "jira"}

// knownKinds lists every kind NewSourceKind accepts, in the order the unknown-kind text names them.
var knownKinds = []SourceKind{KindOutline, KindKiwi, KindJira}

// kindTraits is what the catalog needs to know about one kind beside its name.
type kindTraits struct {
	// usesIdentity is true when a source of the kind needs an identity name (a non-secret variable beside the key).
	usesIdentity bool
}

// traits holds the traits of every known kind. A kind missing from the table uses no identity. Production kinds always
// come from knownKinds, and zeroFault refuses a zero kind before any rule reads it.
var traits = map[SourceKind]kindTraits{
	KindOutline: {usesIdentity: false},
	KindKiwi:    {usesIdentity: false},
	KindJira:    {usesIdentity: true},
}

// UsesIdentity reports whether a source of the kind needs an identity name, the non-secret variable that holds an account
// e-mail address. Jira needs one; Outline and Kiwi refuse one.
func (k SourceKind) UsesIdentity() bool {
	return traits[k].usesIdentity
}

// knownKindNames returns the names of knownKinds joined with ", ", such as "outline, kiwi, jira". The unknown-kind text
// is built from it, so the text always names every known kind.
func knownKindNames() string {
	names := make([]string, 0, len(knownKinds))
	for _, kind := range knownKinds {
		names = append(names, kind.value)
	}
	return strings.Join(names, ", ")
}

// NewSourceKind returns the kind named raw. Any other text returns ErrUnknownSourceKind.
func NewSourceKind(raw string) (SourceKind, error) {
	for _, kind := range knownKinds {
		if kind.value == raw {
			return kind, nil
		}
	}
	return SourceKind{}, ErrUnknownSourceKind
}

// String returns the kind name, such as "outline".
func (k SourceKind) String() string {
	return k.value
}

// Display returns the name of the kind as a person reads it in a failure, such as "Outline" or "Kiwi". A kind without a display name returns its kind name.
func (k SourceKind) Display() string {
	switch k.value {
	case "outline":
		return "Outline"
	case "kiwi":
		return "Kiwi"
	case "jira":
		return "Jira"
	}
	return k.value
}
