# Workflows

GitHub Actions workflows. The workflows run the repository's commands through the pinned HIPPO wrapper, and the
repository gates run through the pinned RHINO wrapper.

- [pr-quality-gate.yml](pr-quality-gate.yml) — runs on every pull request to `main`. It detects affected Nx projects by
  their `lang:*` tag and runs the repository policy under RHINO. When a Go project is affected, it runs the targets of
  every Go project (`lang:go`) on Ubuntu, the application's targets (`type:app`) on macOS, and a cross-compile check of
  the application for four platforms. Projects are selected from the Nx project graph by tag, so a new Go module needs
  no edit here. The Go setup checks the installed Go against the `toolchain` line of every `go.mod`. The workflow
  reports one aggregate `quality-gate` check. That check is the single required status check.
- [hippo-consumer-smoke.yml](hippo-consumer-smoke.yml) — installs the pinned HIPPO release from an empty cache on Linux
  and macOS, and checks its identity against `hippo.lock`. It runs when the HIPPO pin changes and every day.
- [rhino-consumer-smoke.yml](rhino-consumer-smoke.yml) — installs the pinned RHINO release from an empty cache on Linux
  and macOS, and checks its identity against `rhino.lock`. It runs when the RHINO pin changes and every day.

No scheduled full-quality workflow exists yet. The two consumer smoke checks are the only scheduled jobs.

Third-party actions are pinned by full commit SHA, with the release tag as a comment. A new pin needs its SHA verified
against the upstream tag before it is committed.
