# MCP behaviours

The MCP tool server that `retriever mcp` starts over stdio. It offers tools up to the access ceiling named by
`--max-access`, and only the resources named by `--tools`.

## Files

- [mcp-server.feature](mcp-server.feature): the tool list at each ceiling, the read-only and destructive hints, the
  refusal of a write above the ceiling, and bounded, redacted tool results. It runs at the end-to-end level only.
