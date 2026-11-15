package engine

import (
	"regexp"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The prefixes of the collection and document IDs that the engine builds for an issue key.
const (
	projectPrefix = "project-"
	issuePrefix   = "issue-"
)

// issueKeyPattern is the shape of an issue key that the built IDs share: an upper-case project key of 2 to 20
// characters, a hyphen, and 1 to 9 digits with no leading zero. It is the E19 shape the catalog binds, so one rule
// serves both directions of the key.
var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}-[1-9][0-9]{0,8}$`)

// isBuiltField reports whether a field builds its value from an issue key instead of reading it.
func isBuiltField(f spec.Field) bool {
	return f.ProjectKey != "" || f.IssueID != "" || f.CollectionID != "" || f.BrowseURL != ""
}

// builtPointer names the pointer that a built field reads the issue key from.
func builtPointer(f spec.Field) string {
	switch {
	case f.ProjectKey != "":
		return f.ProjectKey
	case f.IssueID != "":
		return f.IssueID
	case f.CollectionID != "":
		return f.CollectionID
	default:
		return f.BrowseURL
	}
}

// issueKeyOf returns the issue key at the built pointer of the field: the text the value must pass. A missing, null,
// or non-text value, or a value outside the issue-key shape, is an unexpected response naming the field, never the
// value.
func issueKeyOf(name string, f spec.Field, obj map[string]any) (string, error) {
	raw, found := pointerValue(obj, builtPointer(f))
	if !found {
		return "", fieldError(name)
	}
	key, isText := raw.(string)
	if !isText || !issueKeyPattern.MatchString(key) {
		return "", fieldError(name)
	}
	return key, nil
}

// projectOfKey returns the project key of an issue key: the text before the last hyphen. The key has passed the
// shared check, so the hyphen is always there.
func projectOfKey(key string) string {
	return key[:strings.LastIndex(key, "-")]
}

// builtValue builds the value of a built ID field from the issue key at its pointer, charging the output text to the
// shared budget like any mapped text. project_key answers the text before the last hyphen, issue_id prefixes the key
// with issue-, collection_id prefixes the project part with project-, and browse_url joins the endpoint origin with
// /browse/ and the key, never a URL the reply carries.
func builtValue(name string, f spec.Field, obj map[string]any, base string, budget *int) (any, error) {
	key, err := issueKeyOf(name, f, obj)
	if err != nil {
		return nil, err
	}
	var out string
	switch {
	case f.ProjectKey != "":
		out = projectOfKey(key)
	case f.IssueID != "":
		out = issuePrefix + key
	case f.CollectionID != "":
		out = projectPrefix + projectOfKey(key)
	default:
		out = base + "/browse/" + key
	}
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}
