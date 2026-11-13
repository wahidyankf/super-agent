# Documentation

super-agent is in alpha. The repository holds its skeleton today: quality gates, governance rules, this
documentation, and an empty plans folder. The command-line tool, `super-agent-cli`, arrives in the next pull
request. Until it does, the pages below explain the project and describe the repository work you can do now.

## Start here

New to super-agent? Read [What super-agent is](./explanation/what-super-agent-is.md) first. It explains the name and
the scope. Then read [Stability and versioning](./explanation/stability-and-versioning.md) before you depend on the
project.

## Find the right kind of help

This documentation follows the [Diátaxis framework](https://diataxis.fr/). Pick the section that matches your
question.

| Section                       | Use it when                                                     |
| ----------------------------- | --------------------------------------------------------------- |
| [Tutorials](./tutorials/)     | You are new and want to learn by doing. None exist yet.         |
| [How-to guides](./how-to/)    | You have a goal and need the steps to reach it.                 |
| [Reference](./reference/)     | You need an exact name, gate id, or the repository layout.      |
| [Explanation](./explanation/) | You want to understand why the project is shaped the way it is. |

## What each section holds

- [Tutorials](./tutorials/) has no pages yet. Tutorials arrive with the command-line tool.
- [How-to guides](./how-to/) are directions for tasks you already decided to do:
  - [Deliver a change with a pull request](./how-to/deliver-a-change-with-a-pull-request.md) covers worktrees, CI
    polling, and the merge rules.
  - [Run the quality gates](./how-to/run-the-quality-gates.md) runs the gates through HIPPO and RHINO.
  - [Repin HIPPO and RHINO](./how-to/repin-hippo-and-rhino.md) moves the checksum pins to a new release.
  - [Dispatch a task to another machine](./how-to/dispatch-a-task-to-another-machine.md) follows the remote-dispatch
    rule.
  - [Cut an alpha pre-release](./how-to/cut-an-alpha-pre-release.md) lists the steps for a GitHub pre-release.
- [Reference](./reference/) states facts exactly:
  - [Repository layout](./reference/repository-layout.md) shows the real directory tree.
  - [Environment variables](./reference/environment-variables.md) lists variable names and where values go.
  - [Quality gates](./reference/quality-gates.md) lists every gate id in `repo-config.yml`.
  - [Scripts overview](./reference/scripts-overview.md) summarises the wrapper scripts.
- [Explanation](./explanation/) gives the reasons behind the project:
  - [What super-agent is](./explanation/what-super-agent-is.md) describes the super agent and its scope.
  - [Stability and versioning](./explanation/stability-and-versioning.md) explains what alpha means for you.
  - [Public data policy](./explanation/public-data-policy.md) explains why the repository is public and what stays
    out.
  - [Software-factory note](./explanation/software-factory-note.md) records a later direction. It is out of scope
    now.

## Related

- [Specifications](../specs/README.md) hold the executable description of the behaviour. None exist yet.
- [Plans](../plans/README.md) record planned work.
- [AGENTS.md](../AGENTS.md) gives the rules for coding agents and contributors.
- [CHANGELOG.md](../CHANGELOG.md) lists each release and its breaking changes.
