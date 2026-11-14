---
title: Repository layout
description: The real directory tree of the super-agent repository, with what each top-level entry holds today.
category: reference
tags:
  - reference
  - layout
---

# Repository layout

The tree below is the repository as it is today. Entries marked "none yet" hold only a README. Folders marked
"generated" come from the canonical sources, so never edit them by hand. Git ignores the contents of `local-tmp/`,
`worktrees/`, `node_modules/`, and `.nx/`.

```text
super-agent/
├── .agents/                 canonical skill sources (skills/<name>/SKILL.md)
├── .claude/                 generated Claude Code adapters
├── .codex/                  generated Codex adapters
├── .commandcode/            generated Command Code adapters
├── .github/                 pull-request template, CI workflows, and local actions
├── .husky/                  Git hooks: pre-commit, commit-msg, pre-push
├── .nx/                     local Nx cache; ignored by Git
├── .opencode/               generated OpenCode adapters
├── .serena/                 Serena policy; only its README is tracked
├── agents/                  canonical agent definitions
├── apps/                    applications; retriever-cli is the command-line tool
├── docs/                    documentation, organised by Diátaxis
│   ├── explanation/         why the project is shaped this way
│   ├── how-to/              task-oriented guides
│   ├── reference/           exact names, gate ids, and layout
│   └── tutorials/           lessons; none yet
├── libs/                    shared libraries; none yet
├── local-tmp/               local working state; ignored by Git
├── node_modules/            npm packages; ignored by Git
├── plans/                   planned work: ideas, backlog, in-progress, done
├── repo-governance/         conventions, principles, workflows, and development practice
├── scripts/                 wrapper scripts and the public-safety gate
├── specs/                   specifications for the command-line tool
├── worktrees/               one checkout per task; ignored by Git
├── AGENTS.md                rules for coding agents and contributors
├── CHANGELOG.md             release history and breaking changes
├── CLAUDE.md                points Claude Code at AGENTS.md
├── LICENSE                  MIT licence
├── README.md                project front page
├── commitlint.config.js     Conventional Commits rules
├── hippo, rhino             bootstrap wrappers for the pinned tools
├── hippo.lock, rhino.lock   pinned versions and SHA-256 digests
├── hippo.identity.json      identity HIPPO records for runs in this repository
├── hippo.local.json.example template for a machine's HIPPO policy
├── repo-config.yml          RHINO configuration: gates, scans, and policies
├── package.json             npm workspace and scripts
├── package-lock.json        npm lock file
├── nx.json                  Nx configuration
├── Brewfile                 Homebrew tools for `brew bundle`
├── .editorconfig            editor rules: UTF-8, LF line endings, final newline
├── .gitignore               paths Git does not track
├── .markdownlint-cli2.jsonc markdownlint configuration
├── .markdownlintignore      paths markdownlint skips
├── .nxignore                paths Nx does not read, including worktrees
├── .prettierignore          paths Prettier skips
├── .prettierrc.json         Prettier settings: 120-column print width
├── .shellcheckrc            ShellCheck settings: bash by default
└── .env.example             environment template; no names yet, values stay empty
```

## GitHub configuration

The `.github/` folder holds the pull-request template, the CI workflows, and the local composite actions. The workflows
run on GitHub-hosted runners only, and no job uses a secret or a self-hosted runner.

- [workflows](../../.github/workflows/README.md): the pull-request quality gate and the consumer smoke checks.
- [actions](../../.github/actions/README.md): composite actions that set up the toolchains the workflows use.
- [pull_request_template.md](../../.github/pull_request_template.md): the description every pull request fills in. It
  has the summary, the gate results, the leak-scan statement, and the alpha breaking-change checkbox.

The repository front page is the root [README.md](../../README.md). GitHub prefers a `README.md` in `.github/` over the
root one, so keep no README there.

## Where to look

- Gate ids and the surfaces they run on: [Quality gates](./quality-gates.md).
- Variable names: [Environment variables](./environment-variables.md).
- Wrapper scripts: [Scripts overview](./scripts-overview.md).
- Why the repository is organised this way: [Explanation](../explanation/README.md).
