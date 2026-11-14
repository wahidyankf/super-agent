# schema

The JSON Schema of a connector spec, version 1: [connector-spec-v1.schema.json](./connector-spec-v1.schema.json), draft
2020-12. The spec module embeds the file and compiles it once. Its compiler has no loader that reads outside the file,
so a reference to another document fails.

## Parts

The named parts sit under `$defs`. They cover the kind, the text, the constant and body values, the names of keys and
slots, the parameter types, the endpoint, the authentication, the limits, the parameter, the response, the pagination,
the example, and the operation. The parameter types are the 21 kinds of the operation catalog, and a test checks that
list against the catalog. The operation holds the access matrix as `allOf` rules, which pair each label with its method
and the `read_over_post` marker. The corpus test checks that the schema and the Go matrix agree on every row.

## Key names

Each map of the spec checks its keys with `patternProperties` and `additionalProperties: false`. A key that breaks the
pattern is reported with its own name, at the path of its map. The file does not use `propertyNames`, because the
validator library reports a name-level failure at a location that a later field can overwrite.

## What the schema does not state

- A reserved header name, such as `Authorization`, as a key of `headers`. The schema refuses it only as the value of
  `auth.header_name`. The Go rules refuse it as a key, without regard to case.
- A rule that compares two fields of one document. Those are in [semantic.go](../semantic.go).
- The nesting depth of a POST read's body. The schema cannot count levels, so [matrix.go](../matrix.go) checks it. The
  engine checks the size of the rendered body, because a slot can fill a list.

## Changing the file

A change to this file needs a change to the Go rules that a test pins. The test in
[corpus_test.go](../corpus_test.go) runs each valid and refused document through the schema and the Go validation, and
it fails when the two disagree about a row.
