package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestSliceC2CriticalRowsHaveNoToolAndNameTheirCanonicalTarget checks the three critical rows of slice c2 against the table of
// canonical targets (ADR 007 section 3.5): the user ID, the sign-in provider ID, and the team name read with auth.whoami.
// Each is critical on every call, needs --yes, and has no MCP tool at any ceiling.
func TestSliceC2CriticalRowsHaveNoToolAndNameTheirCanonicalTarget(t *testing.T) {
	cases := []struct {
		op    domain.OperationID
		kind  string
		read  domain.OperationID
		field string
		name  string
	}{
		{domain.OpUsersDelete, "the user ID", domain.OpUsersGet, "id", "name"},
		{domain.OpGroupsDeleteAll, "the authentication provider ID", domain.OpAuthenticationProvidersGet, "id", "name"},
		{domain.OpTeamsDelete, "the team name", domain.OpAuthWhoami, "team.name", ""},
	}
	for _, tc := range cases {
		row := rowOf(t, tc.op)
		if row.Critical == nil || row.Critical.When != "" || row.Critical.Kind != tc.kind {
			t.Errorf("%s: Critical = %+v, want critical on every call with kind %q", tc.op, row.Critical, tc.kind)
		}
		if row.Tool != "" {
			t.Errorf("%s: critical, but it has the MCP tool %q", tc.op, row.Tool)
		}
		if row.Target == nil || row.Target.Read != tc.read || row.Target.Field != tc.field || row.Target.Name != tc.name {
			t.Errorf("%s: Target = %+v, want read %s, field %q, name %q", tc.op, row.Target, tc.read, tc.field, tc.name)
		}
		if !row.NeedsYes() {
			t.Errorf("%s: NeedsYes false, want true", tc.op)
		}
	}
}

// TestSliceC2SelfRefusedRowsAreTheThreeUserChangesOfTheKeysOwnUser checks which rows the use case refuses for the user of the
// API key (ADR 007 section 3.5, rule 6). Each names that user by its userId parameter, which is required.
func TestSliceC2SelfRefusedRowsAreTheThreeUserChangesOfTheKeysOwnUser(t *testing.T) {
	var got []domain.OperationID
	for _, r := range Rows() {
		if r.RefuseSelf {
			got = append(got, r.ID)
		}
	}
	want := []domain.OperationID{domain.OpUsersDelete, domain.OpUsersSuspend, domain.OpUsersUpdateRole}
	if !slices.Equal(got, want) {
		t.Fatalf("self-refused rows = %v, want %v", got, want)
	}
	for _, id := range want {
		p, ok := rowOf(t, id).Param("userId")
		if !ok || !p.Required || p.Kind != KindUUID {
			t.Errorf("%s: the userId parameter = %+v, want a required UUID", id, p)
		}
	}
}

// TestSliceC2ToolsAndCriticalRowsCount checks the counts of slice c2: 18 rows, 15 with an MCP tool, and 3 critical rows with
// none. Together with the stage c1 rows, they make the 134 tools of the admin ceiling (ADR 007 section 3.3).
func TestSliceC2ToolsAndCriticalRowsCount(t *testing.T) {
	if len(adminRowsC2) != 18 {
		t.Fatalf("slice c2 holds %d rows, want 18", len(adminRowsC2))
	}
	tools, critical := 0, 0
	for _, r := range adminRowsC2 {
		if r.Tool != "" {
			tools++
		}
		if r.Critical != nil {
			critical++
		}
	}
	if tools != 15 || critical != 3 {
		t.Fatalf("slice c2 has %d tools and %d critical rows, want 15 and 3", tools, critical)
	}
}

// TestSliceC2UsersInviteTakesOneEmailAddressAndRefusesAMalformedOne checks the e-mail kind (ADR 007 section 3.5 and the
// redaction rules): a well-formed address is bound, and a malformed one is refused with the fixed text, never the address.
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
