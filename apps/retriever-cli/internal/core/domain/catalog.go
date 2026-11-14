package domain

import (
	"fmt"
	"slices"
	"strings"
)

// maxCatalogSources is the most sources one catalog holds.
const maxCatalogSources = 64

// SourceFault reports one invalid source of a catalog, by its 1-based position in the file and the field that broke
// a rule. Its text carries no value, only the position, the field, and the rule.
type SourceFault struct {
	Position int
	Field    string
	Err      error
}

// Error names the source position and the field, then the rule.
func (e SourceFault) Error() string {
	return fmt.Sprintf("source %d %s: %v", e.Position, e.Field, e.Err)
}

// Unwrap returns the rule that broke, so errors.Is finds it.
func (e SourceFault) Unwrap() error {
	return e.Err
}

// CatalogError is a sources catalog that cannot be used. Its text is safe to print; it never repeats a value. Cause is the
// typed rule that broke, if any, so errors.Is and errors.As see through the catalog to its kind and its reason.
type CatalogError struct {
	Message string
	Cause   error
}

// Error returns the safe text.
func (e CatalogError) Error() string {
	return e.Message
}

// Unwrap returns the rule that broke, so the failure code can name its kind and its reason.
func (e CatalogError) Unwrap() error {
	return e.Cause
}

// SourceFilter selects sources. A nil Kind matches every kind. Every tag of Tags must be on a source.
type SourceFilter struct {
	Kind *SourceKind
	Tags TagSet
}

// NewSourceFilter parses the text arguments of a source query, which the CLI flags and the GraphQL arguments share.
// A nil kind matches every kind, and a given kind must be a known kind. Each tag is validated, and no tag is no filter.
func NewSourceFilter(kind *string, tags ...string) (SourceFilter, error) {
	var filter SourceFilter
	if kind != nil {
		parsed, err := NewSourceKind(*kind)
		if err != nil {
			return SourceFilter{}, err
		}
		filter.Kind = &parsed
	}
	set, err := NewTagSet(tags...)
	if err != nil {
		return SourceFilter{}, err
	}
	filter.Tags = set
	return filter, nil
}

// Catalog is the aggregate root of the sources: an immutable set of 1 to 64 sources with unique IDs. One
// credential name serves one endpoint host only. The zero value holds no source.
type Catalog struct {
	sources []Source
}

// NewCatalog checks the invariants and returns the catalog sorted by source ID. It returns no catalog on a failure.
func NewCatalog(sources []Source) (Catalog, error) {
	if len(sources) == 0 {
		return Catalog{}, ErrEmptyCatalog
	}
	if len(sources) > maxCatalogSources {
		return Catalog{}, ErrTooManySources
	}
	ids := map[string]bool{}
	hosts := map[string]string{}
	identities := map[string]string{}
	for i, src := range sources {
		position := i + 1
		if fault := zeroFault(position, src); fault != nil {
			return Catalog{}, fault
		}
		if fault := kindFault(position, src); fault != nil {
			return Catalog{}, fault
		}
		if ids[src.id.value] {
			return Catalog{}, SourceFault{Position: position, Field: "id", Err: ErrDuplicateSource}
		}
		ids[src.id.value] = true
		name, host := src.credential.name, src.endpoint.host
		if seen, ok := hosts[name]; ok && seen != host {
			return Catalog{}, SourceFault{Position: position, Field: "credential_env", Err: ErrCredentialHostMismatch}
		}
		hosts[name] = host
		if identity := src.identity.name; identity != "" {
			if seen, ok := identities[identity]; ok && seen != host {
				return Catalog{}, SourceFault{Position: position, Field: "identity_env", Err: ErrIdentityHostMismatch}
			}
			identities[identity] = host
		}
	}
	sorted := slices.Clone(sources)
	slices.SortFunc(sorted, func(a, b Source) int { return strings.Compare(a.id.value, b.id.value) })
	return Catalog{sources: sorted}, nil
}

