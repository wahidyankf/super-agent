# Composite Actions

Local composite actions that give the workflows one repeatable toolchain setup. They target GitHub-hosted runners.
`setup-lint-tools` pins Linux x86_64 assets only.

- [setup-node](setup-node/action.yml) — installs the Node.js and npm versions pinned under `volta` in `package.json`
  through Volta, then runs `npm ci --ignore-scripts`. Set `run-npm-ci` to `false` for a job that needs no Node-managed
  tool.
- [setup-go](setup-go/action.yml) — installs the pinned Go toolchain. The Go quality job and the cross-compile check use
  it. Its default version equals the `toolchain` line of each Go module. After the install it fails unless every
  `go.mod` under `apps` and `libs` names the installed release, and the modules agree on their `go` line. Its cache
  key takes one or more dependency-file globs, one per line.
- [setup-lint-tools](setup-lint-tools/action.yml) — installs the shellcheck, actionlint, and shfmt release binaries that
  the declared gates call. Each asset is refused unless its SHA-256 matches the pinned digest.

Keep versions and installation logic in each `action.yml`, not in a calling workflow. Never put a credential or an
environment value in an action file.
