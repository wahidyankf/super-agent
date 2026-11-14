# Spec stub

The loopback HTTPS form of the generic stub for connector specs. `Start` serves the canned exchanges of one validated
spec over a real loopback listener, and each reply comes from the replay stub of the engine. It is built only under the
`integration` tag, because it opens a socket.

The package sits in the app's test tree, not in `libs/retriever-testkit`. The layering rules let the testkit import only
the contract, and this stub needs the engine and the spec module. The composition-root integration tests in
`cmd/retriever` and the integration level of the behaviour suite use it.
