# internal/adapters/in/mcp

The Model Context Protocol server. It exposes the source use cases as tools. Standard output carries only the protocol,
and each tool call writes one log line to standard error. The tools are limited by `--max-access` and `--tools`, and
the `list_sources` tool is always offered.

Source files:

- `mcp.go` runs one protocol session over standard input and standard output.
- `generated.go` holds the generated protocol types. Do not edit it by hand.
- `tools.go` defines the tools and their input schemas.
- `fit.go` fits a result to the size that a client accepts.
- `page.go` pages a list result.

Test files:

- `tools_test.go` and `profile_test.go` test the tool set and the access profile.
- `input_keys_test.go` and `input_names_test.go` test the input keys and names.
- `source_arg_test.go` tests the source argument.
- `kind_sources_test.go` and `kind_error_test.go` test the source kinds and their errors.
- `identity_error_test.go` and `public_errors_test.go` test the identity errors and the public error texts.
- `fit_test.go` and `fit_cap_test.go` test the size fitting.
- `page_test.go` tests the paging.
- `dry_run_text_test.go` tests the dry-run text.
- `serve_test.go` tests the server session.
- `stage_b_test.go` and `stage_c4_test.go` hold the tests of two groups of tools. Their names carry the group label.
- `fixtures_test.go` holds the fixtures that the tests share.
