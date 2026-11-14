package behaviour

import "github.com/wahidyankf/super-agent/libs/retriever-contract/domain"

// registerKind sets the kinds that the in-process levels accept: the three built-in kinds and one more kind, named name.
// The composition root registers the built-in kinds only, so this step stands in for a registration at the levels that
// run in process. The end-to-end level runs the built binary, so a scenario that uses this step exempts that level.
func (w *world) registerKind(name string) error {
	set, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: name},
	)
	if err != nil {
		return err
	}
	w.kinds = set
	return nil
}
