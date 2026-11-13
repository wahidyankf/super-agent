---
description: >-
  Defines the terms this repository uses with a fixed meaning, such as super agent, connector, source, catalog, access
  level, harness, alpha, HIPPO, RHINO, worktree-to-PR, remote dispatch, and the public repository data policy.
when_to_use: >-
  Use when a term in a rule, a page, or a pull request is unclear, or before using a project term in a new document.
---

# Glossary

Terms are listed in alphabetical order. A term's meaning here holds in every governance file.

**Access level.** The highest operation a source allows a connector to perform: `read`, `write`, or `admin`. The level
is a ceiling. A connector never performs an operation above the level that its source is given, and a command or harness
cannot raise it.

**Alpha.** The release stage of super-agent before a stable 1.0. In alpha, commands, flags, file formats, connector
specifications, JSON output, and APIs may change in any release, including a patch release. Each breaking change is
named in the changelog.

**Catalog.** The user's list of sources, with each source's kind, its access level, and the names of the credentials it
uses. A catalog lives in the user's configuration directory, outside this repository. The repository never holds a real
catalog. A profile holds one catalog.

**Connector.** The code that talks to one kind of source and exposes its operations through the command-line tool and
the protocol server. A connector checks access before it reads or writes. A declarative connector is described by a
specification, and it runs through one shared engine.

**Conformance runner.** The check that runs the published contract of the command-line tool, including exit statuses,
error codes, flags, and JSON shapes, before a pull request merges.

**Harness.** A coding-agent runtime that calls super-agent through its command-line tool or its protocol server. Many
harnesses can share the same connectors and the same access levels.

**HIPPO.** A pinned, checksum-verified admission guard for local compute. It starts a build, test, or check only when
shared capacity and memory pressure allow it. See [Resource-Aware
Development](development/practice/resource-aware-development.md).

**Public repository data policy.** The rule that this public repository never holds secrets, real names, host details,
personal paths, work or personal data, or contact details. The policy covers file content, names, commit messages,
history, and release assets. See [Public Repository Data Policy](conventions/security/public-repository-data-policy.md).

**Pre-release.** A GitHub release marked as a pre-release. Releases of super-agent are `v0.x.y` pre-releases until the
owner decides otherwise.

**Profile.** A named set of one catalog, its credentials, and its access ceiling. Profiles keep work and personal
contexts apart, and one profile is active for each run. A profile is planned, not yet built.

**Remote dispatch.** Running compute on another machine that the owner controls. Dispatch is allowed only through a
local binding found at `REMOTE_DISPATCH_HOME`. See [Remote
Dispatch](development/practice/resource-aware-development/remote-dispatch.md).

**RHINO.** The pinned, checksum-verified tool that runs this repository's quality gates, as declared in
`repo-config.yml`.

**Source.** One configured system that a connector reaches: a document system, an issue tracker, a messaging channel, or
a local folder. A source has one kind, one address, one access level, and the names of its credentials.

**Super agent.** The human plus the harness they use. The person orchestrates many coding agents and many data sources
and channels, and super-agent gives them one set of connectors and one access model. The term does not name a single
agent. The repository never defines a catch-all agent by this name.

**Worktree-to-PR.** The delivery path for every change after the bootstrap commit: a task worktree, a commit, a pull
request, the checks, a squash merge when the rules allow it, and cleanup. See [Worktree to Pull
Request](workflows/maintenance/worktree-to-pull-request.md).
