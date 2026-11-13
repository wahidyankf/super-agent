package jirastub

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config is everything a stub holds: the synthetic credentials, the dataset, the faults, and the cookie. The end-to-end
// layer writes a Config as JSON and gives it to the jirastubcmd process, so the process answers exactly what the test
// configured. A nil Fault means no fault is set. FaultLeft counts the replies the fault still applies to, where zero with
// a fault set means every reply, as SetFault does.
type Config struct {
	Email     string
	Token     string
	Account   Account
	Projects  []Project
	Issues    []Issue
	Comments  map[string][]Comment
	Fault     *Fault
	FaultLeft int
	Cookie    string
	// RawDescriptions holds, by issue key, each description that is not valid JSON. The issue itself carries a nil
	// description in Issues, because a JSON encoder cannot write such a description, and ApplyConfig puts it back.
	RawDescriptions map[string]string
}

// splitRawDescriptions returns a copy of issues with each description that is not valid JSON removed, and the removed text
// by issue key. A nil description is the JSON null, so it stays in the copy.
func splitRawDescriptions(issues []Issue) ([]Issue, map[string]string) {
	out := append([]Issue(nil), issues...)
	var raw map[string]string
	for i := range out {
		d := out[i].Description
		if len(d) == 0 || json.Valid(d) {
			continue
		}
		if raw == nil {
			raw = map[string]string{}
		}
		raw[out[i].Key] = string(d)
		out[i].Description = nil
	}
	return out, raw
}

// Config returns a snapshot of everything the stub holds, as a value that ApplyConfig can load into another stub.
func (s *Stub) Config() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	issues, raw := splitRawDescriptions(s.issues)
	c := Config{
		Email:           s.email,
		Token:           s.token,
		Account:         s.account,
		Projects:        append([]Project(nil), s.projects...),
		Issues:          issues,
		Comments:        copyComments(s.comments),
		FaultLeft:       s.faultLeft,
		Cookie:          s.cookie,
		RawDescriptions: raw,
	}
	if s.hasFault {
		f := s.fault
		c.Fault = &f
	}
	return c
}

// ApplyConfig replaces everything the stub holds with c. A fault in c with FaultLeft of zero applies to every reply.
func (s *Stub) ApplyConfig(c Config) {
	s.SetCredentials(c.Email, c.Token)
	s.SetAccount(c.Account)
	s.SetProjects(c.Projects)
	issues := append([]Issue(nil), c.Issues...)
	for i := range issues {
		if raw, ok := c.RawDescriptions[issues[i].Key]; ok {
			issues[i].Description = json.RawMessage(raw)
		}
	}
	s.SetIssues(issues)
	s.SetComments(c.Comments)
	s.mu.Lock()
	s.cookie = c.Cookie // the Set-Cookie value, as Config reports it
	s.mu.Unlock()
	if c.Fault == nil {
		s.ClearFault()
		return
	}
	if c.FaultLeft == 0 {
		s.SetFault(*c.Fault)
		return
	}
	s.SetFaultFirst(c.FaultLeft, *c.Fault)
}

// NewIssue returns one open task of the project in key, with the description as given. A nil description is the JSON null.
// Its ID follows the key's number, so two issues of one test never share an ID.
func NewIssue(key, summary string, description json.RawMessage) Issue {
	_, number, _ := strings.Cut(key, "-")
	if len(description) == 0 {
		description = json.RawMessage(`null`)
	}
	return Issue{
		ID: fmt.Sprintf("%d", 20000+atoi(number)), Key: key, Summary: summary, Type: "Task", Status: "To Do",
		Priority: "Medium", Reporter: "Fixture Author", Created: "2026-11-05T11:00:00.000+0700",
		Updated: "2026-11-05T12:00:00.000+0700", Labels: []string{}, Description: description,
	}
}

// NumberedIssues returns n issues of the project, numbered 1 to n, with a plain summary and no description.
func NumberedIssues(project string, n int) []Issue {
	out := make([]Issue, 0, n)
	for i := 1; i <= n; i++ {
		key := fmt.Sprintf("%s-%d", project, i)
		out = append(out, NewIssue(key, fmt.Sprintf("Numbered issue %d", i), nil))
	}
	return out
}

// NumberedProjects returns n projects with keys PRJ1 to PRJn. Each one has a plain name and the classic style.
func NumberedProjects(n int) []Project {
	out := make([]Project, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, Project{
			ID: fmt.Sprintf("%d", 30000+i), Key: fmt.Sprintf("PRJ%d", i), Name: fmt.Sprintf("Numbered project %d", i),
			Type: "software", Style: "classic",
		})
	}
	return out
}
