package outlinestub_test

import (
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// testClock is the instant every stub of the unit tests stamps its requests with. A stub without an injected clock
// would read the real one, so the tests give each stub this clock.
var testClock = time.Date(2026, time.November, 14, 9, 30, 0, 0, time.UTC)

// newStub returns an in-process stub whose requests carry testClock. An option that follows sets its own clock.
func newStub(opts ...outlinestub.Option) *outlinestub.Stub {
	return outlinestub.New(append([]outlinestub.Option{outlinestub.WithClock(func() time.Time { return testClock })}, opts...)...)
}
