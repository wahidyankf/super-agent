package sourcecontract

import (
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// Reporter receives the first mismatch of one check. A unit test passes its own recorder, so it runs one check, or every
// check in turn, without a *testing.T that stops the test on the first failure. *testing.T satisfies it too.
type Reporter = reporter

// CheckNames returns the names of the checks of the contract for the source that p describes, in the order they run.
func CheckNames(p Profile) []string {
	checks := newSuite(p, StandardDataset()).checks()
	names := make([]string, 0, len(checks))
	for _, c := range checks {
		names = append(names, c.name)
	}
	return names
}

// RunCheck runs the check named name against the gateway that newGateway builds for each Wiring, and reports its first
// mismatch to r. It returns false when the contract for p has no check of that name.
func RunCheck(r Reporter, p Profile, name string, newGateway func(w Wiring) ports.SourceGateway) bool {
	for _, c := range newSuite(p, StandardDataset()).checks() {
		if c.name == name {
			c.run(opener(newGateway), r)
			return true
		}
	}
	return false
}

// PlainValue returns a Record as plain Go values, which is the form the contract compares with its literals.
func PlainValue(rec domain.Record) any {
	return asValue(rec)
}
