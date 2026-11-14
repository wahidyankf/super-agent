# retriever-spec

The connector spec of ADR 001: the model of a declarative connector, its JSON Schema, the validation that refuses a bad
spec, and the plain-text review of a valid one. The module takes bytes and returns a model or findings. It reads no
file, environment variable, clock, network, or process, so the app reads the spec file and passes the bytes in.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-spec`.
- Nx project `retriever-spec`, with the tags `type:spec`, `lang:go`, and `domain:retriever`.
- `go.mod` requires the contract at `v0.4.0`, through a `replace` to the sibling folder. It also requires the JSON
  Schema library and the YAML library named below. The production code imports the contract, those two libraries,
  and the standard library.

## Validation

A document passes three checks, in order. A later check runs only when the earlier one passes.

1. The YAML reader (`parse.go`) reads one document of at most 256 KiB. It refuses an alias, an anchor, a repeated field
   name, a tag other than the plain forms, and a number that is not finite.
2. The JSON Schema (`schema.go`, with the file in [schema/](./schema/README.md)) checks the shape and the per-value
   rules.
3. The Go rules (`semantic.go`) check what a schema cannot state, because each one compares two fields of the document.
   Examples are an operation id against the kind, a slot against a parameter, and a resolve against an operation.

A refusal returns findings. Each finding names a JSON pointer, such as `/operations/1/access`, a fixed rule, and the
stable code `retriever.connector.spec_invalid`. No finding shows a value of the spec, and a field name that is not a
plain identifier is withheld.

The review (`review.go`) states the endpoint, the authentication, the limits, and each operation with its label, method,
and path. It names each variable by name and shows no value.

## Libraries

- YAML: `go.yaml.in/yaml/v3`, the same family that the runtime reads with. It gives the node tree that the parser checks
  before it builds any value.
- JSON Schema: `github.com/santhosh-tekuri/jsonschema/v6`. Its errors carry the location of the failing field and a
  kind that holds no value, so a finding can name a path without echoing a secret. The schema is compiled with a loader
  that refuses every reference outside the document. The library's own package imports the network and file packages,
  so the module relies on that loader and not on the import list to stay offline.

## Folders

- [schema/](./schema/README.md) holds the JSON Schema file of version 1.
- [testdata/](./testdata/README.md) holds the bundled example spec that the tests embed.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

Read [connector spec fields](../../docs/reference/connector-spec-fields.md) for each field and its rule, and
[library layering](../../docs/explanation/library-layering.md) for the imports this module may make.
