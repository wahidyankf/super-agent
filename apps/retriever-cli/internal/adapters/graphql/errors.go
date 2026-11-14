package graphql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// internalMessage is the only text returned for an error outside the typed set. The raw text of such an
// error can carry upstream detail, so it is logged by type only and never sent.
const internalMessage = "internal error"

// publicMessage returns the fixed text of a typed error. ok is false for any other error, which then
// gets internalMessage. A missing key names the variable and never a value.
func publicMessage(err error) (string, bool) {
	var missing domain.MissingKeyError
	if errors.As(err, &missing) {
		return missing.Error(), true
	}
	var invalidIdentity domain.InvalidIdentityError
	if errors.As(err, &invalidIdentity) {
		return invalidIdentity.Error(), true
	}
	var unknown domain.UnknownSourceError
	if errors.As(err, &unknown) {
		return fmt.Sprintf("unknown source %q", unknown.ID), true
	}
	var ambiguous domain.AmbiguousSourceError
	if errors.As(err, &ambiguous) {
		return fmt.Sprintf("more than one %s source exists (%s); pass the source argument", ambiguous.Kind, strings.Join(ambiguous.IDs, ", ")), true
	}
	var noSource domain.NoSourceOfKindError
	if errors.As(err, &noSource) {
		return noSource.Error(), true
	}
	if errors.Is(err, domain.ErrNoSourceOfKind) {
		// A raw sentinel names no kind, so it gets the default kind's text.
		return "no outline source is configured", true
	}
	var kindErr domain.KindError
	if errors.As(err, &kindErr) {
		if text, ok := kindErr.PublicText(); ok {
			return text, true
		}
	}
	for _, typed := range domain.PublicErrors() {
		if errors.Is(err, typed) {
			return typed.Error(), true
		}
	}
	return internalMessage, false
}

// newErrorPresenter returns the presenter for resolver errors. A GraphQL error that gqlgen built keeps its
// message, because it names only the query. Any other error is mapped to its typed text, and an unknown one
// is logged by type and sent as internalMessage.
func newErrorPresenter(logger *slog.Logger) func(context.Context, error) *gqlerror.Error {
	return func(ctx context.Context, err error) *gqlerror.Error {
		// gqlgen wraps every resolver error in a gqlerror whose Err is the resolver's error. The wrapper's
		// Message is that error's raw text, so the wrapped error is mapped, never the wrapper's message.
		inner := err
		var gqlErr *gqlerror.Error
		if errors.As(err, &gqlErr) {
			if gqlErr.Err == nil {
				return gqlErr
			}
			inner = gqlErr.Err
		}
		msg, ok := publicMessage(inner)
		if !ok {
			logger.Warn("graphql resolver failed", "error_type", fmt.Sprintf("%T", inner))
		}
		return &gqlerror.Error{Message: msg, Path: gql.GetPath(ctx)}
	}
}

// recoverPanic turns a resolver panic into an error. The panic value is logged by type only, because it
// can carry anything, the key included.
func recoverPanic(logger *slog.Logger) gql.RecoverFunc {
	return func(ctx context.Context, panicValue any) error {
		logger.Error("graphql resolver panicked", "panic_type", fmt.Sprintf("%T", panicValue))
		return errors.New("resolver panicked")
	}
}

// writeErrors writes a refusal as standard GraphQL error JSON, {"errors":[{"message":...}]}, with the given
// status. Only fixed messages reach it, so no raw text can leak. strconv.Quote escapes the quotes and
// backslashes of a message, which is all a fixed message can hold, and it cannot fail, so the body is built
// from constant parts with no error branch.
func writeErrors(w http.ResponseWriter, status int, message string) {
	body := `{"errors":[{"message":` + strconv.Quote(message) + `}]}`
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		// The client has gone, so there is no one left to report the failure to.
		return
	}
}
