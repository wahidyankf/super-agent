# MCP behaviours

The MCP tool server that `retriever mcp` starts over stdio. It offers every tool of the catalog, and only the resources
named by `--tools`.

## Files

- [mcp-server.feature](mcp-server.feature): the tool list, the read-only and destructive hints, the write tools, the
  source list tool, and bounded, redacted tool results. It runs at the unit, integration, and end-to-end levels.
