package operations

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestSliceC2UsersInviteTakesOneEmailAddressAndRefusesAMalformedOne checks the e-mail kind: a well-formed address is bound, and a malformed one is refused with the fixed text, never the address.
func TestSliceC2UsersInviteTakesOneEmailAddressAndRefusesAMalformedOne(t *testing.T) {
	row := rowOf(t, domain.OpUsersInvite)
	args, err := Bind(row, map[string]any{"email": "ann@example.com", "name": "Ann", "role": "member"})
	if err != nil {
		t.Fatalf("Bind a valid invitation: %v", err)
	}
	if args.String("email") != "ann@example.com" {
		t.Fatalf("the bound email = %q", args.String("email"))
	}
	_, err = Bind(row, map[string]any{"email": "ann example.com", "name": "Ann", "role": "member"})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("Bind a malformed address = %v, want ErrInvalidEmail", err)
	}
}

// TestSliceC2GroupsDeleteAllTakesTheProviderIDAsItsOnlyArgument checks that the positional argument of the critical group
// deletion is the sign-in provider, which the canonical target names.
func TestSliceC2GroupsDeleteAllTakesTheProviderIDAsItsOnlyArgument(t *testing.T) {
	row := rowOf(t, domain.OpGroupsDeleteAll)
	if got := row.Positional(); got != "authenticationProviderId" {
		t.Fatalf("positional = %q, want authenticationProviderId", got)
	}
	if len(row.Params) != 1 {
		t.Fatalf("params = %d, want 1", len(row.Params))
	}
}
