package graphql

import (
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	usecases "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// Resolver is the dependency-injection root of the GraphQL server. Its resolvers call the core use
// cases with these collaborators and hold no logic of their own. The field is Service, not Sources,
// because the generated query resolver has a method named Sources that would hide a field of that name.
type Resolver struct {
	Service usecases.Service
}

// kindArg returns the source kind a query names. An omitted kind is outline, so every call that predates the argument
// keeps its meaning. A kind the code does not know is ErrUnknownSourceKind, before any request.
func kindArg(kind *string) (domain.SourceKind, error) {
	if kind == nil {
		return domain.KindOutline, nil
	}
	return domain.NewSourceKind(*kind)
}
