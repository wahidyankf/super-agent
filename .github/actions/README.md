# Composite Actions

Local composite actions that give the workflows one repeatable toolchain setup. They target GitHub-hosted runners.
`setup-lint-tools` pins Linux x86_64 assets only.

- [setup-node](setup-node/action.yml) — installs the Node.js and npm versions pinned under `volta` in `package.json`
  through Volta, then runs `npm ci --ignore-scripts`. Set `run-npm-ci` to `false` for a job that needs no Node-managed
  tool.
- [setup-go](setup-go/action.yml) — installs the pinned Go toolchain. The Go quality job uses it once a Go module exists.
- [setup-lint-tools](setup-lint-tools/action.yml) — installs the shellcheck, actionlint, and shfmt release binaries that
  the declared gates call. Each asset is refused unless its SHA-256 matches the pinned digest.

Keep versions and installation logic in each `action.yml`, not in a calling workflow. Never put a credential or an
environment value in an action file.
