package operations

import "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"

// jiraOperations is the closed list of the Jira Cloud operations that Supports runs for the Jira kind.
// A new Jira row is one entry here and one row in jiraRows. The kind is read only, so the list holds no write.
var jiraOperations = []domain.OperationID{
	domain.OpJiraWhoami,
	domain.OpJiraProjectsList,
	domain.OpJiraIssuesSearch,
	domain.OpJiraIssuesGet,
	domain.OpJiraCommentsList,
}

// jiraRows are the five read operations of the Jira Cloud source. Jira has no write, admin, destructive,
// or critical operation, and no MCP tool, so Tool, Critical, and Target stay empty. A project is a collection and an issue
// is a document; the adapter maps the provider's words to the domain's.
var jiraRows = []Row{
	{
		ID:          domain.OpJiraWhoami,
		CLI:         "jira whoami",
		Summary:     "show the Jira account that the source's credentials sign in as",
		Description: "Shows the display name, the account type, and the active flag of the Jira account that answers. It does not show the e-mail address or the account ID. Use it to check the credentials.",
		Effect:      domain.EffectRead,
		Access:      domain.AccessRead,
		OpensWorld:  true,
		Fields:      []string{"displayName", "accountType", "active"},
	},
	{
		ID:          domain.OpJiraProjectsList,
		CLI:         "jira projects list",
		Summary:     "list one page of the Jira projects of a Jira source, which Retriever shows as collections",
		Description: "Lists one page of the projects of one Jira source. Pass query to filter them by name. Pass source when more than one jira source exists.",
		Effect:      domain.EffectRead,
		Access:      domain.AccessRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "query", Kind: KindQuery, Help: "a text to match against the project names, 1 to 1000 characters"},
			{Name: "limit", Kind: KindInt, Min: 1, Max: 50, Default: 25, Help: "the number of projects in the page, from 1 to 50"},
			{Name: "offset", Kind: KindOffset, Default: 0, Help: "the number of projects to skip"},
		},
		Fields:  []string{"key", "name", "type", "style"},
		ListKey: "projects",
		Noun:    "projects",
		Columns: []string{"key", "name", "type"},
	},
	{
		ID:          domain.OpJiraIssuesSearch,
		CLI:         "jira issues search",
		Summary:     "search the Jira issues that match a JQL query, one page of at most 100 issues",
		Description: "Runs one search with the JQL text, which Retriever sends as data and never changes. The reply holds at most the number of issues asked for. When more issues match, truncated is true; narrow the JQL or raise the limit.",
		Effect:      domain.EffectRead,
		Access:      domain.AccessRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "jql", Kind: KindQuery, Required: true, Help: "the JQL query, such as project = EXAMPLE AND status = \"In Progress\""},
			{Name: "limit", Kind: KindLimit, Default: 50, Help: "the number of issues to show, from 1 to 100"},
		},
		Fields:  []string{"key", "summary", "status", "type", "priority", "assignee", "updated", "projectKey", "truncated"},
		ListKey: "issues",
		Noun:    "issues",
		Columns: []string{"key", "status", "summary", "assignee"},
		Notice:  "more issues match; narrow the JQL or raise --limit",
	},
	{
		ID:          domain.OpJiraIssuesGet,
		CLI:         "jira issues get",
		Summary:     "print the Markdown of one Jira issue",
		Description: "Returns the Markdown of one issue, with its title and URL. The key is an upper-case issue key, such as EXAMPLE-12. Comments are not included; use jira comments list.",
		Effect:      domain.EffectRead,
		Access:      domain.AccessRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "issueKey", Kind: KindIssueKey, Required: true, Help: "the issue key, such as EXAMPLE-12"},
		},
		Fields: []string{"id", "title", "url", "markdown", "collectionId"},
		Body:   "markdown",
	},
	{
		ID:          domain.OpJiraCommentsList,
		CLI:         "jira comments list",
		Summary:     "list one page of the comments of one Jira issue, as Markdown",
		Description: "Lists one page of the comments of one issue, in the order Jira returns them (oldest first). One request is made.",
		Effect:      domain.EffectRead,
		Access:      domain.AccessRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "issueKey", Kind: KindIssueKey, Required: true, Help: "the issue key, such as EXAMPLE-12"},
			{Name: "limit", Kind: KindLimit, Default: 50, Help: "the number of comments in the page, from 1 to 100"},
			{Name: "offset", Kind: KindOffset, Default: 0, Help: "the number of comments to skip"},
		},
		Fields:  []string{"id", "author", "created", "updated", "markdown"},
		ListKey: "comments",
		Noun:    "comments",
		Columns: []string{"created", "author", "id"},
		Detail:  "markdown",
		Quiet:   true,
	},
}

func init() {
	rows = append(rows, jiraRows...)
}
