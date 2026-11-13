// Package catalog loads the sources catalog, a YAML file the user names with --catalog, RETRIEVER_CATALOG, or a
// sources.yml in the working directory or the user configuration folder (see the configpaths package). The file is the
// whole catalog. A missing file is an error: the tool never invents a source.
//
// The YAML types never leave this package. Every error text names the rule that broke and never a value, because a
// catalog may hold a key pasted by mistake.
package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// maxBytes caps the catalog file size.
const maxBytes = 64 << 10

// schemaVersion is the only catalog version this code reads.
const schemaVersion = 1

// Loader implements ports.CatalogLoader over one catalog file.
type Loader struct {
	path string
}

var _ ports.CatalogLoader = (*Loader)(nil)

// New returns a loader for the catalog file at path. path must be absolute.
func New(path string) (*Loader, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("the catalog path must be absolute")
	}
	return &Loader{path: path}, nil
}

// Load reads the catalog file once. A missing file, a file that is not a regular file, a file that is too large, and a
// file that is invalid each return a domain.CatalogError.
func (l *Loader) Load() (domain.Catalog, error) {
	data, found, err := l.read()
	if err != nil {
		return domain.Catalog{}, err
	}
	if !found {
		return domain.Catalog{}, catalogCause("not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml", domain.ErrCatalogMissing)
	}
	return parse(data)
}

// read returns the file bytes. found is false when no file exists at the path, with no error.
func (l *Loader) read() (data []byte, found bool, err error) {
	info, err := os.Lstat(l.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, catalogErr("could not check the file")
	}
	if !info.Mode().IsRegular() {
		return nil, false, catalogErr("must be a regular file, not a symlink or a folder")
	}
	f, err := os.Open(l.path)
	if err != nil {
		return nil, false, catalogErr("could not open the file")
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()
	opened, err := f.Stat()
	if err != nil {
		return nil, false, catalogErr("could not check the file")
	}
	if !os.SameFile(info, opened) {
		return nil, false, catalogErr("changed while it was being opened")
	}
	data, err = io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, false, catalogErr("could not read the file")
	}
	if len(data) > maxBytes {
		return nil, false, catalogErr("must be at most 64 KiB")
	}
	return data, true, nil
}

// fileDoc is the version 1 shape of the catalog file. Its fields are decoded and then checked by the domain.
type fileDoc struct {
	Version *int          `yaml:"version"`
	Sources []sourceEntry `yaml:"sources"`
}

// sourceEntry is one source as written. A nil Access means the field was absent or null, so the level is read.
type sourceEntry struct {
	ID            string   `yaml:"id"`
	Kind          string   `yaml:"kind"`
	Tags          []string `yaml:"tags"`
	Access        *string  `yaml:"access"`
	Endpoint      string   `yaml:"endpoint"`
	CredentialEnv string   `yaml:"credential_env"`
	// IdentityEnv is the optional name of the variable that holds the source's e-mail address (ADR 010 section 1.1). A nil
	// pointer means the field is absent, which an empty value is not.
	IdentityEnv *string `yaml:"identity_env"`
}

// parse decodes data strictly and maps it onto a domain.Catalog.
func parse(data []byte) (domain.Catalog, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var doc fileDoc
	if err := dec.Decode(&doc); err != nil {
		return domain.Catalog{}, catalogErr("the file is not valid YAML, or it has an unknown or repeated field")
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.Catalog{}, catalogErr("the file must hold one YAML document")
	}
	if doc.Version == nil || *doc.Version != schemaVersion {
		return domain.Catalog{}, catalogErr("version: must be 1")
	}
	list := make([]domain.Source, 0, len(doc.Sources))
	for i, entry := range doc.Sources {
		src, err := toSource(i+1, entry)
		if err != nil {
			return domain.Catalog{}, err
		}
		list = append(list, src)
	}
	cat, err := domain.NewCatalog(list)
	if err != nil {
		return domain.Catalog{}, catalogFault(err)
	}
	return cat, nil
}

