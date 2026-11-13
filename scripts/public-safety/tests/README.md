---
title: Public-safety tests
description: The self-test suite for the public-safety gate. It runs each case and fails when any case fails.
when_to_use: Read this before you change a public-safety screen, or when a case fails.
---

# Public-safety tests

These tests prove the public-safety screens. Each case builds its own synthetic input, so no test needs a real term
set or a real repository.

- [`run.sh`](./run.sh) runs every case, or only the cases whose names match the arguments you give it. It gives each
  case a private temporary directory and fails the suite when any case fails.
- [`cases/`](./cases/README.md) holds one file per case, numbered in run order.

Run the suite from the repository root:

```bash
bash scripts/public-safety/tests/run.sh            # all cases
bash scripts/public-safety/tests/run.sh 080        # one case by name fragment
```
