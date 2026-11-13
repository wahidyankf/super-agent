# Plans

Plans record planned work on super-agent: ideas, work waiting to start, work in progress, and finished work. Plans
are not product documentation. Product documentation lives under [docs](../docs/README.md).

No plan is open today.

## Find the right stage

- [`ideas/`](./ideas/) holds short, promotable idea briefs. Each idea is one `<slug>.md` file.
- [`backlog/`](./backlog/) holds full plans that are ready to start but have not started.
- [`in-progress/`](./in-progress/) holds plans that are being delivered now.
- [`done/`](./done/) holds finished plans, kept as the record of what was decided and delivered.

An idea moves to `backlog/` as a full plan when its open questions are settled. A plan moves to `in-progress/` when
its delivery starts, and to `done/` when it is finished.

## How delivery works

Work is delivered with worktree-to-PR. One worktree and one branch produce one pull request. See [Deliver a change
with a pull request](../docs/how-to/deliver-a-change-with-a-pull-request.md).

Plans are public. They follow the [public data policy](../docs/explanation/public-data-policy.md) in full, so they
hold no machine-specific value, secret, or private name.

For plan structure, naming, and lifecycle rules, read the [Plans Organization
Convention](../repo-governance/conventions/structure/plans.md). That convention is part of the repository
governance.
