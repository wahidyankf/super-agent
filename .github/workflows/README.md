# Workflows

GitHub Actions workflows. The workflows run the repository's commands through the pinned HIPPO wrapper, and the
repository gates run through the pinned RHINO wrapper.

- [pr-quality-gate.yml](pr-quality-gate.yml) — runs on every pull request to `main`. It detects affected Nx projects by
  their `lang:*` tag, runs the repository policy under RHINO, runs the Go targets when a Go project is affected, and
  reports one aggregate `quality-gate` check. That check is the single required status check.
- [hippo-consumer-smoke.yml](hippo-consumer-smoke.yml) — installs the pinned HIPPO release from an empty cache on Linux
  and macOS, and checks its identity against `hippo.lock`. It runs when the HIPPO pin changes and every day.
- [rhino-consumer-smoke.yml](rhino-consumer-smoke.yml) — installs the pinned RHINO release from an empty cache on Linux
  and macOS, and checks its identity against `rhino.lock`. It runs when the RHINO pin changes and every day.

No scheduled full-quality workflow exists yet. It would have no content until the first Nx project lands in PR 2.

Third-party actions are pinned by full commit SHA, with the release tag as a comment. A new pin needs its SHA verified
against the upstream tag before it is committed.
