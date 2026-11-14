---
title: Connector engine
description: The as-built read and write steps, the unknown-outcome rules, the safeguards, the stable codes, and the exit status of the retriever-engine module.
category: reference
tags:
  - reference
  - connectors
  - engine
---

# Connector engine

**Status: as built in `libs/retriever-engine`.** The engine runs the read and write operations of a validated connector
spec against one source. The design is in [ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md).
The package layout and the tests are in [engine/README.md](../../libs/retriever-engine/engine/README.md).

## What a read does

1. `New` checks the source kind, the host suffix of the spec, the ports, and the model rules. A model that the spec rules
   would refuse is refused here too.
2. `Read` binds the arguments with the operation catalog's rules. It refuses a write, a converter, and a resolve before
   any request is built.
3. The request is rendered: the path slots, the query, the headers, and the body. Each value is checked before the
   request leaves.
4. The reply is read up to the size cap, decoded with a depth and a token cap, and mapped to the declared fields.
5. A rate limit is retried after its `Retry-After` wait, at most three times. A paged read follows the spec's paging
   rule, up to the page cap.

## What a write does

1. `Write` binds the arguments with the catalog's rules. It refuses a read, an unknown operation, a converter, and a
   paging rule before any request is built. A write takes one request, so a paging rule would do nothing.
2. Each parameter with a `resolve` declaration is resolved first. The engine runs the named read once, through the same
   path as a read. The read gets the caller's arguments that it declares, and the name under the parameter of the
   field's own name when it declares one. The read must be a list, unpaged, with the field and an `id` field. An entry
   matches when its field equals the name exactly. Exactly one match gives its `id`. Zero matches or two matches fail
   with `unresolved_name`, and no write is sent.
3. The write is rendered and sent once. A body slot takes the resolved ID as a JSON value, so a value never splices into
   text.
4. A 429 reply is waited out and sent again, at most three times, as a read is. The Go connectors do the same, because a
   429 says that the source refused the request before it acted.
5. The reply maps to the declared fields. A reply that reads as a success but holds no mappable field is an unknown
   outcome.

## The unknown outcome

A write whose request may have left has an unknown outcome. It returns `write_outcome_unknown` with exit status 2, and it
is never sent again. The engine decides "may have left" as the Go connectors do:

- A connection that never opened is known. A refused dial, a failed dial, and a TLS handshake that fails all send no
  request, so they keep their own errors, such as `network_refused` or `unreachable`.
- Once a connection exists, any failure is unknown. That covers a timeout, a broken connection, and a cut reply.
- A 5xx reply, a 2xx reply that does not decode, a 2xx reply with no mappable field, and a body over the cap are
  unknown.
- A 4xx reply and a 3xx reply keep their own errors, and each is read as a final answer about the request.
- An injected transport stands for the network, so its failure after the request is an unknown outcome.

The engine sends no idempotency key. The spec model has no field for one, so the engine does not invent one.

## Safeguards in full

The same safeguards apply to a read and to a write. Each item below holds for both, unless it says otherwise.

- **S2**: Undeclared arguments are refused. A request carries only the declared headers, `Accept`, and the auth header.
- **S3**: A 30 second time limit and a 10 MiB reply cap, which a spec may only lower. A reply one byte over the cap fails.
- **S3**: At most 20 pages. A `Retry-After` over 60 seconds is not waited out. A rate limit is retried at most three
  times.
- **S4**: The client never follows a redirect. Every 3xx reply fails with `redirected`, after one request.
- **S5**: An IP literal and a `localhost` name are refused before any lookup. Each resolved address is checked, and the
  engine dials only the checked address, never the name.
- **S5**: A private, loopback, link-local, shared, unspecified, or multicast address is refused. The composition root may
  lift the private, loopback, link-local, and shared refusal by passing the `AllowPrivateNetwork` option to `New`.
  Unspecified and multicast addresses are always refused.
- **S6**: A header value with a control character, or with text that is not UTF-8, is refused. A spec may not set
  `Authorization`, `Cookie`, `Host`, or `Content-Length`.
- **S7**: A path slot fills one whole segment, which is escaped. An empty value, a dot segment, a slash, a backslash, a
  percent sign, or a control character is refused. Query values are encoded.
- **S8**: The key is read for each request from the credentials port. It travels only in the auth header, never in a URL
  or an error.
- **S9**: Pointers follow RFC 6901. Nesting is capped at 64 levels and tokens at 1,048,576. Control characters leave the
  text. A shape that fails is `unexpected_response`.
- **R1**: A transport error names no key, because the key is replaced by `[REDACTED]`. A non-2xx body is never read or
  echoed.

## Stable codes

`Code(kind, err)` returns `retriever.<kind>.<reason>`. A refused network returns `retriever.connector.network_refused`.
The reasons are `timed_out`, `redirected`, `reply_too_large`, `unexpected_response`, `invalid_header`, `not_configured`,
`key_rejected`, `access_denied`, `rate_limited`, `retry_after_too_long`, `not_found`, `bad_request`,
`upstream_failure`, `unreachable`, `document_conflict`, `unresolved_name`, `operation_unavailable`,
`unsupported_operation`, `wrong_operation_class`, `wrong_source_kind`, `invalid_endpoint`, `invalid_id`,
`invalid_limit`, `invalid_offset`, `invalid_number`, `invalid_text`, `invalid_name`, `missing_argument`,
`unknown_argument`, and `write_outcome_unknown`. Any other error is `failed`. No command returns these codes yet.

`write_outcome_unknown` is the code of `domain.ErrWriteOutcomeUnknown`. It is the same code that the Go connectors give,
so a caller sees one code on every surface. The CLI reads the same sentinel, and it exits 2 for it.

## Exit status

`ExitStatus(err)` returns 124 for a timeout and for a rate limit that the engine did not wait out. It returns 2 for every
other failure, and 0 for no error. The values match the command-line contract in
[retriever exit codes](./retriever-cli-exit-codes.md).
