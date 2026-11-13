# Developing Applications: Part 1 of 2

## Place Code Before Writing It

Ask of each new piece whether it decides or does. A decision belongs in the domain or functional core and receives
everything it needs as a parameter. An effect belongs in an adapter or the shell. Coordinating a use case belongs in the
application layer.

Misplacement shows early: a domain function that wants a clock, a connection, or a logger; an adapter holding a
condition about a business rule; an application function choosing a transport status. Which layering fits a given
application is decided in
[Hexagonal Architecture](../../../repo-governance/development/pattern/hexagonal-architecture.md).

## Give Every Error One Fate

At each call that can fail, choose exactly one:

| Fate           | When                                                                      |
| -------------- | ------------------------------------------------------------------------- |
| handle it here | this code can do something meaningful: retry, fall back, or tell the user |
| propagate it   | the caller can decide, and this code adds context the caller lacks        |
| fail loudly    | the failure means a broken invariant or a programming error               |

Discarding it is never a fate. Add context where the meaning changes, such as a storage failure becoming "could not load
order `<id>`", not at every frame it crosses; the same message repeated at each layer buries the cause. Infrastructure
error types stop at their adapter, and a failure becomes a transport response once, as
[Hexagonal Architecture](../../../repo-governance/development/pattern/hexagonal-architecture.md) requires. Whether
expected failures travel as values or exceptions is the stack standard's decision, or, where no stack standard exists,
the language skill states the language's idiom or records the choice as an adopter decision. Every error path gets its
own test.

## A Log Line Is Evidence for Someone Else

Log a failure once, where it is finally handled; lines at every layer make one incident look like several. Write fields
rather than prose: the operation, the identifier acted on, the outcome, and the duration. Choose the level by who must
act: failures needing attention, degraded service, lifecycle events, and diagnostics.

Never write a credential, token, session identifier, full request body, or personal value to a log. Write an identifier
an authorized person can look up instead. Logging is an effect, so it stays in the shell or an adapter, and the core
returns what happened.

## Validate Once, Where Trust Ends

Validate input where it arrives from something the program does not control: a request, a file, a message, the
environment, or another service's response. Turn it into a typed value there, and trust that type inward. Re-checking
the same shape deep inside the domain signals an edge that let an unchecked value through.

Domain invariants are different: the domain enforces its own rules whatever the source. Check authorization on each
operation that touches protected data, not once at sign-in, and build every query with parameters, never by joining
input into its text.

## Test First, at the Narrowest Layer

Start each behaviour with a failing test at the narrowest layer that can prove it: a decision with injected dependencies
at unit, an adapter against a real local resource at integration, and a journey through the public boundary at
end-to-end, kept few. [Red, Green, Refactor](../../../repo-governance/development/workflow/test-driven-development.md)
owns the cycle.
