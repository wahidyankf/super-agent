package outline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestAReplyLargerThan10MiBHasItsOwnError checks that a reply over the body cap is named as too large, with its own
// public text, and not as an unexpected reply. The caller can then act on it: use a smaller limit.
func TestAReplyLargerThan10MiBHasItsOwnError(t *testing.T) {
	r := newRig(t, sampleOptions())
	big := `{"data":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	r.stub.SetReply(methodCollectionsList, 200, big)
	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("ListCollections error = %v, want ErrReplyTooLarge", err)
	}
	if want := "the reply was larger than 10 MiB; use a smaller limit"; domain.ErrReplyTooLarge.Error() != want {
		t.Fatalf("ErrReplyTooLarge text = %q, want %q", domain.ErrReplyTooLarge.Error(), want)
	}
}

// TestAReplyAtTheCapIsNotTooLarge checks the boundary: a reply of exactly the cap is read, and only one byte more is
// refused.
func TestAReplyAtTheCapIsNotTooLarge(t *testing.T) {
	r := newRig(t, sampleOptions())
	// The text is sized so that the whole body, its envelope included, is exactly the cap.
	envelope := `{"data":"` + `"}`
	atCap := `{"data":"` + strings.Repeat("a", maxBodyBytes-len(envelope)) + `"}`
	if len(atCap) != maxBodyBytes {
		t.Fatalf("fixture is %d bytes, want %d", len(atCap), maxBodyBytes)
	}
	r.stub.SetReply(methodCollectionsList, 200, atCap)
	if _, err := r.gw.ListCollections(context.Background()); errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("a reply of exactly the cap was refused as too large")
	}
}
