package engine

import (
	"fmt"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// checkEchoValue runs the key echo of a read of one object: the text at the echo pointer must equal the request
// argument exactly, or the read fails with unexpected_response naming the pointer path, never the values. The
// pointer must read text: a missing, null, or non-text value fails the same way.
func checkEchoValue(o spec.Operation, args domain.Args, root any) error {
	echo := o.Response.Echo
	if echo == nil {
		return nil
	}
	raw, found := pointerValue(root, echo.Pointer)
	if !found {
		return echoError(echo.Pointer)
	}
	reply, isText := raw.(string)
	if !isText {
		return echoError(echo.Pointer)
	}
	want, isText := args.Values()[echo.Param].(string)
	if !isText || reply != want {
		return echoError(echo.Pointer)
	}
	return nil
}

// echoError returns the failure of a key echo that the reply does not hold. It names the pointer path and never a
// value, so neither the request key nor the reply key reaches the message. It matches ErrUnexpectedResponse, so its
// code stays unexpected_response on every surface.
func echoError(pointer string) error {
	return fmt.Errorf("echo %s: %w", pointer, domain.ErrUnexpectedResponse)
}
