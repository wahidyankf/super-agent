---
description: >-
  Sets when a dependency may be added, updated, or pinned, and the checks a change must pass before a new version lands.
when_to_use: >-
  Use when adding a dependency, bumping a version, responding to a vulnerability advisory, or changing a tool pin.
---

# Dependency Bump Policy

A dependency is a liability that the repository must carry. Add one only when it earns its place.

## Adding

- Add a dependency only when the standard library and the existing dependencies cannot do the job. The pull request says
  why.
- Prefer a small, maintained package with a licence that the repository accepts.
- Commit the lock file with the change, so that the exact version is recorded.

## Updating

- Update one dependency per change, unless the change is one coordinated release of a group.
- Read the release notes for breaking changes. Run the tests at each level before merge.
- An update that changes behaviour follows the [test-driven development](test-driven-development.md) cycle. Show that a
  test of the behaviour that matters passes on the new version.

## Security Advisories

A vulnerable dependency moves to the first safe version. The fix follows the same checks as any other change, and it
merges as soon as those checks pass.

## Pinned Tools

- HIPPO and RHINO are pinned by checksum in their lock files. A pin changes only in a change that names the new release,
  verifies its checksum, and runs the gates.
- Go tools are pinned as Go tool dependencies, as the [Golang standard](../quality/stacks/golang-standards.md) states.

## Removal

A change that leaves a dependency unused removes it in the same change.
