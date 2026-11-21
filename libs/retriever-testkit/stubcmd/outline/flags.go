package outline

import (
	"fmt"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// collectionFlag collects repeated -collection id=name flags in listing order.
type collectionFlag []outlinestub.Collection

// String implements flag.Value.
func (c *collectionFlag) String() string { return fmt.Sprint(*c) }

// Set implements flag.Value. The value splits at its first "=", so a name may hold "=". Both parts must be non-empty.
func (c *collectionFlag) Set(value string) error {
	id, name, found := strings.Cut(value, "=")
	if !found || id == "" || name == "" {
		return fmt.Errorf("-collection wants id=name, got %q", value)
	}
	*c = append(*c, outlinestub.Collection{ID: id, Name: name})
	return nil
}
