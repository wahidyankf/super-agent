package domain

// Source is one configured instance of a kind. It is identified by its SourceID. Its fields are value objects,
// so the constructor has no cross-field rule of its own. Read it through the accessors.
type Source struct {
	id         SourceID
	kind       SourceKind
	tags       TagSet
	endpoint   Endpoint
	credential CredentialRef
	// identity names the variable that holds the source's account e-mail address. The zero value means none. It is set by WithIdentity, so NewSource keeps its signature.
	identity IdentityRef
}

// NewSource assembles a source from validated value objects.
func NewSource(id SourceID, kind SourceKind, tags TagSet, endpoint Endpoint, credential CredentialRef) Source {
	return Source{id: id, kind: kind, tags: tags, endpoint: endpoint, credential: credential}
}

// ID returns the source ID.
func (s Source) ID() SourceID { return s.id }

// Kind returns the source kind.
func (s Source) Kind() SourceKind { return s.kind }

// Tags returns the source tags.
func (s Source) Tags() TagSet { return s.tags }

// Endpoint returns the HTTPS origin of the source.
func (s Source) Endpoint() Endpoint { return s.endpoint }

// Credential returns the name of the variable that holds the source's API key.
func (s Source) Credential() CredentialRef { return s.credential }

// Identity returns the name of the variable that holds the source's account e-mail address. The zero value means the
// source has none.
func (s Source) Identity() IdentityRef { return s.identity }

// WithIdentity returns a copy of the source with the identity name ref. Every other field stays as the constructor set
// it, and the receiver does not change.
func (s Source) WithIdentity(ref IdentityRef) Source {
	s.identity = ref
	return s
}

// SourceSummary is what an inbound adapter may show about a source. It has no endpoint and no credential name.
type SourceSummary struct {
	ID   string
	Kind string
	Tags []string
}

// Summary returns the printable view of the source.
func (s Source) Summary() SourceSummary {
	return SourceSummary{
		ID:   s.id.String(),
		Kind: s.kind.String(),
		Tags: s.tags.Values(),
	}
}
