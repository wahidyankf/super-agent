# GitHub Configuration

Repository automation for pull requests and scheduled checks. The workflows run on GitHub-hosted runners only, and no
job uses a secret or a self-hosted runner.

- [workflows/](workflows/README.md) — the pull-request quality gate and the consumer smoke checks.
- [actions/](actions/README.md) — composite actions that set up the toolchains the workflows use.
- [pull_request_template.md](pull_request_template.md) — the description every pull request fills in: summary, gate
  results, leak-scan statement, and the alpha breaking-change checkbox.
