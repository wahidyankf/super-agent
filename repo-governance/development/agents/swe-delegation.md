---
description: >-
  Requires a session doing coding work to dispatch the fitting swe agent instead of doing the work in the main thread.
  Defines coding work and names the two exceptions.
when_to_use: >-
  Use before creating, modifying, updating, or deleting code or declarative configuration in a session, or before
  debugging, reviewing, or testing code.
---

# SWE Delegation

When a session creates, modifies, updates, or deletes code or declarative configuration, or debugs, reviews, or tests
code, it dispatches the fitting `swe-*` agent rather than doing the work in the main thread.

## Coding Work

Coding work is two kinds of content:

- **Code:** the Go application and libraries, scripts, tests, and any script that carries behaviour.
- **Declarative configuration:** settings held as data, such as `repo-config.yml`, JSON files, and package manifests.

Documents, rules, plans, and agent or skill definitions in Markdown are not coding work. The documentation and
governance rules cover them.

## Which Agent

| The work                                                      | Agent              |
| ------------------------------------------------------------- | ------------------ |
| a goal spanning several coding tasks or agents                | `swe-orchestrator` |
| a module boundary, a new dependency, or a trade-off to decide | `swe-architect`    |
| new or changed behaviour, configuration, or findings to apply | `swe-developer`    |
| a failing type check, lint run, or test                       | `swe-debugger`     |
| a static review of changed code against the adopted standards | `swe-reviewer`     |
| judging a running interface, or first use of a command line   | a tester role      |
| cutting or deploying a release, or repinning a pinned tool    | a release role     |

The agent index states which of these roles this repository has adopted. A role not yet adopted is not performed by the
main thread without a reason in the report.

## Family Fixers

A fixer that finds a repair changing declarative configuration or code does not write that change. It decides the
repair, names `swe-developer` as the repairer in its report, and its caller dispatches `swe-developer` to make the
change. The fixer then confirms that the change landed.

## Exceptions

The main thread may do coding work itself only in these cases:

1. **A harness with no subagent support,** where no agent can be dispatched. The report gives this reason.
2. **A tool repin** of a pinned upstream tool, such as HIPPO or RHINO, while no release role is adopted. The report
   names the repin.

A session that uses an exception says so in its report.

## Why

The swe agents carry the adopted standards and the test-first discipline. Execution-tier agents run on the model that
the repository maps to that tier, and the files they read stay out of the main session's context. That keeps the
reasoning that follows clear. The root [AGENTS.md](../../../AGENTS.md) repeats the rule in one line, because a
contributor must see it before opening any file.

## Enforcement

Unenforced by decision: a session's delegation shows only in its transcript, which no repository check reads, and a
check of changed paths cannot tell which agent wrote them. Review and the session's own report carry the rule.
