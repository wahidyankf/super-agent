# integration

The integration bindings of the retriever-connector-kiwi library. Each file here has the `integration` build tag. A binding
runs the testkit's Kiwi stub behind a real TLS listener on the loopback address. The certificate names the endpoint host and
the loopback address, so the adapter's TLS name check runs against a real handshake. A binding that reads a CA file writes
it into the test's temporary directory. No binding needs a real secret, and none reaches a host outside the machine.

Each `*_test.go` file covers the scenarios of one feature file of
[the feature corpus](../../../../specs/libs/retriever-connector-kiwi/behaviours/), and it takes the feature's name. Shared
server, certificate, and handler-wrapper helpers live in `support_test.go`.
