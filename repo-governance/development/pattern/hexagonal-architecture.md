---
description: >-
  Sets the hexagonal layering for the command-line app and its connectors: a core that decides, ports the core needs,
  adapters that perform effects, and a shell that wires them, with dependencies pointing inward.
when_to_use: >-
  Use when placing new code, adding an adapter, or deciding where a rule, an effect, or a value conversion belongs.
---

# Hexagonal Architecture

The core decides, and adapters act. Dependencies point inward: an adapter depends on the core, and the core never
depends on an adapter.

## Layers

| Layer       | Holds                                                                      | May import       |
| ----------- | -------------------------------------------------------------------------- | ---------------- |
| Core        | Value types, rules, and decisions, including each kind's identity rules    | Standard library |
| Application | Use cases that combine the core with ports                                 | Core             |
| Ports       | Interfaces the core needs from outside, such as a source reader or a clock | Core             |
| Adapters    | Effects: the command line, HTTP, files, and each source kind               | Core and ports   |
| Shell       | Wiring: it builds the adapters and calls the application                   | Everything       |

## Rules

- A core function receives everything it needs as parameters. It never reads the clock, the environment, or the network.
- An adapter converts external data into core types at its edge. Infrastructure error types stop at the adapter.
- The source-kind rules sit in the core, so every adapter applies the same rules.
- A port is named for what the core needs, not for the technology behind it.
- Each adapter has integration tests. Each decision has unit tests.

## Related

- [Best Practices](best-practices.md) describes the functional core and imperative shell style that the core follows.
- [Golang Standards](../quality/stacks/golang-standards.md) applies this layering to Go packages.
