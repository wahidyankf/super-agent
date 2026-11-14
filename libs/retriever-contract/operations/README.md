# operations

The operation catalog. Each row describes one operation completely: its ID, its source kind, its parameters, its fields,
and its effect. The use cases look up a row, so a command and its help text come from the same description.

- `bind.go` binds a row to its parameters and checks the values against the row.
- `operations.go` holds the catalog type and the lookups over it.
- `resources.go` holds the resource names that group the rows.
- `rows.go` holds the Outline rows for the first group, and the field lists that several rows share.
- `rows_b.go`, `rows_c.go`, `rows_c2.go`, `rows_c3.go`, `rows_c4.go`, and `rows_2b.go` hold the Outline rows of the
  later groups. The suffix names the group.
- `rows_kiwi.go` holds the Kiwi rows.
- `rows_jira.go` holds the Jira rows.
- `*_test.go` files hold the tests of the catalog.

A new operation is one row here, plus the source's own code that performs it. The
[add a connector](../../../docs/how-to/add-a-connector.md) guide lists the steps.
