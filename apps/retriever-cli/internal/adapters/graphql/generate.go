package graphql

// Regenerate generated.go, models_gen.go, and the resolver stubs from schema.graphqls.
// The gqlgen version is pinned by the tool directive in go.mod, so the command needs no network
// install step. Run from apps/retriever-cli: go generate ./internal/adapters/graphql/
//go:generate go tool gqlgen generate --config gqlgen.yml
