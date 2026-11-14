# registry

Implements `ports.GatewayResolver`. It builds one gateway for each source ID, the first time that ID is used, and it
reuses the gateway after that. The composition root supplies one builder for each source kind.

- `registry.go` holds the resolver and its cache of gateways.
- `registry_test.go`, `capabilities_test.go`, and `writer_test.go` have no build tag, so they run in the unit level.
