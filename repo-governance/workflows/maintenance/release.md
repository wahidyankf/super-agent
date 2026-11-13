---
description: >-
  Defines how a GitHub pre-release of the command-line tool is cut: a v0.x.y tag, an alpha notice, a changelog entry,
  checksum-pinned binaries for each platform, and a leak scan of every asset.
when_to_use: >-
  Use when the owner asks for a release, when preparing release notes, or when verifying a published pre-release.
---

# Release

Releases are GitHub pre-releases, versioned `v0.x.y`. A pre-release is not a stable product. Its notes carry the alpha
notice, and it may contain breaking changes, which are named.

## Before Cutting

1. **Owner approval.** The owner approves the version number and the tag. No agent picks either one.
2. **Green main.** `main` passes every gate at the pull-request surface, and every CI check is green.
3. **Changelog.** `CHANGELOG.md` has a section for the version. It names each breaking change, with the old form, the
   new form, and the step that migrates a caller. See the [public contract](../../development/public-contract.md).
4. **Contract checks.** The conformance runner passes for the command-line tool.

## Cutting

1. Create an annotated tag `vX.Y.Z` on the merged commit on `main`.
2. Build the binary for each platform that the release notes name. Build from the tag, with the pinned tools.
3. Write `SHA256SUMS` with one checksum for each binary. Each binary's checksum is recorded in the notes too.
4. Run the leak scan over every release asset, the release notes, and the changelog excerpt. A failed scan stops the
   release.
5. Publish the release as a GitHub pre-release. The notes open with the alpha notice and list the breaking changes.

## Notes

The release notes include:

- the alpha notice, in plain words, that breaking changes can land in any release;
- the list of breaking changes, each with its migration step;
- the list of the platforms and the checksum file; and
- a link to the changelog section for the version.

## Consumers

A consumer pins a release by its version and its checksum. A consumer never follows "latest", because a pre-release can
change its surface. The pinned checksum is verified before the binary runs.

## Automation

Release automation is added in a later pull request. Until then, a release is cut by hand, following these steps in
order. No step is skipped, and a failed step stops the release.

## After Publishing

Report the release to the owner with the tag, the checksum file, and the result of each check. If a release is wrong, a
new patch release replaces it. A published asset is never silently replaced.
