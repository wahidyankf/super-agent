---
title: Repin HIPPO and RHINO
description: How to move the HIPPO and RHINO lock-file pins to a new release, and how the wrappers verify them.
category: how-to
tags:
  - how-to
  - pins
  - supply-chain
---

# Repin HIPPO and RHINO

HIPPO and RHINO are pinned in two lock files at the repository root.

- `hippo.lock` pins `v0.13.4`, with the platform keys `darwin-amd64`, `darwin-arm64`, `linux-amd64`, and
  `linux-arm64`.
- `rhino.lock` pins `v0.17.0`, with the platform keys `aarch64-apple-darwin`, `x86_64-apple-darwin`,
  `aarch64-unknown-linux-gnu`, and `x86_64-unknown-linux-gnu`.

Each lock file holds the release `version`, the source `commit` that the release was built from, and one SHA-256
digest for each platform archive.

## What the wrappers check

The `./hippo` and `./rhino` wrappers run these checks on every run:

1. Read the lock file. Refuse to run if a value is missing, duplicated, or malformed.
2. Download the archive for this platform from the GitHub release.
3. Compare the archive's SHA-256 digest with the lock. On a mismatch, exit `125` and run nothing.
4. Extract the binary, then check that it reports the pinned version and commit. On a mismatch, exit `125`.

## Choose the release

Pick a tag that GitHub does not mark **Pre-release**. A pre-release is a candidate, not a build to pin.

Find the commit that the tag points to. For HIPPO:

```sh
gh api repos/wahidyankf/hippo/commits/<tag> --jq .sha
```

For RHINO, replace `hippo` with `rhino` in the path.

## Copy the digests

Download the release's `checksums.txt`. Each line holds a digest and an archive name.

- HIPPO archives are named `hippo_<version>_<os>_<arch>.tar.gz`, for example `hippo_v0.13.4_darwin_arm64.tar.gz`.
- RHINO archives are named `rhino-<target>.tar.gz`, for example `rhino-aarch64-apple-darwin.tar.gz`.

Edit the lock file. Set `version` and `commit`. Then copy each digest into the key for its platform. For example,
the `darwin-arm64` key takes the digest of the `darwin_arm64` archive.

## Verify

Run both version commands on this machine. Each one downloads the archive for this platform and checks it.

```sh
./hippo version --json
./rhino version --json
```

The wrappers check only the platform they run on. Compare the other platforms' digests with `checksums.txt` by hand,
or run the wrappers on those machines.

Run the pull-request gates. See [Run the quality gates](./run-the-quality-gates.md).

## Land the change

Commit the lock-file change on a branch in a worktree, and open a pull request. Do not push the change directly to
`main`. See [Deliver a change with a pull request](./deliver-a-change-with-a-pull-request.md).
