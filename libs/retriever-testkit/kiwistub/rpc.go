package kiwistub

import (
	"fmt"
	"strconv"
)

// dispatch runs one JSON-RPC method against the dataset and returns its result or its error. The caller holds the lock.
// The method names and the reply fields are the ten methods of the Kiwi adapter's allowlist.
func (s *Stub) dispatch(method string, p map[string]any) (any, *rpcErr) {
	switch method {
	case "Product.filter":
		return s.filter(query(p), s.productRows()), nil
	case "TestPlan.filter":
		return s.filter(query(p), s.planRows()), nil
	case "TestCase.filter":
		q := query(p)
		if plan, ok := q["plan"]; ok {
			delete(q, "plan")
			return s.filter(q, s.caseRows(s.casesOfPlan(plan))), nil
		}
		return s.filter(q, s.caseRows(s.data.Cases)), nil
	case "TestCase.sortkeys":
		plan, ok := query(p)["plan"]
		if !ok {
			return nil, &rpcErr{code: -32602, message: "Invalid params"}
		}
		return s.sortkeys(plan), nil
	case "Category.filter":
		return s.filter(query(p), s.categoryRows()), nil
	case "Priority.filter":
		return s.filter(query(p), s.priorityRows()), nil
	case "TestCaseStatus.filter":
		return s.filter(query(p), s.statusRows()), nil
	case "TestCase.create":
		return s.createCase(values(p))
	case "TestCase.update":
		return s.updateCase(p)
	case "TestPlan.add_case":
		return s.addCase(p)
	}
	return nil, &rpcErr{code: -32601, message: "Method not found"}
}

// query returns a fresh copy of the named query object, so a filter never changes the recorded params.
func query(p map[string]any) map[string]any {
	return copyObject(p["query"])
}

// values returns a fresh copy of the named values object.
func values(p map[string]any) map[string]any {
	return copyObject(p["values"])
}

// copyObject copies a JSON object, and returns an empty object for anything else.
func copyObject(v any) map[string]any {
	out := map[string]any{}
	if m, ok := v.(map[string]any); ok {
		for key, value := range m {
			out[key] = value
		}
	}
	return out
}

// num reads a JSON number as an int. A JSON number is a float64 after decoding.
func num(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

// filter returns the rows that match every key of the query. A key the row does not hold matches nothing.
func (s *Stub) filter(q map[string]any, rows []map[string]any) []any {
	out := make([]any, 0, len(rows))
	for _, row := range rows {
		if matches(row, q) {
			out = append(out, row)
		}
	}
	return out
}

// matches reports whether each query key equals the row's value for that key, compared as text.
func matches(row, q map[string]any) bool {
	for key, want := range q {
		have, ok := row[key]
		if !ok || fmt.Sprint(have) != fmt.Sprint(want) {
			return false
		}
	}
	return true
}

// productRows returns one row per product: its ID and its name.
func (s *Stub) productRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.data.Products))
	for _, p := range s.data.Products {
		rows = append(rows, map[string]any{"id": p.ID, "name": p.Name})
	}
	return rows
}

// planRows returns one row per plan. A root plan has a null parent.
func (s *Stub) planRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.data.Plans))
	for _, p := range s.data.Plans {
		var parent any
		if p.Parent != 0 {
			parent = p.Parent
		}
		rows = append(rows, map[string]any{"id": p.ID, "name": p.Name, "product": p.Product, "parent": parent, "text": p.Text})
	}
	return rows
}

// caseRows returns one row per case, with the lookup IDs and the legacy fields.
func (s *Stub) caseRows(cases []Case) []map[string]any {
	rows := make([]map[string]any, 0, len(cases))
	for _, c := range cases {
		rows = append(rows, map[string]any{
			"id": c.ID, "summary": c.Summary, "text": c.Text, "category": c.Category,
			"priority": c.Priority, "case_status": c.Status,
			"setup": c.Legacy.Setup, "action": c.Legacy.Action, "effect": c.Legacy.Effect, "breakdown": c.Legacy.Breakdown,
		})
	}
	return rows
}

