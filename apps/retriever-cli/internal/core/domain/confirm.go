package domain

import "errors"

// The confirmation sentinels of the critical tier (ADR 007 section 3.5). ErrYesRequired refuses a destructive or admin
// call that has no --yes. ErrConfirmRequired and ErrConfirmMismatch refuse a critical call whose --confirm is missing
// or does not equal the target that Retriever read.
var (
	// ErrYesRequired refuses a change that is destructive or admin and was not previewed or confirmed with --yes.
	ErrYesRequired = errors.New("this command changes the wiki: pass --dry-run to preview it or --yes to run it")
	// ErrConfirmRequired refuses a critical call that gives no --confirm value.
	ErrConfirmRequired = errors.New("this change is critical: give --confirm with the value that --dry-run names")
	// ErrConfirmMismatch refuses a critical call whose --confirm value is not the target that Retriever read.
	ErrConfirmMismatch = errors.New("--confirm does not match the target: give the value that --dry-run names")
)

// ErrSelfTarget refuses a change to the user of the API key itself: deleting, suspending, or changing the role of that user
// (ADR 007 section 3.5, rule 6). Retriever learns that user from auth.whoami, so a lockout by its own key is not reachable.
var ErrSelfTarget = errors.New("this is the user of the API key: Retriever does not change that user")

// ConfirmError is the refusal of a critical call's --confirm. Its text names the kind of value to give, such as the
// document ID, and never the value itself, so it is safe to show (ADR 007 section 3.5, item 5). It matches
// ErrConfirmRequired when Mismatch is false and ErrConfirmMismatch when it is true.
type ConfirmError struct {
	Kind     string
	Mismatch bool
}

// Error returns the public text for the refusal.
func (e ConfirmError) Error() string {
	if e.Mismatch {
		return "--confirm does not match the target: pass " + e.Kind + " exactly; --dry-run shows the target"
	}
	return "this change is critical: pass --confirm with " + e.Kind + "; --dry-run shows the target"
}

// Is makes the error match the sentinel of its kind of refusal.
func (e ConfirmError) Is(target error) bool {
	if e.Mismatch {
		return target == ErrConfirmMismatch
	}
	return target == ErrConfirmRequired
}
