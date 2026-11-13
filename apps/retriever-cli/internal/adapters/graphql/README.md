# internal/adapters/graphql

The read-only GraphQL service that `retriever serve` runs. It listens on the loopback address only. It exposes the
`sources`, `collections`, `collectionTree`, and `document` queries, and it has no mutation. Introspection is off.
A query
over the depth, complexity, or size limit is refused before any source is called.

Source files:

- `schema.graphqls` is the served schema. It is the input of the code generator.
- `generate.go` holds the `go:generate` directive that runs gqlgen.
- `gqlgen.yml` configures gqlgen.
- `generated.go` is the gqlgen output. Do not edit it by hand.
- `models_gen.go` holds the models that gqlgen generates. Do not edit it by hand.
- `schema.resolvers.go` holds the resolver methods of the schema.
- `resolver.go` holds the root resolver and the services it calls.
- `handler.go` builds the HTTP handler.
- `middleware.go` holds the HTTP middleware around the handler.
- `limits.go` holds the depth, complexity, and size limits.
- `errors.go` maps a service error to a GraphQL error.
- `listen.go` binds the loopback listener and serves until shutdown.

Test files:

- `integration_test.go` runs the service end to end. It carries the `integration` build tag.
- `coverage_gaps_test.go` covers the branches that the other tests miss.
- `fakes_test.go` and `service_fakes_test.go` hold the fake services that the tests use.
- `handler_test.go`, `middleware_test.go`, `limits_test.go`, and `listen_test.go` test the HTTP layer.
- `schema_test.go` and `sources_query_test.go` test the schema and the sources query.
- `errors_test.go`, `identity_error_test.go`, `kind_argument_test.go`, and `kind_error_test.go` test the error
  mapping and
  the kind argument.