// casesOfPlan returns the cases that belong to the plan the value names.
func (s *Stub) casesOfPlan(plan any) []Case {
	n, ok := num(plan)
	if !ok {
		return nil
	}
	var out []Case
	for _, c := range s.data.Cases {
		for _, p := range c.Plans {
			if p == n {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// sortkeys returns the sort key of each case of the plan the value names. The reply has the shape the live Kiwi gives: one
// object that maps each case ID, as a string key, to its sort key number. A plan without cases gets an empty object.
func (s *Stub) sortkeys(plan any) map[string]any {
	out := map[string]any{}
	for _, c := range s.casesOfPlan(plan) {
		out[strconv.Itoa(c.ID)] = c.Sortkey
	}
	return out
}

// categoryRows, priorityRows, and statusRows return the lookup tables.
func (s *Stub) categoryRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.data.Categories))
	for _, c := range s.data.Categories {
		rows = append(rows, map[string]any{"id": c.ID, "name": c.Name, "product": c.Product})
	}
	return rows
}

func (s *Stub) priorityRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.data.Priorities))
	for _, p := range s.data.Priorities {
		rows = append(rows, map[string]any{"id": p.ID, "value": p.Value})
	}
	return rows
}

func (s *Stub) statusRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.data.Statuses))
	for _, st := range s.data.Statuses {
		rows = append(rows, map[string]any{"id": st.ID, "name": st.Name})
	}
	return rows
}

// nextCaseID returns one more than the largest case ID, so a created case is new.
func (s *Stub) nextCaseID() int {
	next := 1
	for _, c := range s.data.Cases {
		if c.ID >= next {
			next = c.ID + 1
		}
	}
	return next
}

// createCase adds a case from the values of TestCase.create. The summary is required, as Kiwi requires it.
func (s *Stub) createCase(v map[string]any) (any, *rpcErr) {
	summary, ok := v["summary"].(string)
	if !ok {
		return nil, &rpcErr{code: -32603, message: "This field is required: summary."}
	}
	c := Case{ID: s.nextCaseID(), Summary: summary}
	applyCaseValues(&c, v)
	s.data.Cases = append(s.data.Cases, c)
	return map[string]any{"id": c.ID, "summary": c.Summary}, nil
}

// updateCase changes the given fields of one case. A case that does not exist is a Kiwi not-found message.
func (s *Stub) updateCase(p map[string]any) (any, *rpcErr) {
	id, ok := num(p["case_id"])
	if !ok {
		return nil, &rpcErr{code: -32602, message: "Invalid params"}
	}
	for i := range s.data.Cases {
		if s.data.Cases[i].ID == id {
			applyCaseValues(&s.data.Cases[i], values(p))
			return map[string]any{"id": id, "summary": s.data.Cases[i].Summary}, nil
		}
	}
	return nil, &rpcErr{code: -32603, message: "TestCase matching query does not exist."}
}

// applyCaseValues sets the fields a create or an update names. A field the values do not name keeps its value.
func applyCaseValues(c *Case, v map[string]any) {
	if s, ok := v["summary"].(string); ok {
		c.Summary = s
	}
	if t, ok := v["text"].(string); ok {
		c.Text = t
	}
	if n, ok := num(v["category"]); ok {
		c.Category = n
	}
	if n, ok := num(v["priority"]); ok {
		c.Priority = n
	}
	if n, ok := num(v["case_status"]); ok {
		c.Status = n
	}
}

// addCase adds one case to one plan. Each of the two must exist, as Kiwi requires.
func (s *Stub) addCase(p map[string]any) (any, *rpcErr) {
	planID, planOK := num(p["plan_id"])
	caseID, caseOK := num(p["case_id"])
	if !planOK || !caseOK {
		return nil, &rpcErr{code: -32602, message: "Invalid params"}
	}
	if !s.hasPlan(planID) {
		return nil, &rpcErr{code: -32603, message: "Plan does not exist"}
	}
	for i := range s.data.Cases {
		if s.data.Cases[i].ID != caseID {
			continue
		}
		for _, member := range s.data.Cases[i].Plans {
			if member == planID {
				return map[string]any{"plan_id": planID, "case_id": caseID}, nil
			}
		}
		s.data.Cases[i].Plans = append(s.data.Cases[i].Plans, planID)
		return map[string]any{"plan_id": planID, "case_id": caseID}, nil
	}
	return nil, &rpcErr{code: -32603, message: "TestCase matching query does not exist."}
}

// hasPlan reports whether the dataset holds a plan with the ID.
func (s *Stub) hasPlan(id int) bool {
	for _, p := range s.data.Plans {
		if p.ID == id {
			return true
		}
	}
	return false
}
