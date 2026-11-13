# Developing Applications: Part 2 of 2

## Prove Host Integrations in Their Host

When behavior depends on an editor, terminal multiplexer, shell, REPL, daemon, or other stateful host, a mocked adapter
does not close the work. Follow Host-Integrated Runtime Verification: keep decision logic narrow,
exercise the real host in a unique isolated fixture, reproduce the exact ordered actions, and inspect observable process
and host state after each transition. Use bounded readiness checks instead of sleeps, test refusal paths before input is
sent, and prove teardown removed only resources owned by the run.

Use a live workspace only with explicit authorization. Inspect it before acting, preserve authorized user files, and
leave user-owned panes and processes in the requested final state. An isolated pass proves repeatability; a live pass
proves installed wiring. Neither substitutes for the other when both apply.

## Before Calling It Ready

- every error path is tested and ends handled, propagated with context, or failing loudly;
- no log line, fixture, or example holds a secret or a personal value;
- every entry point the change adds validates its input;
- no literal leaves a reader guessing its meaning; and
- every new dependency is justified and locked, and any dependency the change left unused is removed.
- host-integrated behavior has isolated real-host evidence and any applicable authorized live smoke test.
