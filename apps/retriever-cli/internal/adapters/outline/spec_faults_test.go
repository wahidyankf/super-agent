package outline

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// idObjectSpecs are the object projections that name their object by an ID. Each one must refuse an object with no ID as
// an unexpected reply, and that refusal is the error branch the projection takes before it reads any other field.
var idObjectSpecs = map[string]objectSpec{
	"membership":           membershipSpec,
	"group membership":     groupMembershipSpec,
	"user":                 userSpec,
	"user notification":    userNotificationSpec,
	"group":                groupSpec,
	"comment":              commentSpec,
	"revision":             revisionSpec,
	"revision row":         revisionRowSpec,
	"share":                shareSpec,
	"template":             templateSpec,
	"pin":                  pinSpec,
	"star":                 starSpec,
	"subscription":         subscriptionSpec,
	"team":                 teamSpec,
	"api key":              apiKeySpec,
	"auth provider":        authProviderSpec,
	"import":               importSpec,
	"integration":          integrationSpec,
	"oauth client":         oauthClientSpec,
	"webhook":              webhookSpec,
	"emoji":                emojiSpec,
	"attachment":           attachmentSpec,
	"event":                eventSpec,
	"file operation":       fileOperationSpec,
	"access request":       accessRequestSpec,
	"notification":         notificationSpec,
	"oauth authentication": oauthAuthenticationSpec,
	"reaction":             reactionSpec,
	"relationship":         relationshipSpec,
	"search":               searchSpec,
}

// TestAnObjectWithNoIDIsAnUnexpectedReply checks that every projection that names its object by ID refuses an object with
// no ID, and that the refusal is an unexpected reply.
func TestAnObjectWithNoIDIsAnUnexpectedReply(t *testing.T) {
	for name, spec := range idObjectSpecs {
		t.Run(name, func(t *testing.T) {
			_, err := spec(map[string]any{"name": "named but with no ID"})
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("spec error = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}