// zeroFault returns a SourceFault for the first required field of src that is a zero value, or nil when none is. A kind
// that uses an identity name (jira) needs a non-zero identity, so a zero identity there is a fault too. Only a validating constructor fills these fields, so a zero value means the source was not built by one.
func zeroFault(position int, src Source) error {
	switch {
	case src.id == SourceID{}:
		return SourceFault{Position: position, Field: "id", Err: ErrInvalidSourceID}
	case src.kind == SourceKind{}:
		return SourceFault{Position: position, Field: "kind", Err: ErrUnknownSourceKind}
	case src.endpoint == Endpoint{}:
		return SourceFault{Position: position, Field: "endpoint", Err: ErrInvalidEndpoint}
	case src.credential == CredentialRef{}:
		return SourceFault{Position: position, Field: "credential_env", Err: ErrInvalidCredentialRef}
	case src.kind.UsesIdentity() && src.identity == IdentityRef{}:
		return SourceFault{Position: position, Field: "identity_env", Err: IdentityRequiredError{Kind: src.kind}}
	}
	return nil
}

// All returns every source, sorted by ID. The caller may change the returned slice.
func (c Catalog) All() []Source {
	return slices.Clone(c.sources)
}

// ByID returns the source with the given ID, and whether one exists.
func (c Catalog) ByID(id SourceID) (Source, bool) {
	for _, src := range c.sources {
		if src.id == id {
			return src, true
		}
	}
	return Source{}, false
}

// Find returns the sources that match the filter, sorted by ID.
func (c Catalog) Find(filter SourceFilter) []Source {
	var out []Source
	for _, src := range c.sources {
		if filter.Kind != nil && src.kind != *filter.Kind {
			continue
		}
		if !src.tags.ContainsAll(filter.Tags) {
			continue
		}
		out = append(out, src)
	}
	return out
}

// Select is the default rule. With an ID, the source must exist, and it must have the kind. Without one, exactly one
// source of the kind must exist.
func (c Catalog) Select(kind SourceKind, id *SourceID) (Source, error) {
	if id != nil {
		src, ok := c.ByID(*id)
		if !ok {
			return Source{}, UnknownSourceError{ID: id.String()}
		}
		if src.kind != kind {
			return Source{}, ErrWrongSourceKind
		}
		return src, nil
	}
	matches := c.Find(SourceFilter{Kind: &kind})
	switch len(matches) {
	case 0:
		return Source{}, NoSourceOfKindError{Kind: kind}
	case 1:
		return matches[0], nil
	}
	ids := make([]string, 0, len(matches))
	for _, src := range matches {
		ids = append(ids, src.id.String())
	}
	return Source{}, AmbiguousSourceError{Kind: kind, IDs: ids}
}

// NoSourceOfKindError reports that no source of the kind is configured. Its text names the kind, such as "no kiwi source
// is configured", and for the Outline kind it is "no outline source is configured". It matches ErrNoSourceOfKind.
type NoSourceOfKindError struct {
	Kind SourceKind
}

// Error names the kind that has no source.
func (e NoSourceOfKindError) Error() string {
	return "no " + e.Kind.String() + " source is configured"
}

// Is matches ErrNoSourceOfKind, so a caller that tests the sentinel still finds it.
func (e NoSourceOfKindError) Is(target error) bool {
	return target == ErrNoSourceOfKind
}

// UnknownSourceError reports a source ID that the catalog does not hold. The ID is a validated slug.
type UnknownSourceError struct {
	ID string
}

// Error names the unknown source.
func (e UnknownSourceError) Error() string {
	return fmt.Sprintf("unknown source %q", e.ID)
}

// Is matches ErrUnknownSource.
func (e UnknownSourceError) Is(target error) bool {
	return target == ErrUnknownSource
}

// AmbiguousSourceError reports that more than one source of a kind exists and none was named. IDs are sorted.
type AmbiguousSourceError struct {
	Kind SourceKind
	IDs  []string
}

// Error lists the sources that match.
func (e AmbiguousSourceError) Error() string {
	return fmt.Sprintf("more than one %s source exists: %s", e.Kind, strings.Join(e.IDs, ", "))
}
