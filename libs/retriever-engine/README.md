# retriever-engine

The generic engine of ADR 001. It takes one validated connector spec and one source, and it runs the read operations of
that spec against the source. It renders each request from the spec, sets the key in the one header that the auth type
names, sends the request over HTTPS, and maps the reply to the declared fields. Writes, Markdown converters, and name
resolution are not served yet. Each one fails before any request is sent.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-engine`.
- Nx project `retriever-engine`, with the tags `type:engine`, `lang:go`, and `domain:retriever`.
- `go.mod` requires the contract and the spec at `v0.4.0`, through `replace` directives to the sibling folders. The
  production code imports the contract, the spec, and the standard library.

## Packages

- `engine/` holds the engine. [engine/README.md](./engine/README.md) lists its files.
- `engine/replay/` holds the replay stub, which answers the read examples of a spec. Tests give it to the engine in
  place of the network.

## What a read does

1. `New` checks the source kind, the host suffix of the spec, the ports, and the model rules. A model that the spec
   rules would refuse is refused here too.
2. `Read` binds the arguments with the operation catalog's rules. It refuses a write, a converter, and a resolve before
   any request is built.
3. The request is rendered: the path slots, the query, the headers, and the body. Each value is checked before the
   request leaves.
4. The reply is read up to the size cap, decoded with a depth and a token cap, and mapped to the declared fields.
5. A rate limit is retried after its `Retry-After` wait, at most three times. A paged read follows the spec's paging
   rule, up to the page cap.

## Safeguards

- **S2**: Undeclared arguments are refused. A request carries only the declared headers, `Accept`, and the auth header.
- **S3**: A 30 second time limit and a 10 MiB reply cap, which a spec may only lower. A reply one byte over the cap
  fails.
- **S3**: At most 20 pages. A `Retry-After` over 60 seconds is not waited out. A rate limit is retried at most three
  times.
- **S4**: The client never follows a redirect. Every 3xx reply fails with `redirected`, after one request.
- **S5**: An IP literal and a `localhost` name are refused before any lookup. Each resolved address is checked, and the
  engine dials only the checked address, never the name.
- **S5**: A private, loopback, link-local, shared, unspecified, or multicast address is refused. The composition root
  may lift the private, loopback, link-local, and shared refusal by passing the `AllowPrivateNetwork` option to `New`.
  Unspecified and multicast addresses are always refused.
- **S6**: A header value with a control character, or with text that is not UTF-8, is refused. A spec may not set
  `Authorization`, `Cookie`, `Host`, or `Content-Length`.
- **S7**: A path slot fills one whole segment, which is escaped. An empty value, a dot segment, a slash, a backslash, a
  percent sign, or a control character is refused. Query values are encoded.
- **S8**: The key is read for each request from the credentials port. It travels only in the auth header, never in a
  URL or an error.
- **S9**: Pointers follow RFC 6901. Nesting is capped at 64 levels and tokens at 1,048,576. Control characters leave the
  text. A shape that fails is `unexpected_response`.
- **R1**: A transport error names no key, because the key is replaced by `[REDACTED]`. A non-2xx body is never read or
  echoed.

### Partly met

- S5: the ADR asks for a check of the connected address in the dialer's control hook. That hook needs syscall types,
  which the module layout forbids in the engine's production code. The engine dials the checked IP literal instead, so
  the connected address is the checked address. The result is the same, and the mechanism differs.
- R1: the engine writes no log, because the logger is not part of this pull request. There is no log text to redact
  yet.

## Stable codes

`Code(kind, err)` returns `retriever.<kind>.<reason>`. A refused network returns `retriever.connector.network_refused`.
The reasons are `timed_out`, `redirected`, `reply_too_large`, `unexpected_response`, `invalid_header`, `not_configured`,
`key_rejected`, `access_denied`, `rate_limited`, `retry_after_too_long`, `not_found`, `bad_request`,
`upstream_failure`, `unreachable`, `document_conflict`, `unresolved_name`, `operation_unavailable`,
`unsupported_operation`, `wrong_operation_class`, `wrong_source_kind`, `invalid_endpoint`, `invalid_id`,
`invalid_limit`, `invalid_offset`, `invalid_number`, `invalid_text`, `invalid_name`, `missing_argument`, and
`unknown_argument`. Any other error is `failed`. No command returns these codes yet.

## Exit status

`ExitStatus(err)` returns 124 for a timeout and for a rate limit that the engine did not wait out. It returns 2 for
every other failure, and 0 for no error. The values match the command-line contract in
[retriever-cli-exit-codes](../../docs/reference/retriever-cli-exit-codes.md).

## Not in this module yet

- Writes, name resolution, and the Markdown converters belong to the next pull request of ADR 001. Each one returns
  `unsupported_operation` before any request is sent.
- No production code wires a real clock or a real dialer. The composition root supplies them through `Ports`.
- A cursor read that reaches its page cap returns the pages it has, with no continuation.

## Tests

- The untagged tests use fakes for the clock, the credentials, and the identities. They open no socket and start no
  timer, and the replay stub answers every request.
- The tests tagged `integration` run the real dialer and real TLS against a loopback server. They cover the time limit,
  the redirect refusal, and the dial refusal, with and without the `AllowPrivateNetwork` option.
- The architecture tests in `apps/retriever-cli/tests/architecture` check the import rules. They forbid the engine from
  importing packages that read files or run processes.
