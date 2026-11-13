package cli

// OutlineLeafPaths returns the full path of every leaf command under the outline group, such as "outline documents get",
// in the order of the command tree. The catalog sync test compares it with the rows of the operation catalog (ADR 007
// section 4), so a command that no row describes is found.
func OutlineLeafPaths() []string {
	var out []string
	var walk func(prefix string, c *command)
	walk = func(prefix string, c *command) {
		for _, ch := range c.children {
			path := prefix + " " + ch.name
			if ch.leaf != nil {
				out = append(out, path)
			}
			walk(path, ch)
		}
	}
	walk("outline", commandTree().child("outline"))
	return out
}
