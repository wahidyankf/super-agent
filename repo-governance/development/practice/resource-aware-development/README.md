---
description: >-
  Indexes the detailed modules of the resource-aware development standard: Nx cache eviction, unadmitted tool runs,
  resource tiers, and remote dispatch.
when_to_use: >-
  Use to find the module that states one part of the HIPPO admission rules.
---

# Resource-Aware Development Details

Sub-modules of [Resource-Aware Development](../resource-aware-development.md). Each file states one part of the HIPPO
admission rules.

## Pages

- [Nx cache eviction](./001-external-nx-cache-eviction.md): the bounded consequences of an approved external eviction of
  a local Nx task cache.
- [Unadmitted runs](./002-unadmitted-tool-runs.md): which spellings of `prettier` and `markdownlint-cli2` may run
  without admission, and why every other spelling stays guarded.
- [Resource tiers](./003-resource-tiers.md): the HIPPO resource tiers as reservations, and how to choose one.
- [Remote dispatch](./remote-dispatch.md): when compute may run on another controlled machine, and the lifecycle each
  dispatch follows.
