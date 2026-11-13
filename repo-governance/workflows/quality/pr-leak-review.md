---
description: >-
  Defines the leak review of every pull request: the scan over the diff, the reading of names and messages, the leak
  classes to look for, and the rule that a review reports where a leak is, never what it is.
when_to_use: >-
  Use before merging any pull request, and when a review needs to decide whether a diff, a branch name, or a commit
  message may be published.
---

# Pull Request Leak Review

Every pull request is read for leaks before it merges. The review covers the diff, the names, and the text around the
change. A scan is necessary but not enough: it matches shapes, and a person reads what the shapes miss.

## Scope

The review covers every surface that a pull request publishes:

- the content added, changed, or deleted in the diff;
- the names of files, folders, and branches that the diff touches;
- every commit message in the range; and
- the title and the body of the pull request, and any comment on it.

Commit author and committer identity is not reviewed for leaks. See [git
identity](../../development/workflow/git-identity.md).

## Leak Classes

The classes are the categories in the [public repository data
policy](../../conventions/security/public-repository-data-policy.md):

- secrets and credentials;
- names of real organisations, people, tenants, and private repositories;
- host names, domain names, network addresses, and device names;
- personal paths, usernames, and account identifiers;
- real work or personal data; and
- contact details other than the accepted commit identity.

## Steps

1. **Run the scan** over the diff and over the commit messages in the range. A scan that cannot run blocks the review.
2. **Read the diff.** Read every added line and every renamed path, not only the lines the scan flagged.
3. **Read the names.** Check each branch name, file name, and commit message as content.
4. **Read the text around the change.** Check the title and body, and every review comment that the pull request has
   published.
5. **Decide.** The review passes only when no class applies. Any doubt blocks the merge until it is resolved.

## Reporting

A review reports the file, the rule, and the line or commit where a leak sits. It never copies the matched value into
the report, because the report would publish it again. A reviewer who finds a leak stops the merge and tells the owner.

## Rules

- No allowlist, suppression comment, environment variable, or force flag bypasses the review.
- A false positive is fixed by narrowing the rule in a reviewed change, never by ignoring the finding.
- A leak that has already been pushed follows [No Secrets in Tracked
  Files](../../conventions/security/no-secrets-in-tracked-files.md): rotate first, then remove the value from every
  reachable ref.

## Enforcement

The `public-safety-range` gate runs the scan on the pull-request range. The review itself is a person's reading, and no
gate can replace it.
