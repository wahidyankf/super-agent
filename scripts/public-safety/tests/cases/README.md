---
title: Public-safety test cases
description: One file per public-safety test case. Each file defines a run function that the suite calls.
when_to_use: Read this before you add a case, or when you need to know what one case proves.
---

# Public-safety test cases

Each file here is one case. A case is sourced by [`run.sh`](../run.sh), so it defines a `run` function and names its
shell dialect with a directive rather than a shebang. The cases run in the order of their numbers.

- [`010-missing-term-set.sh`](./010-missing-term-set.sh): an absent term set blocks.
- [`020-empty-term-set.sh`](./020-empty-term-set.sh): an empty or comments-only term set blocks.
- [`030-malformed-term-set.sh`](./030-malformed-term-set.sh): short lines, unknown classes or kinds, bad regexes, and
  credentials block.
- [`040-matching-content.sh`](./040-matching-content.sh): literal, regex, and name matches all block.
- [`050-clean-content.sh`](./050-clean-content.sh): clean input passes, including against the tracked shape set.
- [`060-non-disclosing-output.sh`](./060-non-disclosing-output.sh): a blocked run reproduces neither a credential nor a
  term.
- [`070-surface-coverage.sh`](./070-surface-coverage.sh): all seven leaf surfaces accept, block, or reject unknowns.
- [`080-tracked-shape-set.sh`](./080-tracked-shape-set.sh): every tracked shape matches, and ordinary text still passes.
- [`090-surface-dispatch.sh`](./090-surface-dispatch.sh): a missing, empty, or unknown gate surface is a scan error.
- [`100-credential-finding-names-its-input.sh`](./100-credential-finding-names-its-input.sh): a credential finding names
  its input, not a temporary copy.
- [`110-file-list-inputs.sh`](./110-file-list-inputs.sh): listed files and names are screened; a bad list is a scan
  error.
- [`120-large-tree-dispatch.sh`](./120-large-tree-dispatch.sh): a tracked tree larger than the argument limit is still
  screened.
- [`130-hook-environment-isolation.sh`](./130-hook-environment-isolation.sh): a suite started from a Git hook leaves the
  hook's own repository untouched.
- [`140-cidr-network-prefix.sh`](./140-cidr-network-prefix.sh): a CIDR network prefix passes; host forms in every private
  range still block.
- [`150-hostname-trailing-underscore.sh`](./150-hostname-trailing-underscore.sh): an underscore continues an identifier;
  real hostnames still block.
- [`160-typed-commit-message-adapter.sh`](./160-typed-commit-message-adapter.sh): typed hook and pull-request messages
  are screened without publishing a hook path.
- [`170-range-history.sh`](./170-range-history.sh): a range is screened commit by commit; a value added and then deleted
  still blocks.
- [`180-canary-fails-closed.sh`](./180-canary-fails-closed.sh): missing detection, disclosure, or scanner failure
  blocks and cleans up.
- [`190-a-scanner-cache-never-exposes-a-scanner-without-its-stamp.sh`](./190-a-scanner-cache-never-exposes-a-scanner-without-its-stamp.sh):
  a scanner cache publishes a scanner only with its stamp, and refuses an entry with no matching stamp.
- [`200-go-package-qualified-identifiers.sh`](./200-go-package-qualified-identifiers.sh): a Go package-qualified
  identifier passes; host forms in every other context still block.
