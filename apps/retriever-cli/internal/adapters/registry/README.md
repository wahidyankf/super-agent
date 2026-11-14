# internal/adapters/registry

Implements `ports.GatewayResolver`. It builds one gateway for each source ID, the first time that ID is used, and it
reuses the gateway after that.

- `registry.go` holds the resolver and its cache of gateways.
- `registry_test.go` and `capabilities_test.go` test the resolver and the capabilities that each gateway offers. They
  carry the `integration` build tag, so run them with `-tags integration`.
- `writer_test.go` tests the write gateways. It has no build tag, so it runs in the unit pass.
