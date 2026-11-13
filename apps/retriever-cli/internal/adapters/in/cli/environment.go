package cli

import "os"

// SecretEnv is the process environment lookup that the composition root can inject as Deps.Getenv. It reads only the
// variables retriever accepts for the sources; any other key reads as empty. The Jira e-mail (EXAMPLE_JIRA_EMAIL) is listed
// here because the lookup reads it, not because it is secret: it is an identity name, not a key (ADR 010 section 1.6).
//
// Each variable is named by a literal os.Getenv call, so a reader can see every variable the tool reads. A new key belongs
// in this switch as well as in the source contract of the environment.
func SecretEnv(key string) string {
	switch key {
	case "EXAMPLE_OUTLINE_API_KEY":
		return os.Getenv("EXAMPLE_OUTLINE_API_KEY")
	case "EXAMPLE_KIWI_API_KEY":
		return os.Getenv("EXAMPLE_KIWI_API_KEY")
	case "EXAMPLE_JIRA_API_TOKEN":
		return os.Getenv("EXAMPLE_JIRA_API_TOKEN")
	case "EXAMPLE_JIRA_EMAIL":
		return os.Getenv("EXAMPLE_JIRA_EMAIL")
	}
	return ""
}
