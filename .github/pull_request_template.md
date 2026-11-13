## Summary

<!-- What changes and why, in one or two short paragraphs. -->

## Gate results

<!-- Name the head commit and the result of each gate. Do not paste log output. -->

- [ ] `quality-gate` is green on the head commit.
- [ ] `./rhino gate run --surface pull-request` passed for this range, through `./hippo run`.
- [ ] Affected Nx targets (`typecheck`, `lint`, `test:quick`) passed, or no Go project is affected.

## Leak scan

- [ ] The public-safety scan is clean for the tree, for every commit in this pull request, and for this title and body.

<!-- State what was scanned, not what matched. Never paste a matched value into this description. -->

## Compatibility

This project is alpha, and its interfaces can change without notice until a stable release.

- [ ] This change breaks a user of the CLI, a connector spec, or a JSON output contract. It is recorded in `CHANGELOG.md`.
- [ ] This change does not break an existing user.
