package kiwi

import (
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The ten Kiwi JSON-RPC methods the adapter may call. The list is closed: a method outside it never
// reaches the network, and no remove, delete, or Auth method is here, because Kiwi's remove with an empty query matches
// every case.
const (
	methodProductFilter        = "Product.filter"
	methodTestPlanFilter       = "TestPlan.filter"
	methodTestCaseFilter       = "TestCase.filter"
	methodTestCaseSortkeys     = "TestCase.sortkeys"
	methodCategoryFilter       = "Category.filter"
	methodPriorityFilter       = "Priority.filter"
	methodTestCaseStatusFilter = "TestCaseStatus.filter"
	methodTestCaseCreate       = "TestCase.create"
	methodTestCaseUpdate       = "TestCase.update"
	methodTestPlanAddCase      = "TestPlan.add_case"
)

// methodAccess is the access level each allowed method needs. Reads need a source at read; writes need a source at write.
var methodAccess = map[string]domain.Access{
	methodProductFilter:        domain.AccessRead,
	methodTestPlanFilter:       domain.AccessRead,
	methodTestCaseFilter:       domain.AccessRead,
	methodTestCaseSortkeys:     domain.AccessRead,
	methodCategoryFilter:       domain.AccessRead,
	methodPriorityFilter:       domain.AccessRead,
	methodTestCaseStatusFilter: domain.AccessRead,
	methodTestCaseCreate:       domain.AccessWrite,
	methodTestCaseUpdate:       domain.AccessWrite,
	methodTestPlanAddCase:      domain.AccessWrite,
}

// permittedMethod reports whether a source at level may call method. A method outside the table is never permitted.
func permittedMethod(method string, level domain.Access) bool {
	need, ok := methodAccess[method]
	return ok && level.Allows(need)
}

// forbiddenKeys are argument names that Kiwi refuses. The adapter builds every key from fixed names in the code, and this
// guard stops a key of these names from reaching the network at any depth.
var forbiddenKeys = map[string]bool{
	"password": true,
	"token":    true,
	"api_key":  true,
	"apikey":   true,
	"secret":   true,
}

// guardArguments returns ErrBadRequest when any map key in params, at any depth, is a forbidden name. A value is never
// checked, so a forbidden word inside a value, such as a case named "token rules", is allowed.
func guardArguments(params any) error {
	switch v := params.(type) {
	case map[string]any:
		for key, value := range v {
			if forbiddenKeys[strings.ToLower(key)] {
				return domain.ErrBadRequest
			}
			if err := guardArguments(value); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			if err := guardArguments(item); err != nil {
				return err
			}
		}
	}
	return nil
}
