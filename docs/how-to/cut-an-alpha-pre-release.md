---
title: Cut an alpha pre-release
description: Manual steps to cut a v0.x pre-release: changelog, tag, checksums, leak scan, and GitHub pre-release.
category: how-to
tags:
  - how-to
  - release
  - alpha
---

# Cut an alpha pre-release

No release exists yet. The command-line tool in `apps/retriever-cli` is the first thing a release would publish. This
guide lists the manual steps a release uses. The repository has no release automation yet. If automation is added
later, it replaces this guide.

## 1. Check the state

- Every change for the release is merged into `main` through a pull request.
- The pull-request gates pass on `main`. See [Run the quality gates](./run-the-quality-gates.md).

## 2. Update the changelog

In [CHANGELOG.md](../../CHANGELOG.md), move the entries under `Unreleased` into a heading for the version, such as
`## v0.1.0`. Do not add a date to the heading. The GitHub release records the date.

List every breaking change under its release with a **Breaking** label. A reader must be able to see every break
before they upgrade.

Merge the changelog change through a pull request, as in [Deliver a change with a pull
request](./deliver-a-change-with-a-pull-request.md).

## 3. Build the archives

No archive build exists yet. The `build` target in `apps/retriever-cli/project.json` writes one binary,
`dist/retriever`,
for the host platform only. A release that publishes archives for each platform, plus a `checksums.txt` file, needs a
build step that this repository does not have. Document that step here when it exists.

## 4. Write the checksum list

Compute the SHA-256 digest of each archive. Write one line per archive in the form `<digest> <file name>`.

```sh
shasum -a 256 <archive> ... > checksums.txt   # macOS
sha256sum <archive> ... > checksums.txt       # Linux
```

Check the list against the archives in the same directory:

```sh
shasum -a 256 -c checksums.txt   # macOS
sha256sum -c checksums.txt       # Linux
```

## 5. Tag the release commit

After the repository owner approves the release, tag the merged commit on `main` and push the tag.

```sh
git tag -a v0.1.0 -m "v0.1.0" <commit>
git push origin v0.1.0
```

## 6. Screen the release before publishing

Screen the release notes and every asset with the `release` surface of the public-safety leaf. Run it once per file.

```sh
scripts/public-safety/outbound-preflight.sh --surface release --file <path>
```

Exit `0` means the file is clean. Any other exit code blocks the release. Do not publish until every file exits `0`.

## 7. Write the release notes

The notes must contain three things:

- the alpha notice: super-agent is alpha, and breaking changes are allowed in any release;
- the breaking changes for this release, copied from the changelog; and
- the contents of `checksums.txt`.

## 8. Create the pre-release

```sh
gh release create v0.1.0 --prerelease --title "v0.1.0" --notes-file <notes-file> <archive> ... checksums.txt
```

The `--prerelease` flag marks the release as a pre-release. Do not mark it as the latest release.

## 9. Verify the published assets

Download the assets into an empty directory, then check them against the published checksum list:

```sh
gh release download v0.1.0 --dir <empty-directory>
```

Run the check from step 4 in that directory. Confirm on the release page that it is marked **Pre-release**.
