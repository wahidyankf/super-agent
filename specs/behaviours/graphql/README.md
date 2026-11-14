# GraphQL behaviours

The read-only GraphQL server that `retriever serve` starts on the loopback address.

## Files

- [graphql-serve.feature](graphql-serve.feature): the loopback listener, the read queries, and the refusal of a
  mutation. It runs at the end-to-end level only, because it starts the built binary.
