package domain

import "regexp"

// eventNamePattern matches one webhook event name: `*` for every event, or a name with a dot between the resource and its
// action, such as documents.create or authenticationProviders.update. The list of names is the wiki's and is not closed
// here, so the pattern only refuses text that cannot be a name.
var eventNamePattern = regexp.MustCompile(`^(\*|[A-Za-z]+\.[A-Za-z]+)$`)

// ValidateEventList accepts at most MaxIDList event names, and each one must match eventNamePattern. A refused name is
// ErrInvalidName, the refusal of a name that is not text of the right shape.
func ValidateEventList(names []string) error {
	if len(names) > MaxIDList {
		return ErrInvalidName
	}
	for _, name := range names {
		if !eventNamePattern.MatchString(name) {
			return ErrInvalidName
		}
	}
	return nil
}
