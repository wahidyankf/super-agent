---
title: "E18 and E10: the JSON-RPC envelope and the error object"
description: The fixed JSON-RPC shape with the id echo check, and the closed error-object mapping.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E18 and E10: the JSON-RPC envelope and the error object

This page belongs to [ADR 003](../003-kiwi-gaps.md). E18 is built by pull request 25 with E10. It covers K9, the id echo
check of `client.go`, and K1, the error table of `errors.go`.

## E18: the envelope

Every Kiwi call is a POST to one path with a body of `jsonrpc`, `method`, named `params`, and an `id`. The engine
cannot write that body today: E1 builds bodies from constants and slots but has no sequential id, and it never checks
a reply id. E18 adds one closed envelope.

```yaml
transport:
  envelope: jsonrpc # new; the only value
operations:
  - id: kiwi.products.list
    rpc_method: Product.filter # new; a constant service name
```

- **Rule.** The body is fixed: `jsonrpc` is always `"2.0"`, `method` is the constant `rpc_method`, `params` is the
  named object as in E1, and `id` counts 1, 2, 3 per gateway. The reply passes the HTTP checks first, then the id
  must equal the request id, then exactly one of `result` and `error` must be present. A mismatch fails loud:
  `unexpected_response` for a read, `write_outcome_unknown` for a write. A failure names the operation and the field
  path, never the body.
- **Limits.** One call per HTTP request. No batch, no notification, no id reuse. `rpc_method` is a constant of at
  most 64 characters; the engine never builds it from an argument or a reply.
- **L19. A reply is accepted for the wrong call.** Control: the per-call echo check above. Proof:
  `TestJsonRpcIdMismatchIsUnexpected`, `TestJsonRpcWriteMismatchIsUnknownOutcome`,
  `TestRpcMethodNeverBuiltFromArgs`, and `TestJsonRpcErrorAndResultIsUnexpected`.
- **Cannot do.** Batch calls, server-sent ids, id reuse across calls, or a method from reply data.

## E10: the error object

Kiwi reports failures inside HTTP 200 as an `error` object with a numeric code. The engine maps HTTP statuses only.
E10 reads the object through a fixed table that moves from `errors.go` and is reviewed once.

```yaml
response:
  error_object: true # new; read the /error member of a 2xx reply
  not_found: document_not_found # new; closed: collection_not_found, document_not_found, not_found
```

- **Rule.** With `error_object`, a 2xx reply carrying `error` maps by code: `-32600`, `-32601`, and `-32602` mean a
  bad request. `-32603` matches the lowercased message against the fixed word table: authentication words mean the key
  is rejected, permission words mean access denied, existence words mean the declared `not_found`, and request words
  mean a bad request. Any other code, and any unmatched message, is an upstream failure for a read and an unknown
  outcome for a write. The message is only compared, never stored or shown. Without `error_object`, an `error`
  member is `unexpected_response`.
- **Why not codes only.** The scenarios pin message-mapped sentinels: "login required" must stay key rejected, and a
  missing plan must stay document not found. A code-only table would change behaviour and lose the consumer the cause.
  The word table is engine-fixed data, not a spec-author pattern, so S1 still holds.
- **L12. An error object is mismapped, or its message leaks.** Control: the fixed table and the compare-only rule
  above. Proof: `TestRpcLoginRequiredIsKeyRejected`, `TestRpcUnknownCodeIsUpstream`,
  `TestRpcErrorMessageNeverShown` (plants a secret in the message and finds it in no error or log),
  `TestRpcErrorNeedsDeclaredFlag`, and `TestRpcNotFoundNamesDeclaredSentinel`.
- **Cannot do.** Match another code, carry message text into an error or a log, or name a `not_found` outside the
  three closed sentinels. A service needing another table needs a new ADR.
