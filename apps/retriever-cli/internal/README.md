# internal

The private packages of the module. Go does not let another module import them.

- [`adapters/`](./adapters/README.md) connects the core to the services, the terminal, the files, and the protocols.
- [`core/`](./core/README.md) holds the domain types, the port interfaces, and the use cases. It imports no adapter.
