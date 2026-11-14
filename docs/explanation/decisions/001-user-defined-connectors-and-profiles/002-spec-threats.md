---
title: Threats from specs
description: Threats from a connector spec as untrusted input, with the control and the proving scenario for each.
category: explanation
tags:
  - explanation
  - security
  - threat-model
  - planned
---

# Threats from specs

A spec is untrusted input (boundary B2). These threats follow the [scope](./001-threat-model.md) of
[ADR 001](../001-user-defined-connectors-and-profiles.md). The limits named here are those of the built-in connectors
today. A spec may lower a limit and never raise it.

**S1. Code in a template.**

- Control: a template holds only `{name}` and `"{{name}}"` slots for declared parameters. It has no function, loop,
  condition, or include. A slot fills a whole JSON value or a whole path segment, so a value never splices into text.
- Proof: a spec whose body holds a function call fails `connector validate` with `retriever.connector.spec_invalid`.
  A parameter value that looks like a slot stays literal text in the request.

**S2. Reads beyond declared names.**

- Control: the schema has no file, include, or expansion field. A spec names only the credential and identity
  variables, which must match the catalog patterns.
- Proof: a spec with `${HOME}` in its endpoint rule fails validation. An undeclared variable set in the environment
  never appears in a request.

**S3. Large or slow replies.**

- Control: a 30 second timeout, a 10 MiB body, 20 pages, and a `Retry-After` wait of at most 60 seconds. A write that
  hits a limit reports an unknown outcome and is not retried.
- Proof: a reply one byte over the cap fails with the reply-too-large reason. A spec that asks for 100 pages gets only
  the first twenty.

**S4. Redirect to another host.**

- Control: the client follows no redirect. A 3xx reply is an error, and the key goes to the catalog host only.
- Proof: a stub that answers 302 to another host receives no second request.

**S5. SSRF to private ranges.**

- Control: the dialer checks the address it connects to, so a changed DNS answer cannot bypass the check. Only the
  catalog source may set `allow_private_network: true`, never a spec. IP literals and `localhost` stay refused.
- Proof: a resolver that returns a private address fails with `retriever.connector.network_refused`. The same run with
  the catalog setting connects.

**S6. Header injection.**

- Control: headers come from spec constants and the declared authentication. A parameter may fill a declared header
  slot, and a value with a control character is refused. `Authorization`, `Cookie`, `Host`, and `Content-Length` are
  reserved.
- Proof: a value with a carriage return and a line feed is refused before any request. A spec that sets `Authorization`
  fails validation.

**S7. Path and query injection.**

- Control: the engine percent-encodes a path value as one segment and refuses `..`, a slash, and an encoded slash. It
  encodes query values itself.
- Proof: the value `../admin` fails as an invalid argument, and the stub sees no request.

**S8. Key sent to the wrong place.**

- Control: the key goes in the authentication header of the declared type, to the catalog host only. It never enters a
  URL, a query, or a body.
- Proof: a spec that puts the key in a query slot fails validation. A stub that records every byte finds the key only in
  the declared header.

**S9. Hostile reply shape.**

- Control: mapping reads JSON pointers only and caps depth, node count, and output size. Control characters leave the
  output.
- Proof: a reply nested past the depth cap fails as an unexpected response, and nothing panics.

**S10. Spec as a parse bomb.**

- Control: the loader reads a regular file of at most 256 KiB, one YAML document, with no alias. A kind cannot repeat or
  shadow a built-in kind or a command word.
- Proof: a spec with an alias is refused, and so is a second spec with kind `outline`.

**S11. A spec that lies about access.**

- Control: the engine cannot know. Validation refuses a method that no label allows, such as DELETE with `read`.
  `connector validate` prints each operation's label, method, and path for review. retriever does not stop a write
  that the token allows. The token's own access on the source is the guard.
- Proof: a read that uses DELETE fails validation, and the review lists every operation.
