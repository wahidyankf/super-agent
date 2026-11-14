package kiwi

import (
	"context"
	"sort"
	"strconv"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// ListCollections returns every product of the server as a collection. Kiwi has no pagination, so one
// Product.filter request returns them all.
func (g *Gateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	cols, err := g.listCollections(ctx)
	if err != nil {
		return nil, kindErr(err)
	}
	return cols, nil
}

// listCollections is ListCollections without the kind wrap, for the callers inside the package.
func (g *Gateway) listCollections(ctx context.Context) ([]domain.Collection, error) {
	products, err := filterRows[wireProduct](ctx, g, methodProductFilter, map[string]any{})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Collection, 0, len(products))
	for _, p := range products {
		if !validNumber(p.ID) {
			return nil, domain.ErrUnexpectedResponse
		}
		out = append(out, domain.Collection{ID: productID(p.ID), Name: title(p.Name)})
	}
	return out, nil
}

// CollectionTree returns the test plans and test cases of one product as a flat list, depth first.
func (g *Gateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	nodes, err := g.collectionTree(ctx, id)
	if err != nil {
		return nil, kindErr(err)
	}
	return nodes, nil
}

// Document returns one test plan (plan-<n>) or one test case (case-<n>) with its Markdown.
func (g *Gateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	doc, err := g.document(ctx, id.String())
	if err != nil {
		return domain.SourceDocument{}, kindErr(err)
	}
	return doc, nil
}

// Read runs one of the three Kiwi read operations and returns its Record. The use case has checked
// the source already.
func (g *Gateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	rec, err := g.read(ctx, op, args)
	if err != nil {
		return domain.Record{}, kindErr(err)
	}
	return rec, nil
}

// read binds the arguments with the catalog's rules, so a value that reaches a request has passed them, then runs the
// operation.
func (g *Gateway) read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	row, ok := operations.Lookup(op)
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if row.Effect != domain.EffectRead {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	bound, err := operations.Bind(row, args.Values())
	if err != nil {
		return domain.Record{}, err
	}
	switch op {
	case domain.OpKiwiProductsList:
		return g.productsList(ctx)
	case domain.OpKiwiProductsTree:
		return g.productsTree(ctx, bound.String("collectionId"))
	case domain.OpKiwiDocumentsGet:
		return g.documentsGet(ctx, bound.String("documentId"))
	}
	return domain.Record{}, domain.ErrUnsupportedOperation
}

// productsList is the record of kiwi.products.list: one object per product, with its ID and name.
func (g *Gateway) productsList(ctx context.Context) (domain.Record, error) {
	cols, err := g.listCollections(ctx)
	if err != nil {
		return domain.Record{}, err
	}
	items := make([]domain.Record, 0, len(cols))
	for _, c := range cols {
		items = append(items, domain.NewRecord(field("id", c.ID), field("name", c.Name)))
	}
	return domain.NewList(items), nil
}

// productsTree is the record of kiwi.products.tree: one object per plan or case, with its parent and depth.
func (g *Gateway) productsTree(ctx context.Context, rawID string) (domain.Record, error) {
	id, err := domain.NewCollectionID(rawID)
	if err != nil {
		return domain.Record{}, err
	}
	nodes, err := g.collectionTree(ctx, id)
	if err != nil {
		return domain.Record{}, err
	}
	items := make([]domain.Record, 0, len(nodes))
	for _, n := range nodes {
		var parent any
		if n.ParentID != nil {
			parent = *n.ParentID
		}
		items = append(items, domain.NewRecord(
			field("id", n.ID), field("title", n.Title), field("url", n.URL),
			field("parentId", parent), field("depth", n.Depth),
		))
	}
	return domain.NewList(items), nil
}

// documentsGet is the record of kiwi.documents.get: the ID, title, URL, Markdown, and product of one document.
func (g *Gateway) documentsGet(ctx context.Context, rawID string) (domain.Record, error) {
	id, err := domain.NewDocumentID(rawID)
	if err != nil {
		return domain.Record{}, err
	}
	doc, err := g.document(ctx, id.String())
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", doc.ID), field("title", doc.Title), field("url", doc.URL),
		field("markdown", doc.Markdown), field("collectionId", doc.CollectionID),
	), nil
}

