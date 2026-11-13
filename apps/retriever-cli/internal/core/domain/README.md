# internal/core/domain

The plain types that the use cases share. The package has no I/O and no adapter import.

- `access.go` defines the access levels `read`, `write`, and `admin`, the rule that one level allows another, and the
  error for a level that is too low.
- `apikey.go` defines `ApiKey`, a key type that never prints its value, and the error for a missing key.
- `args.go` defines the argument, field, record, and page types that carry a command's inputs and a result.
- `catalog.go` defines the catalog and the errors for a missing, unknown, or ambiguous source.
- `confirm.go` defines the error for an operation that needs a confirmation.
- `credentialref.go` defines `CredentialRef`, the name of the variable that holds a key.
- `document.go` defines the collection, document, and tree-node types.
- `domain.go` defines the output format type.
- `endpoint.go` defines `Endpoint`, an HTTPS host that a source calls.
- `errors.go` defines the shared error values.
- `events.go` validates an event list.
- `identifiers.go` defines the collection and document IDs.
- `identity.go` defines `Identity`, the account e-mail type. Like `ApiKey`, it never prints its value.
- `identityref.go` defines `IdentityRef`, the name of the variable that holds an account e-mail.
- `kinderror.go` defines the error that tells a kind which operation failed.
- `kindrules.go` defines the rules that a kind imposes: identity required or not used, and the access ceiling.
- `operation.go` defines the operation ID, its effect, and the error for an operation the server does not offer.
- `params.go` validates the ID, UUID, name, e-mail, text, time, index, colour, and icon parameters.
- `public_errors.go` lists the errors whose text may be shown to a caller.
- `reason.go` validates an error reason.
- `source.go` defines `Source` and its summary.
- `sourceid.go` defines `SourceID`.
- `sourcekind.go` defines `SourceKind`, with its display name and its access ceiling.
- `tag.go` defines a tag and a set of tags.
- `urls.go` validates a list of URLs.
- `*_test.go` files hold the unit tests of the types above. `accessors_test.go` covers the accessor methods.
