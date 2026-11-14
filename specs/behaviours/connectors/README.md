# Connector spec behaviours

Connector specs: a declarative YAML file that adds a source kind, its CLI commands, and its MCP tools, with no Go change.

## Files

- [spec-connectors.feature](spec-connectors.feature): a spec runs through the generic engine, the CLI and MCP switches,
  the catalog pin, the start-up refusals of a bad spec or folder, and the failure of a Markdown converter.
