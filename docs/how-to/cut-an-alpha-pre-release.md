---
title: Cut an alpha pre-release
description: Manual steps to cut a v0.x pre-release: changelog, tag, archives, checksums, leak scan, and GitHub pre-release.
category: how-to
tags:
  - how-to
  - release
  - alpha
---

# Cut an alpha pre-release

This guide cuts a GitHub pre-release of the command-line tool in `apps/retriever-cli`. It follows the
[Release](../../repo-governance/workflows/maintenance/release.md) workflow, step by step. The repository has no release
automation yet. If automation is added later, it replaces this guide.

## 1. Check the state

- The repository owner has approved the version number and the tag.
- Every change for the release is merged into `main` through a pull request.
- The pull-request gates pass on `main`. See [Run the quality gates](./run-the-quality-gates.md).
- No tag with the version name exists, locally or on `origin`. A version that has a tag is spent, so choose the next
  one instead.

## 2. Update the changelog

In [CHANGELOG.md](../../CHANGELOG.md), move the entries under `Unreleased` into a heading for the version, such as
`## v0.1.0`. Do not add a date to the heading. The GitHub release records the date.

List every breaking change under its release with a **Breaking** label. A reader must be able to see every break
before they upgrade.

Merge the changelog change through a pull request, as in [Deliver a change with a pull
request](./deliver-a-change-with-a-pull-request.md).

## 3. Tag the release commit

Tag the merged commit on `main` with an annotated tag, and push the tag. The archives are built from this tag in step 4,
so the tag comes first.

```sh
git tag -a v0.1.0 -m "v0.1.0" <commit>
git push origin v0.1.0
```

## 4. Build the archives

The build step clones the tag into a temporary directory and builds from that clone, never from the working tree. It
uses the Go release that `go.mod` names, with `CGO_ENABLED=0` and `-trimpath`. The output directory must be empty, and it
must be outside the repository.

```sh
RELEASE_VERSION=v0.1.0 RELEASE_OUT=<empty-directory> ./hippo run --class ephemeral --resource-tier heavy --disk-path . \
  -- npx nx run retriever-cli:build:release
```

The step writes one archive for each platform, and `SHA256SUMS`:

- `retriever_v0.1.0_darwin_amd64.tar.gz`
- `retriever_v0.1.0_darwin_arm64.tar.gz`
- `retriever_v0.1.0_linux_amd64.tar.gz`
- `retriever_v0.1.0_linux_arm64.tar.gz`

Each archive holds one file, `retriever`, with mode `0755`, owned by root. The archive carries the tagged commit's time,
so a second build of the same tag on the same host reproduces the same bytes. The step also runs the binary for the host
platform, and it refuses the build unless `--version` prints the version and the alpha notice.

## 5. Check the checksum list

`SHA256SUMS` has one line for each archive, in the form `<digest>  <file name>`. The build step checks it. Check it again
before you publish:

```sh
shasum -a 256 -c SHA256SUMS   # macOS
sha256sum -c SHA256SUMS       # Linux
```

## 6. Screen the release before publishing

Screen the release notes and every asset with the `release` surface of the public-safety leaf. Run it once per file.
Also unpack each archive into its own empty directory, and screen the `retriever` binary the same way. Build paths and
personal values can sit in a binary, and the scanner reads text.

```sh
scripts/public-safety/outbound-preflight.sh --surface release --file <path>
```

A workspace that holds named private identifiers passes its own list with `--terms`, and keeps that list outside the
checkout.

Exit `0` means the file is clean. Any other exit code blocks the release. Do not publish until every file exits `0`.

## 7. Write the release notes

The notes must contain these things:

- the alpha notice: super-agent is alpha, and breaking changes are allowed in any release;
- the breaking changes for this release, copied from the changelog;
- the platforms, and the contents of `SHA256SUMS`; and
- a link to the changelog section for the version.

## 8. Create the pre-release

```sh
gh release create v0.1.0 --prerelease --latest=false --verify-tag --title "v0.1.0 (alpha)" \
  --notes-file <notes-file> <archive> ... SHA256SUMS
```

The `--prerelease` flag marks the release as a pre-release. The `--latest=false` flag keeps it from becoming the latest
release. The `--verify-tag` flag aborts the release when the tag is not on `origin`.

## 9. Verify the published assets

Download the assets into an empty directory, and check them against the published `SHA256SUMS`:

```sh
gh release download v0.1.0 --dir <empty-directory>
```

Run the check from step 5 in that directory. Run `retriever --version` and `retriever --help` for your host platform.
Confirm on the release page that it is marked **Pre-release**.

## If a release is wrong

Never move a tag, replace an asset, or edit `SHA256SUMS` by hand. Cut the next version instead, as
[Release](../../repo-governance/workflows/maintenance/release.md) requires.
