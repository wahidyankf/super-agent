# retriever-engine

The generic engine of ADR 001. It takes one validated connector spec and one source, and it runs the read and write
operations of that spec against the source. It renders each request from the spec, sets the key in the one header that
the auth type names, sends the request over HTTPS, and maps the reply to the declared fields. Markdown converters are
not served yet, and an operation that needs one fails before any request is sent.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-engine`.
- Nx project `retriever-engine`, with the tags `type:engine`, `lang:go`, and `domain:retriever`.

## Packages

- `engine/` holds the engine. [engine/README.md](./engine/README.md) lists its files and its tests.
- `engine/replay/` holds the replay stub, which answers the read and write examples of a spec, so that tests run
  without a network. [engine/replay/README.md](./engine/replay/README.md) describes it.

## Public surface

- `New(model, source, ports, options...)` checks the source kind, the host suffix, the ports, and the model rules, then
  builds an `Engine`.
- `Read` runs a read operation, and `Write` runs a write operation. `ListCollections`, `CollectionTree`, and `Document`
  run the spec's gateway roles.
- `Ports` supplies the clock, the credentials, the identities, and the network. `AllowPrivateNetwork()` is the only
  option.
- `Code(kind, err)` returns the stable `retriever.<kind>.<reason>` code, and `ExitStatus(err)` returns the exit status.

## What it may import

The production code imports the contract (`retriever-contract`), the spec (`retriever-spec`), and the standard library.
`go.mod` requires the contract and the spec at `v0.4.0`, through `replace` directives to the sibling folders. The
architecture tests in `apps/retriever-cli/tests/architecture` forbid the engine from importing a package that reads files
or runs processes.

## Safeguards

The same safeguards apply to a read and to a write. The table gives each one in brief. The full rules are in the
[connector engine reference](../../docs/reference/connector-engine.md).

| ID    | Safeguard                                                                                                  |
| ----- | ---------------------------------------------------------------------------------------------------------- |
| S2    | Undeclared arguments are refused. A request carries only its declared headers, `Accept`, and the auth key. |
| S3    | A 30 second time limit and a 10 MiB reply cap, which a spec may only lower. At most 20 pages.              |
| S3    | A rate limit is retried at most three times. A `Retry-After` over 60 seconds is not waited out.            |
| S4    | A redirect is never followed. A 3xx reply fails with `redirected`, after one request.                      |
| S5    | An IP literal and `localhost` are refused before any lookup. Only the checked address is dialed.           |
| S5    | Private, loopback, link-local, shared, unspecified, and multicast addresses are refused, unless the        |
|       | composition root passes `AllowPrivateNetwork`, which lifts the first four and never the last two.          |
| S6    | A header value with a control character, or with text that is not UTF-8, is refused.                       |
| S6    | A spec may not set `Authorization`, `Cookie`, `Host`, or `Content-Length`.                                 |
| S7    | A path slot fills one whole, escaped segment. Query values are encoded.                                    |
| S8    | The key is read per request and travels only in the auth header, never in a URL or an error.               |
| S9    | JSON pointers follow RFC 6901. Nesting is capped at 64 levels and tokens at 1,048,576.                     |
| R1    | A transport error names no key. A non-2xx body is never read or echoed.                                    |
| Write | A write is sent once. A failure after the request may have left returns `write_outcome_unknown`, exit 2,   |
|       | and is never retried.                                                                                      |

## Links

- [Connector engine reference](../../docs/reference/connector-engine.md): the read and write steps, the unknown outcome,
  the safeguards in full, the stable codes, and the exit status.
- [engine/README.md](./engine/README.md): the files, the tests, the rules that the code follows, and what is partly met
  or not yet in this module.
- [ADR 001](../../docs/explanation/decisions/001-user-defined-connectors-and-profiles.md): the design this engine
  implements.
