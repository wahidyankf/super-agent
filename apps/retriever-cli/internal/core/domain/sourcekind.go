package domain

import "strings"

// SourceKind is a type of external system that Retriever can read. It is a closed set in code.
type SourceKind struct {
	value string
}

// KindOutline is the Outline wiki kind.
var KindOutline = SourceKind{value: "outline"}

// KindKiwi is the Kiwi TCMS kind, the test case management system of Example Org (ADR 009).
var KindKiwi = SourceKind{value: "kiwi"}

// KindJira is the Jira Cloud kind, read only, of Example Org (ADR 010).
var KindJira = SourceKind{value: "jira"}

// knownKinds lists every kind NewSourceKind accepts, in the order the unknown-kind text names them.
var knownKinds = []SourceKind{KindOutline, KindKiwi, KindJira}

// kindTraits is what the catalog needs to know about one kind beside its name (ADR 010 section 1.1).
type kindTraits struct {
	// usesIdentity is true when a source of the kind needs an identity name (a non-secret variable beside the key).
	usesIdentity bool
	// maxAccess is the highest access level the catalog accepts for a source of the kind.
	maxAccess Access
}

// traits holds the traits of every known kind. A kind missing from the table uses no identity and has no ceiling below
// admin, the highest level Retriever knows. Production kinds always come from knownKinds, and zeroFault refuses a zero kind
// before any rule reads it.
var traits = map[SourceKind]kindTraits{
	KindOutline: {usesIdentity: false, maxAccess: AccessAdmin},
	KindKiwi:    {usesIdentity: false, maxAccess: AccessAdmin},
	KindJira:    {usesIdentity: true, maxAccess: AccessRead},
}

// UsesIdentity reports whether a source of the kind needs an identity name, the non-secret variable that holds an account
// e-mail address (ADR 010 section 1.1). Jira needs one; Outline and Kiwi refuse one.
func (k SourceKind) UsesIdentity() bool {
	return traits[k].usesIdentity
}

// MaxAccess returns the highest access level the catalog accepts for a source of the kind. A kind with no traits returns
// admin, so it gets no ceiling below the highest level.
func (k SourceKind) MaxAccess() Access {
	t, ok := traits[k]
	if !ok {
		return AccessAdmin
	}
	return t.maxAccess
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

// Display returns the name of the kind as a person reads it in a failure, such as "Outline" or "Kiwi" (ADR 010 section
// 7). A kind without a display name returns its kind name.
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
