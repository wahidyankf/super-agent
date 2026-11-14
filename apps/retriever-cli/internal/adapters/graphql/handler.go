package graphql

import (
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// Deps are the collaborators of the GraphQL server. Service holds the catalog and the gateway resolver. A zero Service
// holds no source. Logger may be nil, in which case nothing is logged.
type Deps struct {
	Service usecases.Service
	Logger  *slog.Logger
}

// Option configures NewHandler.
type Option func(*options)

type options struct {
	listener net.Listener
}

// WithListener sets the listener the handler serves. The served port is read from its address, and the
// Host allowlist accepts only that port. Pass the same listener to Serve.
func WithListener(l net.Listener) Option {
	return func(o *options) { o.listener = l }
}

var (
	errNoListener = errors.New("graphql: WithListener is required")
	errNotTCP     = errors.New("graphql: the listener address must be TCP")
)

// NewHandler returns the HTTP handler of the GraphQL API. Every request passes these guards, in order:
// logging of refusals, the Host allowlist, the Origin rejection, the POST-only rule, the JSON content type,
// the body size cap, and the depth and complexity limits. Only then does gqlgen run the operation.
// Introspection is off: a query that reads the schema is refused by the limits guard.
func NewHandler(deps Deps, opts ...Option) (http.Handler, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.listener == nil {
		return nil, errNoListener
	}
	addr, ok := o.listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errNotTCP
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	schema := NewExecutableSchema(Config{Resolvers: &Resolver{Service: deps.Service}})
	srv := handler.New(schema)
	srv.AddTransport(transport.POST{})
	srv.SetErrorPresenter(newErrorPresenter(logger))
	srv.SetRecoverFunc(recoverPanic(logger))

	guards := []middleware{
		logRefusals(logger),
		requireLoopbackHost(addr.Port),
		rejectBrowserOrigin(),
		requirePOST(),
		requireJSON(),
		limitBody(maxBodyBytes),
		limitQuery(),
	}
	var h http.Handler = srv
	for i := len(guards) - 1; i >= 0; i-- {
		h = guards[i](h)
	}
	return h, nil
}
