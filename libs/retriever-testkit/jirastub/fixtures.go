package jirastub

import "encoding/json"

// Keys of the synthetic dataset. Tests name them through these constants, not by copying the literals.
const (
	ProjectKey    = "EXAMPLE"
	OtherProject  = "OPS"
	IssueKey      = "EXAMPLE-12"
	ParentKey     = "EXAMPLE-1"
	UnassignedKey = "EXAMPLE-13"
	HostileKey    = "EXAMPLE-14"
	OtherKey      = "OPS-3"
	MissingKey    = "EXAMPLE-999"
)

// Project is one Jira project the stub holds.
type Project struct {
	ID    string
	Key   string
	Name  string
	Type  string
	Style string
}

// Issue is one Jira issue the stub holds. An empty Assignee, Resolution, Parent, or Due is sent as null. Description is the
// raw ADF JSON value, and the JSON null when it is empty.
type Issue struct {
	ID          string
	Key         string
	Summary     string
	Type        string
	Status      string
	Priority    string
	Resolution  string
	Assignee    string
	Reporter    string
	Parent      string
	Due         string
	Created     string
	Updated     string
	Labels      []string
	Description json.RawMessage
}

// Comment is one comment of an issue. Body is the raw ADF JSON value.
type Comment struct {
	ID      string
	Author  string
	Created string
	Updated string
	Body    json.RawMessage
}

// Account is the account that GET /rest/api/3/myself answers with. The e-mail address is the one the stub was built with.
type Account struct {
	AccountID   string
	DisplayName string
	AccountType string
	Active      bool
}

// defaultProjects are the two projects of the synthetic site.
var defaultProjects = []Project{
	{ID: "10001", Key: ProjectKey, Name: "Example Org", Type: "software", Style: "classic"},
	{ID: "10002", Key: OtherProject, Name: "Operations", Type: "business", Style: "next-gen"},
}

// defaultAccount is the signed-in account of the synthetic site.
var defaultAccount = Account{AccountID: "5b10ac8d82e05b22cc7d4ef5", DisplayName: "Fixture Reviewer", AccountType: "atlassian", Active: true}

// defaultIssues are the issues of the synthetic site. Each description is written in ADF.
var defaultIssues = []Issue{
	{
		ID: "10001", Key: ParentKey, Summary: "Epic: login rework", Type: "Epic", Status: "In Progress", Priority: "High",
		Assignee: "Fixture Reviewer", Reporter: "Fixture Reviewer", Created: "2026-10-01T08:00:00.000+0700",
		Updated: "2026-11-03T10:00:00.000+0700", Labels: []string{"auth"},
		Description: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Rework the login flow."}]}]}`),
	},
	{
		ID: "10012", Key: IssueKey, Summary: "Login fails on second try", Type: "Bug", Status: "To Do", Priority: "Medium",
		Assignee: "Fixture Reviewer", Reporter: "Fixture Author", Parent: ParentKey,
		Created: "2026-11-02T09:15:00.000+0700", Updated: "2026-11-04T16:30:00.000+0700", Due: "2026-12-01",
		Labels:      []string{"login", "beta"},
		Description: json.RawMessage(issueTwelveADF),
	},
	{
		ID: "10013", Key: UnassignedKey, Summary: "Unassigned task", Type: "Task", Status: "Done", Priority: "Low",
		Resolution: "Done", Reporter: "Fixture Author", Created: "2026-11-05T11:00:00.000+0700",
		Updated: "2026-11-05T12:00:00.000+0700", Labels: []string{},
		Description: json.RawMessage(`null`),
	},
	{
		ID: "10014", Key: HostileKey, Summary: "Hostile <b>summary</b> with [link](javascript:alert(1))", Type: "Bug",
		Status: "To Do", Priority: "Highest", Reporter: "Fixture Author", Created: "2026-11-06T09:00:00.000+0700",
		Updated: "2026-11-06T09:00:00.000+0700", Labels: []string{},
		Description: json.RawMessage(hostileADF),
	},
	{
		ID: "10020", Key: OtherKey, Summary: "Weekly rota", Type: "Task", Status: "Done", Priority: "Medium",
		Resolution: "Done", Assignee: "Fixture Author", Reporter: "Fixture Author",
		Created: "2026-11-07T07:00:00.000+0700", Updated: "2026-11-07T07:30:00.000+0700", Labels: []string{},
		Description: json.RawMessage(`null`),
	},
}

// issueTwelveADF is the description of EXAMPLE-12. It holds a heading, a list, a code block, a link, a mention, a date, and a
// media item, so the renderer and the export see each node type the allow-list covers.
const issueTwelveADF = `{"type":"doc","version":1,"content":[` +
	`{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Steps"}]},` +
	`{"type":"bulletList","content":[` +
	`{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Open the login page"}]}]},` +
	`{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Submit twice"}]}]}]},` +
	`{"type":"codeBlock","attrs":{"language":"text"},"content":[{"type":"text","text":"HTTP 500 on the second call"}]},` +
	`{"type":"paragraph","content":[` +
	`{"type":"text","text":"See the "},` +
	`{"type":"text","text":"runbook","marks":[{"type":"link","attrs":{"href":"https://wiki.example.com/runbook"}}]},` +
	`{"type":"text","text":" and ask "},` +
	`{"type":"mention","attrs":{"id":"5b10ac8d82e05b22cc7d4ef5","text":"@Fixture Reviewer"}},` +
	`{"type":"text","text":" on "},` +
	`{"type":"date","attrs":{"timestamp":"1767225600000"}}]},` +
	`{"type":"mediaSingle","content":[{"type":"media","attrs":{"type":"file","id":"media-secret-1","alt":"screenshot.png"}}]}` +
	`]}`

// hostileADF is the description of EXAMPLE-14. Its text holds HTML, Markdown structure, a script link, a bidirectional
// override, and an escape sequence, so the export is tested against every class of hostile text.
const hostileADF = `{"type":"doc","version":1,"content":[` +
	`{"type":"paragraph","content":[{"type":"text","text":"<script>alert(1)</script> # heading | cell"}]},` +
	`{"type":"paragraph","content":[` +
	`{"type":"text","text":"click here","marks":[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]},` +
	`{"type":"text","text":" and‮ reversed​"}]},` +
	`{"type":"paragraph","content":[{"type":"text","text":"\u001b[31mred\u001b[0m"}]}` +
	`]}`

// defaultComments are the comments of EXAMPLE-12. Other issues have none.
var defaultComments = map[string][]Comment{
	IssueKey: {
		{
			ID: "20001", Author: "Fixture Author", Created: "2026-11-02T10:00:00.000+0700", Updated: "2026-11-02T10:00:00.000+0700",
			Body: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Reproduced on staging."}]}]}`),
		},
		{
			ID: "20002", Author: "Fixture Reviewer", Created: "2026-11-03T08:30:00.000+0700", Updated: "2026-11-03T09:00:00.000+0700",
			Body: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Fix is in review."},{"type":"hardBreak"},{"type":"text","text":"- not a list"}]}]}`),
		},
	},
}