// collectionTree reads the plans of one product, then the cases of each plan, and flattens them depth first. The product
// must exist, and a product with more than planCap plans is refused before any case is read.
func (g *Gateway) collectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	n, err := parseProductID(id.String())
	if err != nil {
		return nil, err
	}
	products, err := filterRows[wireProduct](ctx, g, methodProductFilter, map[string]any{"id": n})
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, domain.ErrCollectionNotFound
	}
	plans, err := filterRows[wirePlan](ctx, g, methodTestPlanFilter, map[string]any{"product": n})
	if err != nil {
		return nil, err
	}
	if len(plans) > planCap {
		return nil, domain.ErrPageCapReached
	}
	for _, p := range plans {
		if !validNumber(p.ID) {
			return nil, domain.ErrUnexpectedResponse
		}
	}
	tree := newPlanTree(plans)
	var out []domain.TreeNode
	visited := map[int]bool{}
	for _, root := range tree.roots {
		if err := g.walkPlan(ctx, tree, root, 0, nil, visited, &out); err != nil {
			return nil, err
		}
	}
	// A loop of parent links is reached from no root, so each plan the roots did not reach is listed as a root too.
	for _, id := range tree.order {
		if err := g.walkPlan(ctx, tree, id, 0, nil, visited, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// planTree indexes the plans of one product by ID and by parent. Children, roots, and order are in ascending ID order.
type planTree struct {
	plans    map[int]wirePlan
	children map[int][]int
	roots    []int
	order    []int
}

// newPlanTree indexes the plans. A plan whose parent is not in the list is a root, so a missing parent never hides a plan.
func newPlanTree(plans []wirePlan) planTree {
	sorted := append([]wirePlan(nil), plans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	tree := planTree{plans: map[int]wirePlan{}, children: map[int][]int{}}
	for _, p := range sorted {
		tree.plans[p.ID] = p
		tree.order = append(tree.order, p.ID)
	}
	for _, p := range sorted {
		if p.Parent != nil && *p.Parent != p.ID {
			if _, known := tree.plans[*p.Parent]; known {
				tree.children[*p.Parent] = append(tree.children[*p.Parent], p.ID)
				continue
			}
		}
		tree.roots = append(tree.roots, p.ID)
	}
	return tree
}

// walkPlan appends a plan, then its cases in sort order, then its child plans. A plan is visited once, so a loop of parent
// links cannot repeat it.
func (g *Gateway) walkPlan(ctx context.Context, tree planTree, id, depth int, parent *string, visited map[int]bool, out *[]domain.TreeNode) error {
	if visited[id] {
		return nil
	}
	visited[id] = true
	plan := tree.plans[id]
	pid := planID(id)
	*out = append(*out, domain.TreeNode{ID: pid, Title: title(plan.Name), URL: g.planURL(id), ParentID: parent, Depth: depth})
	cases, err := g.planCases(ctx, id)
	if err != nil {
		return err
	}
	for _, c := range cases {
		under := pid
		*out = append(*out, domain.TreeNode{ID: caseID(c.ID), Title: title(c.Summary), URL: g.caseURL(c.ID), ParentID: &under, Depth: depth + 1})
	}
	for _, child := range tree.children[id] {
		under := pid
		if err := g.walkPlan(ctx, tree, child, depth+1, &under, visited, out); err != nil {
			return err
		}
	}
	return nil
}

// planCases returns the cases of one plan in their sort order: by sort key, then by ID. It reads two replies, the case
// rows and the sort key object that maps each case ID to its position. Every case ID is checked before it is used.
func (g *Gateway) planCases(ctx context.Context, plan int) ([]wireCase, error) {
	cases, err := filterRows[wireCase](ctx, g, methodTestCaseFilter, map[string]any{"plan": plan})
	if err != nil {
		return nil, err
	}
	for _, c := range cases {
		if !validNumber(c.ID) {
			return nil, domain.ErrUnexpectedResponse
		}
	}
	rawKeys, err := g.call(ctx, methodTestCaseSortkeys, map[string]any{"query": map[string]any{"plan": plan}})
	if err != nil {
		return nil, err
	}
	order, err := decodeSortkeys(rawKeys)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(cases, func(i, j int) bool {
		if order[cases[i].ID] != order[cases[j].ID] {
			return order[cases[i].ID] < order[cases[j].ID]
		}
		return cases[i].ID < cases[j].ID
	})
	return cases, nil
}

// document returns the plan or case that an ID names, with its Markdown.
func (g *Gateway) document(ctx context.Context, rawID string) (domain.SourceDocument, error) {
	kind, n, err := parseDocumentID(rawID)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	if kind == docPlan {
		return g.readPlan(ctx, n)
	}
	return g.readCase(ctx, n)
}

// readPlan reads one plan, its cases, and its parent into a plan document.
func (g *Gateway) readPlan(ctx context.Context, n int) (domain.SourceDocument, error) {
	plans, err := filterRows[wirePlan](ctx, g, methodTestPlanFilter, map[string]any{"id": n})
	if err != nil {
		return domain.SourceDocument{}, err
	}
	if len(plans) == 0 {
		return domain.SourceDocument{}, domain.ErrDocumentNotFound
	}
	plan := plans[0]
	if !validNumber(plan.Product) {
		return domain.SourceDocument{}, domain.ErrUnexpectedResponse
	}
	cases, err := g.planCases(ctx, n)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	lines := make([]caseLine, 0, len(cases))
	for _, c := range cases {
		lines = append(lines, caseLine{ID: caseID(c.ID), Summary: c.Summary})
	}
	var parent string
	if plan.Parent != nil && validNumber(*plan.Parent) {
		parent = planID(*plan.Parent)
	}
	return domain.SourceDocument{
		ID:    planID(n),
		Title: title(plan.Name),
		URL:   g.planURL(n),
		Markdown: planMarkdown(planDocument{
			Name:    plan.Name,
			Product: productID(plan.Product),
			Parent:  parent,
			Text:    plan.Text,
			Cases:   lines,
		}),
		CollectionID: productID(plan.Product),
	}, nil
}

// readCase reads one case and resolves its category, priority, and status names from the lookup tables. The product
// of a case is the product of its category.
func (g *Gateway) readCase(ctx context.Context, n int) (domain.SourceDocument, error) {
	cases, err := filterRows[wireCase](ctx, g, methodTestCaseFilter, map[string]any{"id": n})
	if err != nil {
		return domain.SourceDocument{}, err
	}
	if len(cases) == 0 {
		return domain.SourceDocument{}, domain.ErrDocumentNotFound
	}
	c := cases[0]
	tables, err := g.loadLookups(ctx)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	category := tables.categories[c.Category]
	var collection string
	if validNumber(category.Product) {
		collection = productID(category.Product)
	}
	return domain.SourceDocument{
		ID:    caseID(n),
		Title: title(c.Summary),
		URL:   g.caseURL(n),
		Markdown: caseMarkdown(caseDocument{
			Summary:   c.Summary,
			Status:    tables.statuses[c.CaseStatus].Name,
			Priority:  tables.priorities[c.Priority].Value,
			Category:  category.Name,
			Text:      c.Text,
			Setup:     c.Setup,
			Action:    c.Action,
			Effect:    c.Effect,
			Breakdown: c.Breakdown,
		}),
		CollectionID: collection,
	}, nil
}

// planURL and caseURL build a document's URL from the catalog endpoint and the number. A URL in a reply is never used.
func (g *Gateway) planURL(n int) string { return g.endpoint + "/plan/" + strconv.Itoa(n) + "/" }
func (g *Gateway) caseURL(n int) string { return g.endpoint + "/case/" + strconv.Itoa(n) + "/" }
