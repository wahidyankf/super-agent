# internal/core

The core of the tool. It holds the domain types, the port interfaces, and the use cases that read, export, write, and
administer sources. The core imports no adapter, so every service and terminal stays behind a port.

- [`domain/`](./domain/README.md) holds the plain types that the use cases share.
- [`ports/`](./ports/README.md) declares the interfaces that the adapters implement.
- [`sources/`](./sources/README.md) holds the use cases, and [`operations/`](./sources/operations/README.md) holds the
  catalog of operations that they call.
