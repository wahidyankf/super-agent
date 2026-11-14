package operations

import "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"

// kiwiOperations is the closed list of the Kiwi TCMS operations that Supports runs for the Kiwi kind.
// A new Kiwi row is one entry here and one row in kiwiRows.
var kiwiOperations = []domain.OperationID{
	domain.OpKiwiProductsList,
	domain.OpKiwiProductsTree,
	domain.OpKiwiDocumentsGet,
	domain.OpKiwiTestCasesCreate,
	domain.OpKiwiTestCasesUpdate,
	domain.OpKiwiPlansAddCase,
}

// kiwiRows are the six operations of the Kiwi TCMS source: three reads and three writes. Kiwi has no admin or destructive operation, and no MCP tool, so Tool stays
// empty. Retriever maps the provider's words to the domain's:
// a product is a collection, a test plan is a tree node, and a test case is a document.
var kiwiRows = []Row{
	{
		ID:          domain.OpKiwiProductsList,
		CLI:         "kiwi products list",
		Summary:     "list the products of a Kiwi TCMS source, which Retriever shows as collections",
		Description: "Lists the products of one Kiwi TCMS source. Pass source when more than one kiwi source exists.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Fields:      []string{"id", "name"},
		ListKey:     "products",
		Noun:        "products",
		Columns:     []string{"id", "name"},
	},
	{
		ID:          domain.OpKiwiProductsTree,
		CLI:         "kiwi products tree",
		Summary:     "show the test plans and test cases of one Kiwi product as a flat tree",
		Description: "Lists the test plans and the test cases of one product as a flat tree. Each node names its parent and its depth; a plan with no parent is a root.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "collectionId", Kind: KindCollectionID, Required: true, Help: "the product ID, such as product-3, from kiwi products list"},
		},
		Fields:  []string{"id", "title", "url", "parentId", "depth"},
		ListKey: "nodes",
		Noun:    "test plans and cases",
		Columns: []string{"title", "id", "url"},
		Indent:  "depth",
	},
	{
		ID:          domain.OpKiwiDocumentsGet,
		CLI:         "kiwi documents get",
		Summary:     "print the Markdown of one Kiwi test plan or test case",
		Description: "Returns the Markdown of one test plan (plan-<n>) or one test case (case-<n>) with its title and URL.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the plan ID, such as plan-12, or the case ID, such as case-40"},
		},
		Body:   "markdown",
		Fields: []string{"id", "title", "url", "markdown", "collectionId"},
	},
	{
		ID:          domain.OpKiwiTestCasesCreate,
		CLI:         "kiwi test-cases create",
		Summary:     "create one Kiwi test case, given the names of its category, priority, and status",
		Description: "Creates one test case in a product. The category and the priority are names that Retriever resolves to IDs; a name that matches no entry, or more than one, is refused before the change. This command changes data in Kiwi.",
		Effect:      domain.EffectAdditive,
		OpensWorld:  true,
		Params: []Param{
			{Name: "summary", Kind: KindName, Required: true, Help: "the summary of the case, 1 to 255 characters"},
			{Name: "product", Kind: KindCollectionID, Required: true, Help: "the product the case goes in, such as product-3"},
			{Name: "category", Kind: KindName, Required: true, Help: "the category name, as Kiwi holds it"},
			{Name: "priority", Kind: KindName, Required: true, Help: "the priority name, as Kiwi holds it"},
			{Name: "status", Kind: KindName, Help: "the status name, as Kiwi holds it"},
			{Name: "text", Kind: KindText, Help: "the Markdown body of the case, at most 1 MiB"},
		},
		Fields: []string{"id", "title", "url"},
	},
	{
		ID:          domain.OpKiwiTestCasesUpdate,
		CLI:         "kiwi test-cases update",
		Summary:     "change the summary, text, status, or priority of one Kiwi test case",
		Description: "Changes the named fields of one test case. At least one of summary, text, status, or priority is required. The category is not changed. This command changes data in Kiwi.",
		Effect:      domain.EffectUpdate,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the case ID, such as case-40"},
			{Name: "summary", Kind: KindName, Help: "the new summary, 1 to 255 characters"},
			{Name: "text", Kind: KindText, Help: "the new Markdown body, at most 1 MiB"},
			{Name: "status", Kind: KindName, Help: "the new status name, as Kiwi holds it"},
			{Name: "priority", Kind: KindName, Help: "the new priority name, as Kiwi holds it"},
		},
		AnyOf:  []string{"summary", "text", "status", "priority"},
		Fields: []string{"id", "title", "url"},
	},
	{
		ID:          domain.OpKiwiPlansAddCase,
		CLI:         "kiwi plans add-case",
		Summary:     "add one Kiwi test case to one test plan",
		Description: "Adds a test case to a test plan. The plan ID is plan-<n> and the case ID is case-<n>. This command changes data in Kiwi.",
		Effect:      domain.EffectAdditive,
		OpensWorld:  true,
		Params: []Param{
			{Name: "planId", Kind: KindDocumentID, Required: true, Help: "the test plan ID, such as plan-12"},
			{Name: "caseId", Kind: KindDocumentID, Required: true, Help: "the test case ID, such as case-40"},
		},
		Fields: []string{"planId", "caseId"},
	},
}

func init() {
	rows = append(rows, kiwiRows...)
}
