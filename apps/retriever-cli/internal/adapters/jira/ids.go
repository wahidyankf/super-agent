package jira

import (
	"regexp"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// Prefixes of the collection and document IDs that Retriever forms for Jira.
const (
	projectPrefix = "project-"
	issuePrefix   = "issue-"
)

// projectKeyPattern is the shape of a Jira project key: upper case, 2 to 20 characters.
var projectKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}$`)

// issueKeyPattern is the shape of a Jira issue key: a project key, a hyphen, and a number of 1 to 9 digits with no leading zero.
var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}-[1-9][0-9]{0,8}$`)

// ProjectCollectionID returns the collection ID of a project key, which is project-<KEY>. A key that is not upper case or
// does not match the pattern is refused, not corrected.
func ProjectCollectionID(key string) (domain.CollectionID, error) {
	if !projectKeyPattern.MatchString(key) {
		return domain.CollectionID{}, domain.ErrInvalidCollectionID
	}
	return domain.NewCollectionID(projectPrefix + key)
}

// ProjectKey returns the project key of a collection ID. Only the form project-<KEY> is accepted.
func ProjectKey(id domain.CollectionID) (string, error) {
	key, ok := strings.CutPrefix(id.String(), projectPrefix)
	if !ok || !projectKeyPattern.MatchString(key) {
		return "", domain.ErrInvalidCollectionID
	}
	return key, nil
}

// ParseIssueKey checks a bare issue key, such as EXAMPLE-12, as a user types it. It is refused, not corrected, when it is not
// upper case or does not match the pattern.
func ParseIssueKey(raw string) (string, error) {
	if !issueKeyPattern.MatchString(raw) {
		return "", domain.ErrInvalidDocumentID
	}
	return raw, nil
}

// IssueDocumentID returns the document ID of a bare issue key, which is issue-<KEY-123>.
func IssueDocumentID(key string) (domain.DocumentID, error) {
	checked, err := ParseIssueKey(key)
	if err != nil {
		return domain.DocumentID{}, err
	}
	return domain.NewDocumentID(issuePrefix + checked)
}

// IssueKey returns the issue key of a document ID. Only the form issue-<KEY-123> is accepted; a project ID in a document
// position is refused.
func IssueKey(id domain.DocumentID) (string, error) {
	key, ok := strings.CutPrefix(id.String(), issuePrefix)
	if !ok {
		return "", domain.ErrInvalidDocumentID
	}
	return ParseIssueKey(key)
}
