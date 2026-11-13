---
description: >-
  Defines how super-agent relates to the other repositories of its group: its membership, what it shares, and what it
  never shares.
when_to_use: >-
  Use when a change touches another repository, when proposing a shared artifact, or when a task asks to link or
  synchronise repositories.
---

# Related Repositories

super-agent is a member of the ose-projects group of repositories. Membership means shared conventions and shared tools.
It does not make super-agent a parent, a container, or a copy of any other repository.

## What It Shares

- **Pinned tools.** HIPPO and RHINO are used as checksum-pinned release binaries. The pins are recorded in lock files in
  this repository, and nothing reads another repository's checkout.
- **Conventions, by explicit copy.** A shared convention reaches this repository only by an explicit copy. The copy
  records the source it came from, so a later reader can compare the two.

## What It Never Shares

- **Parent or container.** No other repository contains this one, and this one contains no other repository.
- **Sync group.** A change in one repository is never mirrored into another automatically.
- **Release train.** Releases are this repository's own. No other repository's release moves a version here.
- **Private names.** This public repository names no private repository, host, or group member that is not public.

## Coordination

Work that spans repositories is split into one change per repository. Each change has its own pull request in the
repository that owns it, and each one names its owner. A change to a shared convention starts in the repository that
owns the convention, and this repository then adopts it by copy.

The convention for who owns a rule, and how a rule reaches each binding, is in [Rules
Propagation](../../workflows/quality/rules-propagation.md).
