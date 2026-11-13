# Specifications

This tree is meant to hold the executable description of what super-agent does. Nothing is specified yet. The first
specification arrives with the command-line tool.

## What lives here

Each deployed surface has one owner folder. The command-line tool will be the first owner, at
`specs/apps/super-agent/cli/`.

```text
specs/apps/<product>/<owner>/
├── README.md                          what is here and where to go next
├── architecture.md                    the as-built system: context, containers, and components (C4)
└── behaviours/
    └── <domain>/<feature>.feature     Gherkin scenarios that describe observable behaviour
```

- **Gherkin scenarios** go under `behaviours/`, one folder per domain, one `.feature` file per feature.
- **The C4 architecture** goes in `architecture.md`, one document per owner.

Nothing in this tree exists yet. The layout is fixed so that the first owner lands in the right place.

## Related

- [Apps](../apps/README.md) holds the implementations. None exist yet.
- [Documentation](../docs/README.md) explains the project for people who use it.
