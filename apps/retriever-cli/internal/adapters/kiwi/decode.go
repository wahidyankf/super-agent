package kiwi

import (
	"encoding/json"
	"strconv"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The reply shapes of the Kiwi methods the adapter reads. They are the one place that names Kiwi's result fields, so a
// field that the first live read shows differently is changed here and in the stub's fixtures, and nowhere else. Unknown
// fields are ignored. A value that is not a positive number in the ID range is refused by validNumber.

// wireProduct is one row of Product.filter.
type wireProduct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// wirePlan is one row of TestPlan.filter. Parent is null for a root plan.
type wirePlan struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Product int    `json:"product"`
	Parent  *int   `json:"parent"`
	Text    string `json:"text"`
}

// wireCase is one row of TestCase.filter. Category, Priority, and CaseStatus are the IDs of the lookup tables.
type wireCase struct {
	ID         int    `json:"id"`
	Summary    string `json:"summary"`
	Text       string `json:"text"`
	Category   int    `json:"category"`
	Priority   int    `json:"priority"`
	CaseStatus int    `json:"case_status"`
	Setup      string `json:"setup"`
	Action     string `json:"action"`
	Effect     string `json:"effect"`
	Breakdown  string `json:"breakdown"`
}

// The TestCase.sortkeys result is not a list of rows. The live server returns one object that maps each case ID of the plan,
// as a string key, to the sort key number of the case in that plan, and an empty object for a plan without cases. See
// decodeSortkeys.

// wireCategory is one row of Category.filter.
type wireCategory struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Product int    `json:"product"`
}

// wirePriority is one row of Priority.filter. Its name field is value, such as "P1".
type wirePriority struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// wireStatus is one row of TestCaseStatus.filter.
type wireStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// wireWrite is the object a TestCase.create or TestCase.update reply returns.
type wireWrite struct {
	ID      int    `json:"id"`
	Summary string `json:"summary"`
}

// decodeList reads a JSON array of rows. Any other shape is an unexpected response.
func decodeList[T any](raw json.RawMessage) ([]T, error) {
	var rows []T
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, domain.ErrUnexpectedResponse
	}
	return rows, nil
}

// decodeOne reads one JSON object. Any other shape is an unexpected response.
func decodeOne[T any](raw json.RawMessage) (T, error) {
	var row T
	if err := json.Unmarshal(raw, &row); err != nil {
		return row, domain.ErrUnexpectedResponse
	}
	return row, nil
}

// decodeSortkeys reads the TestCase.sortkeys result: an object that maps case ID text to a sort key number. Any other shape
// is an unexpected response, as is a key that is not a whole number. A key outside the ID range is kept, because the
// listed cases decide which IDs count and each of them is checked on its own.
func decodeSortkeys(raw json.RawMessage) (map[int]int, error) {
	var byCase map[string]int
	if err := json.Unmarshal(raw, &byCase); err != nil || byCase == nil {
		return nil, domain.ErrUnexpectedResponse
	}
	out := make(map[int]int, len(byCase))
	for key, sortkey := range byCase {
		n, err := strconv.Atoi(key)
		if err != nil {
			return nil, domain.ErrUnexpectedResponse
		}
		out[n] = sortkey
	}
	return out, nil
}

// field returns one named field of a Record.
func field(name string, value any) domain.Field {
	return domain.Field{Name: name, Value: value}
}

// validNumber reports whether n can be an ID number: positive, and at most nine digits, the same bound as the ID form.
func validNumber(n int) bool {
	return n > 0 && n <= 999999999
}
