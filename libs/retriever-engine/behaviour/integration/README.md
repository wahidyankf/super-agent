# integration

The integration bindings of the retriever-engine library. Each file here has the `integration` build tag and crosses a
boundary of the machine only through loopback: a loopback server, a real socket that a test opens on the loopback
address, or a refused dial. No test here needs a real secret, and none reaches a source outside the machine.

Each `*_test.go` file covers the scenarios of one feature file of
[the feature corpus](../../../../specs/libs/retriever-engine/behaviours/), and it takes the feature's name.
