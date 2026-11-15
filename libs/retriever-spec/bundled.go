package spec

import (
	"strconv"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// The fixed rules of the bundled ID mapping. Each names the broken rule, never a value.
const (
	ruleCatalogIDUserOnly = "is a field that only a bundled spec may name"
	ruleCatalogIDRow      = "names no row of the kind"
	ruleCatalogIDClaimed  = "names a row that another operation already claims"
	ruleCatalogIDHidden   = "serves an operation that no catalog row names"
	ruleCatalogIDParam    = "names no parameter of the catalog row"
	ruleCatalogIDMissing  = "misses a parameter of the catalog row"
	ruleCatalogIDType     = "is not the type of the catalog row"
	ruleCatalogIDRequired = "differs from the catalog row on whether it is required"
)

// checkCatalogIDRefused returns a finding for each catalog_id of a user spec. Only a bundled spec may name one, so
// the user-folder entry refuses the field wherever it appears.
func checkCatalogIDRefused(s *Spec) []Finding {
	var fs []Finding
	for i := range s.Operations {
		if s.Operations[i].CatalogID != "" {
			fs = append(fs, finding("/operations/"+strconv.Itoa(i)+"/catalog_id", ruleCatalogIDUserOnly))
		}
	}
	return fs
}

// CheckBundledMapping returns the findings of the ID mapping of a bundled spec model: each catalog_id names one
// existing row of the same built-in kind, and no two operations claim one row. An operation that no row names takes
// no catalog_id and stays off both surfaces, and the parameters of a claimed operation match the row in names, types,
// and required flags. The validation calls it for every bundled document, so the mapping cannot drift from the
// catalog row that stays the source of the CLI name, the MCP tool name, and the parameters. Each finding names the
// field path and a fixed rule.
func CheckBundledMapping(s *Spec) []Finding {
	catalog := operations.BuiltinCatalog()
	var fs []Finding
	claimed := make(map[domain.OperationID]bool, len(s.Operations))
	for i := range s.Operations {
		op := &s.Operations[i]
		base := "/operations/" + strconv.Itoa(i)
		if op.CatalogID == "" {
			if op.CLI.Enabled {
				fs = append(fs, finding(base+"/cli", ruleCatalogIDHidden))
			}
			if op.MCP.Enabled {
				fs = append(fs, finding(base+"/mcp", ruleCatalogIDHidden))
			}
			continue
		}
		row, ok := catalog.Lookup(domain.OperationID(op.CatalogID))
		if !ok {
			fs = append(fs, finding(base+"/catalog_id", ruleCatalogIDRow))
			continue
		}
		kind, ok := catalog.KindOf(row)
		if !ok || kind.String() != s.Kind {
			fs = append(fs, finding(base+"/catalog_id", ruleCatalogIDRow))
			continue
		}
		if claimed[row.ID] {
			fs = append(fs, finding(base+"/catalog_id", ruleCatalogIDClaimed))
			continue
		}
		claimed[row.ID] = true
		fs = append(fs, checkBundledParams(base, op, row)...)
	}
	return fs
}

// checkBundledParams returns the findings of one claimed operation against its catalog row: every row parameter has
// a spec parameter of the same name, type, and required flag, and every spec parameter names a parameter of the row.
// The names compare exactly, so a port that needs another spelling extends the language first.
func checkBundledParams(base string, op *Operation, row operations.Row) []Finding {
	var fs []Finding
	type placed struct {
		param Param
		index int
	}
	byName := make(map[string]placed, len(op.Params))
	for j, p := range op.Params {
		if _, repeated := byName[p.Name]; !repeated {
			byName[p.Name] = placed{param: p, index: j}
		}
	}
	for _, want := range row.Params {
		got, ok := byName[want.Name]
		if !ok {
			fs = append(fs, finding(base+"/params", ruleCatalogIDMissing))
			continue
		}
		at := base + "/params/" + strconv.Itoa(got.index)
		if got.param.Type != string(want.Kind) {
			fs = append(fs, finding(at+"/type", ruleCatalogIDType))
		}
		if got.param.Required != want.Required {
			fs = append(fs, finding(at+"/required", ruleCatalogIDRequired))
		}
	}
	rowNames := make(map[string]bool, len(row.Params))
	for _, want := range row.Params {
		rowNames[want.Name] = true
	}
	for j, p := range op.Params {
		if !rowNames[p.Name] {
			fs = append(fs, finding(base+"/params/"+strconv.Itoa(j)+"/name", ruleCatalogIDParam))
		}
	}
	return fs
}
