package domain

// Collection is one collection of a source wiki.
type Collection struct {
	ID   string
	Name string
}

// SourceDocument is one document fetched from a source, with its Markdown body.
type SourceDocument struct {
	ID           string
	Title        string
	URL          string
	Markdown     string
	CollectionID string
}

// TreeNode is one entry of a flat collection tree. ParentID is nil for a root
// document, and Depth is 0 for a root. The caller rebuilds the hierarchy.
type TreeNode struct {
	ID       string
	Title    string
	URL      string
	ParentID *string
	Depth    int
}
