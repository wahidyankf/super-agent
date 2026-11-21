# Workflows

GitHub Actions workflows. The workflows run the repository's commands through the pinned HIPPO wrapper, and the
repository gates run through the pinned RHINO wrapper. The gate inventory, which lists every gate and the surface it
runs on, is in [Quality gates](../../docs/reference/quality-gates.md).

- [pr-quality-gate.yml](pr-quality-gate.yml) runs on every pull request to `main`. It runs the pull-request surface of
  the registry under RHINO, once for the repository. It asks Nx which projects declare each target, and it runs one job
  per project and per level. Go projects run lint, typecheck, and dependency audit where declared, unit and integration
  tests on Ubuntu, and the app's unit and conformance tests on macOS. The behaviour check runs once for the repository,
  and a cross-compile check builds the app for four platforms. The workflow reports one aggregate `quality-gate` check.
  That check is the single required status check.
- [scheduled-e2e.yml](scheduled-e2e.yml) runs `test:e2e` for every project that declares it. It runs weekly, on manual
  dispatch, and on each push to `main`, and the app's suite also runs on macOS. The end-to-end level stays off the
  pull-request path. The workflow has no aggregate check: a failed job fails the run.
- [hippo-consumer-smoke.yml](hippo-consumer-smoke.yml) installs the pinned HIPPO release from an empty cache on Linux
  and macOS, and checks its identity against `hippo.lock`. It runs on pull requests that change the HIPPO pin or its
  workflow, and every day.
- [rhino-consumer-smoke.yml](rhino-consumer-smoke.yml) installs the pinned RHINO release from an empty cache on Linux
  and macOS, and checks its identity against `rhino.lock`. It runs on pull requests that change the RHINO pin or its
  workflow, and every day.

Third-party actions are pinned by full commit SHA, with the release tag as a comment. A new pin needs its SHA verified
against the upstream tag before it is committed.
