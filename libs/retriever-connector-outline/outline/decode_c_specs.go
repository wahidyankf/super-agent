package outline

import (
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// teamSpec projects the team of the key's user, with the fields of the Team row of the field reference. The preferences,
// the avatar, the plan, the entitlements, the default collection, and the allowed domains are never read, so they never
// reach a caller.
func teamSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("description", textOrNil(m, "description")),
		field("sharing", boolOf(m, "sharing")),
		field("documentEmbeds", boolOf(m, "documentEmbeds")),
		field("memberCollectionCreate", boolOf(m, "memberCollectionCreate")),
		field("memberTeamCreate", boolOf(m, "memberTeamCreate")),
		field("defaultUserRole", textOf(m, "defaultUserRole")),
		field("inviteRequired", boolOf(m, "inviteRequired")),
		field("guestSignin", boolOf(m, "guestSignin")),
		field("passkeysEnabled", boolOf(m, "passkeysEnabled")),
		field("subdomain", textOrNil(m, "subdomain")),
		field("domain", textOrNil(m, "domain")),
		field("url", textOf(m, "url")),
	), nil
}

// projectStarted projects authenticationProviders.startGroupSync. The reply holds data.url, the redirect address of the
// identity provider, which is never shown. The result is {started: true} once the reply names that address; a reply without
// it is an unexpected reply.
func projectStarted(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	if textOf(m, "url") == "" {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return domain.NewRecord(field("started", true)), nil
}
