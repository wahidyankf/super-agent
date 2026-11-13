---
name: plan-writing-gherkin-criteria
description: >-
  Guides writing Given/When/Then acceptance scenarios that describe observable behaviour and can actually fail.
when_to_use: >-
  Use when authoring or reviewing the acceptance criteria section of a plan's product requirements.
compatibility: Requires no tools beyond the plan being written.
---

# Writing Gherkin Criteria

A scenario states a condition that could fail. That is the whole test of a good one.

## The Three Clauses Do Different Work

- **Given** — the state before, and only what matters. Setup that no clause depends on is noise, and noise is where
  wrong assumptions hide.
- **When** — one action. Two actions means two scenarios, or a scenario that cannot say which action caused the result.
- **Then** — one observable outcome. Observable means someone or something outside the system can see it.

## Observable, Not Internal

`Then the cache is warmed` describes an implementation. It will be marked complete by whoever wrote the cache, and it
will keep passing after the cache is removed.

`Then the second request returns without contacting the upstream service` describes behaviour. It survives the rewrite,
and it fails when the behaviour goes away — which is the only thing it was for.

## Falsifiability Is the Filter

Ask: what would this look like if it were false? A scenario with no answer is not an acceptance criterion. It is a
statement of intent that has been formatted like one.

`Then the interface is intuitive` has no failing case. `Then a first-time user completes checkout without opening help`
does.

## One Scenario, One Claim

A scenario asserting several things fails as a unit and reports nothing about which part broke. Splitting costs a few
lines and buys a diagnosis.

## Concrete Values

State actual values: `Given a basket of 4 items worth 80.00`, `When the voucher "TENOFF" is redeemed`,
`Then the amount due is 70.00`. A basket "with some items" and an amount that "goes down" pass against nearly any
implementation, wrong ones included. When one behaviour holds across several inputs, use a scenario outline; each
examples row runs as its own scenario, so keep only the inputs that change the outcome.

## Each Success Has a Failing Twin

Criteria covering only the path where everything works leave the plan silent about the rest. For each success scenario,
ask which rule could refuse the same action, and write that case too:

- **create:** a missing required field shows its error and creates nothing;
- **list:** with nothing stored, the empty state appears;
- **update:** an invalid value is refused and the old one stays;
- **delete:** a record already gone is reported and nothing changes;
- **restricted action:** another role is refused and told why; and
- **dependency:** when it does not answer, the failure is shown and input is kept.

Boundaries count too: the last accepted value and the first refused one are two scenarios. [Discovery and
Scenarios][discovery-and-scenarios] surfaces such examples; this check confirms none was dropped.

## Behaviour, Not Clicks

Write what the actor does and sees, such as `When the member signs in with a valid password`, not the controls used.
Steps naming selectors, positions, or press sequences break on a redesign that leaves behaviour untouched. Use the
present tense, so each scenario states how the system behaves.

## Repeated Steps Are an Adopter Decision

Whether a scenario may repeat `When` and `Then` as one continuous journey is recorded once, under [Discovery and
Scenarios][discovery-and-scenarios], never per scenario. The clause rules above describe one action; under the journey
form they apply to each action and the `Then` that follows it, so every result still names its cause and the journey
checks one logical outcome. Never split a recorded journey only to reach one `When`, nor join unrelated behaviours into
one.

## Stable Identifiers

Each scenario carries an identifier that does not change when the text is edited or the list is reordered. Delivery
items cite it, reviews record against it, and other implementations match on it.

Renumbering scenarios silently breaks every reference to them, and nothing reports the break — the citations still
parse, they just point somewhere else now.

## Write Them Before the Technical Shape

Criteria written after the design describe the design. Written before, they constrain it, which is the entire reason to
write them down rather than assume them.

[discovery-and-scenarios]: https://github.com/wahidyankf/ose-rules/blob/18760a44f7ae4d88e89c0aa79349297f55bb9d9c/repo-governance/development/quality/testing/behaviour-driven-development/001-discovery-and-scenarios.md
