---
description: >-
  Defines the HIPPO resource tiers as reservations: three required tiers and one optional last-resort tier, the two
  failure modes of a wrong tier, and the queue rule that follows from them.
when_to_use: >-
  Use when choosing the `--resource-tier` of a HIPPO-guarded command, when a tier looks too low or too high for the work
  it guards, or when considering the optional `very-heavy` tier or swap counting.
---

# Resource Tiers

A resource tier is a reservation, not a limit: HIPPO does not cap the process, so classify by what the work really
needs. `light` is one narrow static check (a linter, a formatter, one validator). `standard` is an ordinary check, one
project's tests, or a writer. `heavy` is a full build, a whole suite, or a quality gate.

A tier that is too low lets real use exceed the reservation: the host reaches pressure and HIPPO sheds ephemeral work. A
tier that is too high, or `heavy` by habit, holds capacity other runs need and starves the queue. Never choose a tier to
be admitted faster.

Pick the smallest tier that fits the work, and move up only on measured need. `hippo history` reports each run's child
CPU and wall time; the build tool or the operating system reports memory.

Schema 3 keeps one FIFO waiter and launches at most once.

## Optional Settings

`very-heavy` is a last-resort tier, with 4 to 6 CPUs and 6144 to 10240 MiB in HIPPO's example. Declare it only in the
machine-specific `hippo.local.json` of a machine that can hold its maximum. A run that names it where the loaded file
does not declare it exits `2` with `hippo.args.invalid`.

`coordination.countSwap` is off by default. On Linux with schema 3, it counts unused swap as memory up to an allowance.
Never enable it on zram-backed swap: zram is carved from RAM, so the same memory would count twice.

Declare either setting only once every consumer on the machine runs a binary that knows it. An older binary refuses any
file that declares them.

## Operation Notes

Run guarded work as `./hippo run --class <ephemeral|transactional|service> --resource-tier <light|standard|heavy>
--disk-path . -- <cmd>`. Exit `124` means a limit stopped the work: `storage-blocked` means free disk first, since
waiting frees none; `capacity-deferred` means retry when the host is quieter. Exit `125` means the guard refused and
only `never-started` receipts apply. v0.13.0 also adds opt-in proportional thresholds (`coordination.memoryPolicy`,
schema 3 only) and an opt-in per-machine swap gate (`coordination.swapUsedMaxMultiple`); an omitted key keeps the legacy
behavior. No production percents, floors, or multiples are set anywhere yet.

## Observations

A command follows the rule when its tier is the one its work needs and the tier is not changed after a look at the
queue. It breaks the rule when a lone linter runs as `heavy`, a full build runs as `light`, a tier is raised without a
measured need, or a tier is edited after `./hippo status` shows a shorter wait.

## Enforcement

**Unenforced by decision**, because the right tier depends on what the work really needs, which no mechanical check
settles. Review checks that each guarded command carries a `--resource-tier`.
