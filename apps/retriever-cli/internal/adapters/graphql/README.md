# internal/adapters/graphql

The read-only GraphQL service that `retriever serve` runs. It listens on the loopback address only. It exposes the
`sources`, `collections`, `collectionTree`, and `document` queries, and it has no mutation. Introspection is off.
A query over the depth, complexity, or size limit is refused before any source is called.

Production files:

- `schema.graphqls` is the served schema. It is the input of the code generator.
- `generate.go` holds the `go:generate` directive that runs gqlgen.
- `gqlgen.yml` configures gqlgen.
- `generated.go` is the gqlgen output. Do not edit it by hand.
- `models_gen.go` holds the models that gqlgen generates. Do not edit it by hand.
- `schema.resolvers.go` holds the resolver methods of the schema.
- `resolver.go` holds the root resolver and the services it calls.
- `handler.go` builds the HTTP handler, and `middleware.go` holds the HTTP middleware around it.
- `limits.go` holds the depth, complexity, and size limits.
- `errors.go` maps a service error to a GraphQL error.
- `listen.go` binds the loopback listener and serves until shutdown.

Tests sit beside the code. `integration_test.go` and `schema_integration_test.go` carry the `integration` build tag. The
untagged tests use the fake services in `fakes_test.go` and `service_fakes_test.go`.
