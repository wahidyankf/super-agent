package cli

// SecretLookup returns the lookup that the composition root injects as Deps.Getenv. The lookup reads each variable retriever
// accepts for the sources through getenv, and any other key reads as empty without asking getenv. The composition root
// passes the process environment as getenv. The Jira e-mail (EXAMPLE_JIRA_EMAIL) is listed because the lookup reads it, not
// because it is secret: it is an identity name, not a key.
//
// Each variable is named by a literal getenv call, so a reader can see every variable the tool reads. A new key belongs
// in this switch as well as in the source contract of the environment.
func SecretLookup(getenv func(string) string) func(string) string {
	return func(key string) string {
		switch key {
		case "EXAMPLE_OUTLINE_API_KEY":
			return getenv("EXAMPLE_OUTLINE_API_KEY")
		case "EXAMPLE_KIWI_API_KEY":
			return getenv("EXAMPLE_KIWI_API_KEY")
		case "EXAMPLE_JIRA_API_TOKEN":
			return getenv("EXAMPLE_JIRA_API_TOKEN")
		case "EXAMPLE_JIRA_EMAIL":
			return getenv("EXAMPLE_JIRA_EMAIL")
		}
		return ""
	}
}
