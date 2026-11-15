---
title: "E16, E17, and clean-text: the write guard, pacing, and text cleaning"
description: The write guard, per-source pacing, and the fixed invisible-character strip.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E16, E17, and clean-text: the write guard, pacing, and text cleaning

This page belongs to [ADR 003](../003-kiwi-gaps.md). E16 covers K7, the update-first-reads guard of `writer.go`. E17
covers K8, the per-call pacing of `client.go`. The `clean-text` converter covers K10, the unescape and strip of
`markdown.go`. E16 and E17 are built by pull requests 28 and 27; `clean-text` is built by pull request 27.

## E16: the write guard

`updateCase` reads the case first and refuses a missing one, so no write goes out for nothing. The engine has no
read before a write. E16 adds one closed guard.

```yaml
- id: kiwi.cases.update
  access: write
  guard:
    read: kiwi.cases.get # new; one read of the same spec
    params: { document_id: "{{document_id}}" } # new; from the write's own args only
    missing: document_not_found # new; closed: collection_not_found, document_not_found, not_found
```

- **Rule.** The guard runs immediately before the write, with params from the write's declared args only. An empty
  or missing guard reply ends the write with the declared not-found, and the stub counts zero writes. The race
  between guard and write stays, as in Go; it is closed by the existing unknown-outcome rule, because the write reply
  must name the guarded object or the outcome is unknown and never retried.
- **L17. A write runs without its guard, or the guard checks the wrong object.** Control: the guard is declared on
  the write and run by the engine, params come from args only, and the target must be a read. Proof:
  `TestGuardMissingSendsNoWrite`, `TestGuardParamsFromArgsOnly`, and `TestGuardTargetMustBeRead`.
- **Cannot do.** Guard a read, declare two guards, guard across specs or hosts, or take a param from a reply.

## E17: pacing

Kiwi calls go one at a time with a 100 millisecond pause after each. The engine has no pacing. E17 adds a fixed
per-source delay.

```yaml
pacing: { min_delay_ms: 100 } # new; on the spec, 0 to 1000
```

- **Rule.** After each request to the source, the next waits until the delay has passed. The delay applies to reads
  and writes equally and is serialized per source, so no burst can slip through parallel calls. A spec may only set
  0 to 1000; the default is 0, which changes nothing.
- **L18. Pacing is evaded by parallel or unspaced calls.** Control: the per-source serialization and the fixed range
  above. Proof: `TestPacingSpacesRequests` with a fake clock, `TestPacingSerializesCalls`, and
  `TestPacingRefusesAbove1000`.
- **Cannot do.** Pace per operation, sleep past 1000 milliseconds, or pace another source.

## clean-text

Kiwi text is unescaped one level of HTML, then stripped of C0 and C1 controls except tab and newline, DEL,
zero-width and bidirectional characters, and the byte order mark. The engine strips controls only. `clean-text` is a
closed converter with the fixed algorithm moved from `clean()`, reviewed once.

```yaml
fields:
  summary: { pointer: /result/summary, convert: clean-text } # new converter name
```

- **Rule.** Unescape exactly once, then strip exactly the fixed set. Tab, newline, and all other characters stay,
  including the non-breaking space. Cleaning runs before mapping, so titles, metadata, and Markdown bodies all see
  clean text.
- **L20. Invisible text reaches the consumer, or cleaning hides evidence.** Control: the fixed set, documented above,
  and parity with the Go output asserted by the moved tests. Proof: `TestCleanTextStripsInvisible` (plants a
  zero-width space and a bidi override and finds neither), `TestCleanTextKeepsTabNewlineAndNbsp`, and
  `TestCleanTextUnescapesOnce` (`&amp;lt;` becomes `&lt;`, never `<`).
- **Cannot do.** A second unescape level, a custom strip set, or cleaning after mapping.
