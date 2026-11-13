---
description: >-
  States that every rule, file, step, and feature must earn its place by being needed, and that the smallest sufficient
  form is the right one.
when_to_use: >-
  Use when adding a rule, file, dependency, or feature, when a document grows past its purpose, or when reviewing a
  change for what it does not need.
---

# Minimal Sufficiency

Add only what the goal needs. Keep only what the reader or the program actually uses. The smallest form that does the
job is the right one.

## The Test

Before adding something, answer three questions:

1. What does it let someone do or check that they cannot do or check now?
2. Who needs it, and when?
3. What is the smallest form that meets that need?

If the answer to the first question is vague, the addition is not yet justified. If the smallest form is larger than the
need, reduce it.

## Where It Applies

- **Rules.** A rule states one requirement once. A rule that repeats another is removed, and the survivor is linked.
- **Documents.** A document holds what its reader needs for its task. Detail that a reader rarely needs moves to a
  linked page, which is progressive disclosure.
- **Code and configuration.** A dependency, flag, or setting exists only while something uses it.
- **Features.** A feature is built when a user needs it, not when someone imagines a use.

## What It Does Not Remove

Minimal sufficiency never removes a check that protects a contract, a safety rule, or a record that an auditor needs.
Removing one of those is a loss, not a simplification. A requirement that is hard to meet is a reason to design better,
not to drop it.

## Related

- [Word Limit](../conventions/writing/word-limit.md) is the sanctioned way to shorten an over-budget file.
- [Vision](../vision/super-agent.md) states what the repository is for, which decides what it needs.