// toSource maps one entry onto the value objects. position is the 1-based source number used in the error text.
func toSource(position int, e sourceEntry) (domain.Source, error) {
	id, err := domain.NewSourceID(e.ID)
	if err != nil {
		return domain.Source{}, fieldErr(position, "id", err)
	}
	kind, err := domain.NewSourceKind(e.Kind)
	if err != nil {
		return domain.Source{}, fieldErr(position, "kind", err)
	}
	tags, err := domain.NewTagSet(e.Tags...)
	if err != nil {
		return domain.Source{}, fieldErr(position, "tags", err)
	}
	access := domain.AccessRead
	if e.Access != nil {
		access, err = domain.NewAccess(*e.Access)
		if err != nil {
			return domain.Source{}, fieldErr(position, "access", err)
		}
	}
	endpoint, err := domain.NewEndpoint(e.Endpoint)
	if err != nil {
		if errors.Is(err, domain.ErrHTTPSRequired) {
			// A plain HTTP endpoint is refused at load, before any request, under the kind's own code.
			err = domain.KindError{Kind: kind, Err: err}
		}
		return domain.Source{}, fieldErr(position, "endpoint", err)
	}
	credential, err := domain.NewCredentialRef(e.CredentialEnv)
	if err != nil {
		return domain.Source{}, fieldErr(position, "credential_env", err)
	}
	src := domain.NewSource(id, kind, tags, access, endpoint, credential)
	// A nil pointer means the field is absent, so the source has no identity. An empty value is a name that fails the
	// rule (ADR 010 section 1.1).
	if e.IdentityEnv != nil {
		identity, err := domain.NewIdentityRef(*e.IdentityEnv)
		if err != nil {
			return domain.Source{}, fieldErr(position, "identity_env", err)
		}
		src = src.WithIdentity(identity)
	}
	return src, nil
}

// catalogErr returns the safe catalog error with a rule that names no value. The text never holds the file path, which
// may name a personal folder.
func catalogErr(rule string) error {
	return domain.CatalogError{Message: "sources catalog: " + rule}
}

// catalogCause returns the safe catalog error with the typed rule that broke as its cause, so the failure code can name
// the kind and the reason. The rule text stays the same as catalogErr's.
func catalogCause(rule string, cause error) error {
	return domain.CatalogError{Message: "sources catalog: " + rule, Cause: cause}
}

// fieldErr reports one source field that broke its rule.
func fieldErr(position int, field string, err error) error {
	return catalogCause(fmt.Sprintf("source %d %s: %s", position, field, fieldRule(err)), err)
}

// catalogFault maps a domain invariant failure onto the safe catalog error.
func catalogFault(err error) error {
	var fault domain.SourceFault
	switch {
	case errors.As(err, &fault):
		return catalogCause(fmt.Sprintf("source %d %s: %s", fault.Position, fault.Field, invariantRule(fault.Err)), fault)
	case errors.Is(err, domain.ErrEmptyCatalog):
		return catalogErr("sources: " + domain.ErrEmptyCatalog.Error())
	case errors.Is(err, domain.ErrTooManySources):
		return catalogErr("sources: " + domain.ErrTooManySources.Error())
	}
	return catalogErr("sources: invalid")
}

// fieldRule returns the rule text for a field that broke its own grammar. Each rule states the requirement and
// never a value, so the text is the same for every value a field could hold.
func fieldRule(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidSourceID):
		return "must be a lowercase slug"
	case errors.Is(err, domain.ErrInvalidTag):
		return "must be lowercase slugs"
	case errors.Is(err, domain.ErrTooManyTags):
		return "at most 16 distinct tags"
	case errors.Is(err, domain.ErrInvalidAccess):
		return domain.ErrInvalidAccess.Error()
	case errors.Is(err, domain.ErrUnknownSourceKind):
		return "unknown kind (known: outline, kiwi, jira)"
	case errors.Is(err, domain.ErrInvalidEndpoint), errors.Is(err, domain.ErrHTTPSRequired):
		return "must be an https host name"
	case errors.Is(err, domain.ErrInvalidCredentialRef):
		return "must be an upper-case *_API_KEY name"
	case errors.Is(err, domain.ErrInvalidIdentityRef):
		return "must be an upper-case *_EMAIL name"
	}
	return "invalid"
}

// invariantRule returns the rule text for a catalog invariant that broke.
func invariantRule(err error) string {
	switch {
	case errors.Is(err, domain.ErrDuplicateSource):
		return "duplicate"
	case errors.Is(err, domain.ErrCredentialHostMismatch):
		return "one credential serves one host only"
	case errors.Is(err, domain.ErrIdentityHostMismatch):
		return domain.ErrIdentityHostMismatch.Error()
	case errors.Is(err, domain.ErrIdentityRequired), errors.Is(err, domain.ErrIdentityNotUsed),
		errors.Is(err, domain.ErrAccessAboveKindCeiling):
		// These texts name the kind and the rule, never a value, so the typed error's own text is the rule.
		return err.Error()
	}
	return "invalid"
}
