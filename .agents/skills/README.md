---
description: >-
  Indexes the governed documents and reading order for Canonical Skills.
when_to_use: >-
  Use when navigating Canonical Skills or locating one of its indexed artifacts.
---

# Canonical Skills

One directory per skill, each holding the complete skill body and any companion resources it references. A skill
longer than one file splits into numbered parts (`001-`, `002-`, ...) that its `SKILL.md` lists in reading order.

Generated harness adapters under `.claude/skills/` route to these directories. A harness whose profile declares no skill
adapter reads this tree directly.

## Directory Map

- [building-command-line-interfaces](./building-command-line-interfaces/) — applying the command-line interface contract:
  choosing an exit status, placing output on the right stream, and recognising which runtime default to override.
- [cutting-releases](./cutting-releases/) — judging whether a version is ready to publish, when a version number is
  already spent, and how to verify what consumers receive.
- [developing-applications](./developing-applications/) — language-agnostic application judgement: layers, error fates,
  logs, input validation, and test-first work.
- [docs-applying-content-quality](./docs-applying-content-quality/) — Markdown content quality: active voice, heading
  hierarchy, accessibility, and professional formatting.
- [docs-validating-factual-accuracy](./docs-validating-factual-accuracy/) — verifying factual correctness in
  documentation, with a confidence-classification system.
- [exploratory-testing](./exploratory-testing/) — session-based exploratory testing of a running interface or API, with a
  cited oracle for every expected result.
- [grill-me](./grill-me/) — presenting a decision as exclusive options with one recommendation, so a material choice is
  resolved deliberately.
- [plan-writing-gherkin-criteria](./plan-writing-gherkin-criteria/) — writing Given/When/Then acceptance scenarios that
  describe observable behaviour and can fail.
- [producing-review-findings](./producing-review-findings/) — what a discipline reviewer raises and how it writes each
  finding.
- [programming-golang](./programming-golang/) — Go work under the Go standard.
- [programming-shell](./programming-shell/) — shell work under the shell standards.
- [propagating-rules](./propagating-rules/) — recognising rule work and entering Rules Propagation before edits.
- [repo-applying-maker-checker-fixer](./repo-applying-maker-checker-fixer/) — the three-stage content quality pattern and
  how each role executes.
- [repo-assessing-criticality-confidence](./repo-assessing-criticality-confidence/) — classifying findings by criticality
  and confidence, and the priority matrix that follows.
- [repo-generating-validation-reports](./repo-generating-validation-reports/) — generating validation reports with UUID
  chains, progressive writing, and UTC+7 timestamps.
