package domain

// Access is the most Retriever may do with one source. The order is read < write < admin. The zero value is
// invalid. Each operation checks the level it needs with Allows before it calls the source.
type Access struct {
	value string
	rank  int
}

// The three access levels, in order.
var (
	AccessRead  = Access{value: "read", rank: 1}
	AccessWrite = Access{value: "write", rank: 2}
	AccessAdmin = Access{value: "admin", rank: 3}
)

// NewAccess accepts exactly read, write, or admin. It does not trim or fold case, and any other text,
// including the empty string, returns ErrInvalidAccess.
func NewAccess(raw string) (Access, error) {
	switch raw {
	case AccessRead.value:
		return AccessRead, nil
	case AccessWrite.value:
		return AccessWrite, nil
	case AccessAdmin.value:
		return AccessAdmin, nil
	}
	return Access{}, ErrInvalidAccess
}

// Allows reports whether a is at least need. A zero value on either side allows nothing.
func (a Access) Allows(need Access) bool {
	return a.rank != 0 && need.rank != 0 && a.rank >= need.rank
}

// String returns the level name, such as "write". The zero value returns "".
func (a Access) String() string {
	return a.value
}

// AccessLevelError reports that a source's level is below the level an operation needs. Its text
// names the source ID and the two levels, which are closed, validated values, so the text is safe to show. It matches
// ErrAccessLevelTooLow. Nothing upstream was called.
type AccessLevelError struct {
	Source string
	Have   Access
	Need   Access
}

// Error returns the public text, such as: source "outline" allows read only; this operation needs write.
func (e AccessLevelError) Error() string {
	return `source "` + e.Source + `" allows ` + e.Have.String() + ` only; this operation needs ` + e.Need.String()
}

// Is makes the error match ErrAccessLevelTooLow.
func (e AccessLevelError) Is(target error) bool {
	return target == ErrAccessLevelTooLow
}
