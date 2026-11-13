---
name: cutting-releases
description: >-
  Guides the judgement around publishing a version: whether a revision is ready, when a version number is already spent,
  what makes documentation true to a build, and how to verify what consumers will receive.
when_to_use: >-
  Use when preparing or running a release of a versioned artifact consumers pin by version and digest, or when a
  published release turns out to be wrong.
compatibility: Requires permission to tag and publish, and read access to the remote default branch.
---

# Cutting Releases

[Release Cut](../../../repo-governance/workflows/maintenance/release.md) owns the sequence, its refusals, and the
build-location decision. [Public Contract](../../../repo-governance/development/public-contract.md)
decides which version a change requires. This skill covers the judgement those leave open.

## Ready Belongs to the Revision

A release describes a revision everyone can reach: already on the remote default branch, through the repository's
integration path. A local commit that passes every gate is not ready, and neither is a checkout carrying untracked files
that never reached review. When any part of that is uncertain, stop before building. A release cannot be taken back, so
doubt resolves toward not cutting.

## A Taken Version Is Spent

A version is spent once any tag with its name exists, locally or remotely, including one left by an abandoned or partial
earlier attempt. Reasoning that nobody has fetched it yet is exactly how a pin breaks silently. Choose the next version
and move on; version numbers are cheap and trust is not.

## Documentation Is Part of What Ships

Write the changelog entry from the changes since the previous release, read from history, not from memory of what the
work was meant to do. Then read the readme and documentation against what is actually being built: a command renamed, a
flag removed, or a default changed makes the documentation false on the day of release.

## Verify What Consumers Receive

A digest recorded is not yet a digest verified. After publishing, look at the release as a consumer would:

1. one artifact exists for every supported platform, and nothing extra;
2. the digest file covers every artifact;
3. an artifact fetched from the published location matches its recorded digest; and
4. where an artifact reports its own version, it names the released tag and revision.

The workflow requires this before announcing under native runners. After a local build it costs little, and it is the
only check that inspects what was published rather than what was built.

## When a Published Release Is Wrong

The tempting repairs are the damaging ones. Moving the tag, re-uploading an artifact, editing the digest file by hand,
or relaxing a consumer's digest check each makes a small problem invisible and turns every existing pin false. Record
the defect, cut the next version, and treat a consumer that refuses a mismatched digest as working correctly.

A run that fails while publishing ends partial: its tag stays, and the fix is still the next version.

## Services Add Their Own Rules

Promoting a release of a running service adds candidate, migration, and traffic decisions, which this skill does not
cover.
