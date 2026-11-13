package domain

import "strings"

// IdentityRequiredError reports a source of a kind that needs an identity name and has none (ADR 010 section 1.1). Its text
// names the kind, never a value, and it matches ErrIdentityRequired.
type IdentityRequiredError struct {
	Kind SourceKind
}

// Error returns the rule text, such as "required for a jira source".
func (e IdentityRequiredError) Error() string {
	return "required for " + kindPhrase(e.Kind)
}

// Is matches ErrIdentityRequired.
func (e IdentityRequiredError) Is(target error) bool {
	return target == ErrIdentityRequired
}

// IdentityNotUsedError reports a source of a kind that does not use an identity name but has one (ADR 010 section 1.1).
// Its text names the kind, and it matches ErrIdentityNotUsed.
type IdentityNotUsedError struct {
	Kind SourceKind
}

// Error returns the rule text, such as "not used by an outline source".
func (e IdentityNotUsedError) Error() string {
	return "not used by " + kindPhrase(e.Kind)
}

// Is matches ErrIdentityNotUsed.
func (e IdentityNotUsedError) Is(target error) bool {
	return target == ErrIdentityNotUsed
}

// KindCeilingError reports an access level above the highest level its kind accepts (ADR 010 section 1.1). Its text names
// the kind and the highest level, such as "a jira source allows read at most", and it matches ErrAccessAboveKindCeiling.
type KindCeilingError struct {
	Kind SourceKind
	Max  Access
}

// Error returns the rule text, such as "a jira source allows read at most".
func (e KindCeilingError) Error() string {
	return kindPhrase(e.Kind) + " allows " + e.Max.String() + " at most"
}

// Is matches ErrAccessAboveKindCeiling.
func (e KindCeilingError) Is(target error) bool {
	return target == ErrAccessAboveKindCeiling
}

// kindPhrase returns the kind as a noun phrase with its article, such as "an outline source" or "a jira source". The
// article follows the sound of the first letter: a vowel takes "an".
func kindPhrase(kind SourceKind) string {
	article := "a"
	if strings.ContainsRune("aeiou", rune(kind.value[0])) {
		article = "an"
	}
	return article + " " + kind.value + " source"
}

// kindFault returns a SourceFault for the first rule of its kind that src breaks, beyond the zero-value checks of
// zeroFault: an identity name on a kind that does not use one, and an access level above the kind's highest level. It
// returns nil when none is broken. The caller has already passed zeroFault, so the kind is known.
func kindFault(position int, src Source) error {
	if !src.kind.UsesIdentity() && src.identity != (IdentityRef{}) {
		return SourceFault{Position: position, Field: "identity_env", Err: IdentityNotUsedError{Kind: src.kind}}
	}
	if ceiling := src.kind.MaxAccess(); !ceiling.Allows(src.access) {
		return SourceFault{Position: position, Field: "access", Err: KindCeilingError{Kind: src.kind, Max: ceiling}}
	}
	return nil
}
