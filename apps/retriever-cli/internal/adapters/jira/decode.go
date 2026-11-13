package jira

import (
	"encoding/json"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// This file holds the shapes of the Jira replies that Retriever reads. Each struct names only the members Retriever uses. Unknown
// members are ignored (ADR 010 section 5). A member Retriever needs is a pointer or a required value, so a reply that lacks it
// is an unexpected reply and not a silent empty value. Only the members of ADR 010 section 3 are read: no e-mail address, no
// account ID, and no avatar or self URL is ever decoded.

// myselfReply is the reply of GET /rest/api/3/myself. The display name and the active flag are required, because whoami
// shows them. The account type is read when present.
type myselfReply struct {
	DisplayName *string `json:"displayName"`
	AccountType string  `json:"accountType"`
	Active      *bool   `json:"active"`
}

// record returns the three fields that whoami shows, or ErrUnexpectedResponse when a required member is missing.
func (r myselfReply) record() (domain.Record, error) {
	if r.DisplayName == nil || r.Active == nil {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return domain.NewRecord(
		domain.Field{Name: "displayName", Value: plainText(*r.DisplayName)},
		domain.Field{Name: "accountType", Value: plainText(r.AccountType)},
		domain.Field{Name: "active", Value: *r.Active},
	), nil
}

// projectsReply is the reply of GET /rest/api/3/project/search: one page of projects. Values and isLast are required.
type projectsReply struct {
	Values []projectValue `json:"values"`
	IsLast *bool          `json:"isLast"`
}

// projectValue is one project of a page. Its key must pass the collection rule, and its name is required.
type projectValue struct {
	Key            string  `json:"key"`
	Name           *string `json:"name"`
	ProjectTypeKey string  `json:"projectTypeKey"`
	Style          string  `json:"style"`
}

// project is one project as Retriever shows it.
type project struct {
	Key   string
	Name  string
	Type  string
	Style string
}

// projects returns the checked projects of the page, and whether the server says it is the last page.
func (r projectsReply) projects() ([]project, bool, error) {
	if r.Values == nil || r.IsLast == nil {
		return nil, false, domain.ErrUnexpectedResponse
	}
	out := make([]project, 0, len(r.Values))
	for _, v := range r.Values {
		if _, err := ProjectCollectionID(v.Key); err != nil || v.Name == nil {
			return nil, false, domain.ErrUnexpectedResponse
		}
		out = append(out, project{
			Key:   v.Key,
			Name:  plainText(*v.Name),
			Type:  plainText(v.ProjectTypeKey),
			Style: plainText(v.Style),
		})
	}
	return out, *r.IsLast, nil
}

// searchRequest is the JSON body of POST /rest/api/3/search/jql. Only these members are sent. The JQL is one string value, and
// expand and properties are never sent (ADR 010 section 5).
type searchRequest struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults"`
	Fields        []string `json:"fields"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// searchFields is the fixed list of fields that a search returns. The caller never sets it.
var searchFields = []string{"summary", "status", "issuetype", "priority", "assignee", "updated", "project", "parent"}

// issueFieldsList is the fixed list of fields that the issue endpoint returns, in the query string of the request.
const issueFieldsList = "summary,description,issuetype,status,priority,resolution,assignee,reporter,labels,created,updated,duedate,project,parent"

// searchReply is the reply of POST /rest/api/3/search/jql: one page of issues, and a token when more issues match.
type searchReply struct {
	Issues        []issueBody `json:"issues"`
	IsLast        bool        `json:"isLast"`
	NextPageToken string      `json:"nextPageToken"`
}

// issueBody is one issue as the search and the issue endpoint send it. The key and the fields object are required, and the
// summary inside the fields is required.
type issueBody struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Fields *issueFields `json:"fields"`
}

// issueFields holds the fields of an issue that Retriever reads. Each one is optional except the summary, which the issue
// document and the search row both need.
type issueFields struct {
	Summary     *string         `json:"summary"`
	Description json.RawMessage `json:"description"`
	IssueType   *named          `json:"issuetype"`
	Status      *named          `json:"status"`
	Priority    *named          `json:"priority"`
	Resolution  *named          `json:"resolution"`
	Assignee    *person         `json:"assignee"`
	Reporter    *person         `json:"reporter"`
	Labels      []string        `json:"labels"`
	Created     string          `json:"created"`
	Updated     string          `json:"updated"`
	DueDate     *string         `json:"duedate"`
	Project     *keyed          `json:"project"`
	Parent      *keyed          `json:"parent"`
}

// issueReply is the reply of GET /rest/api/3/issue/{key}. The ID, the key that must be the one asked for, and the fields are
// required.
type issueReply struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Fields *issueFields `json:"fields"`
}

// commentsReply is the reply of GET /rest/api/3/issue/{key}/comment: one page of comments.
type commentsReply struct {
	Comments []commentBody `json:"comments"`
}

// commentBody is one comment. Its ID is required. Its body is the ADF value, rendered by RenderADF.
type commentBody struct {
	ID      string          `json:"id"`
	Author  *person         `json:"author"`
	Created string          `json:"created"`
	Updated string          `json:"updated"`
	Body    json.RawMessage `json:"body"`
}

// maxCommentIDDigits is the longest numeric comment ID that Retriever accepts (ADR 010 section 3).
const maxCommentIDDigits = 18

// checkComments returns the comments of a page, each with a numeric ID. A page with no comments member is an unexpected reply.
func (r commentsReply) checkComments() ([]commentBody, error) {
	if r.Comments == nil {
		return nil, domain.ErrUnexpectedResponse
	}
	for _, c := range r.Comments {
		if !isCommentID(c.ID) {
			return nil, domain.ErrUnexpectedResponse
		}
	}
	return r.Comments, nil
}

// isCommentID reports whether id is a non-empty run of digits of at most maxCommentIDDigits.
func isCommentID(id string) bool {
	if id == "" || len(id) > maxCommentIDDigits {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	return true
}

// named is an object with a name, such as a status or a priority.
type named struct {
	Name string `json:"name"`
}

// person is an account. Only the display name is read; the e-mail address and the account ID are not part of this type.
type person struct {
	DisplayName string `json:"displayName"`
}

// keyed is an object with a key, such as a project or a parent issue.
type keyed struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// checkIssue reports whether an issue has a key that passes the issue pattern and a fields object with a summary.
func checkIssue(b issueBody) error {
	if _, err := ParseIssueKey(b.Key); err != nil {
		return domain.ErrUnexpectedResponse
	}
	if b.Fields == nil || b.Fields.Summary == nil {
		return domain.ErrUnexpectedResponse
	}
	return nil
}

// issues returns the issues of a search page, each one checked. A page with no issues member is an unexpected reply.
func (r searchReply) issues() ([]issueBody, error) {
	if r.Issues == nil {
		return nil, domain.ErrUnexpectedResponse
	}
	for _, b := range r.Issues {
		if err := checkIssue(b); err != nil {
			return nil, err
		}
	}
	return r.Issues, nil
}

// namedText returns the name of an optional named object, or "" when it is absent.
func namedText(n *named) string {
	if n == nil {
		return ""
	}
	return plainText(n.Name)
}

// personText returns the display name of an optional account, or "" when it is absent.
func personText(p *person) string {
	if p == nil {
		return ""
	}
	return plainText(p.DisplayName)
}
