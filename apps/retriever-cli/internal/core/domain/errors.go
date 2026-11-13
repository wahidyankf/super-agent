package domain

import "errors"

// Validation errors. A validated value type returns one of these from its constructor.
var (
	// ErrInvalidCollectionID reports a collection ID outside ^[A-Za-z0-9_-]+$.
	ErrInvalidCollectionID = errors.New("invalid collection ID")
	// ErrInvalidDocumentID reports a document ID outside ^[A-Za-z0-9_-]+$.
	ErrInvalidDocumentID = errors.New("invalid document ID")
)

// Source catalog validation errors. Each value and each constructor returns one of these, and no text holds a value.
var (
	// ErrInvalidSourceID reports a source ID outside the slug grammar, such as "Outline" or "a--b".
	ErrInvalidSourceID = errors.New("invalid source ID")
	// ErrInvalidTag reports a tag outside the lowercase slug grammar.
	ErrInvalidTag = errors.New("invalid tag")
	// ErrInvalidAccess reports an access level other than read, write, or admin.
	ErrInvalidAccess = errors.New("must be read, write or admin")
	// ErrTooManyTags reports more than 16 distinct tags in one set.
	ErrTooManyTags = errors.New("invalid tag")
	// ErrUnknownSourceKind reports a kind name the code does not know.
	ErrUnknownSourceKind = errors.New("unknown source kind (known: " + knownKindNames() + ")")
	// ErrInvalidEndpoint reports an endpoint that is not an HTTPS host name with no userinfo, port, path, query, or fragment.
	ErrInvalidEndpoint = errors.New("invalid endpoint")
	// ErrInvalidCredentialRef reports a variable name that is not a *_API_KEY name.
	ErrInvalidCredentialRef = errors.New("invalid credential name")
	// ErrCatalogMissing reports that no sources catalog file exists at the path the flag, the environment, or the working
	// directory names. Retriever never invents a source, so a missing catalog is a refusal, not a default.
	ErrCatalogMissing = errors.New("sources catalog not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml")
	// ErrEmptyCatalog reports a catalog with no source.
	ErrEmptyCatalog = errors.New("the catalog needs at least one source")
	// ErrTooManySources reports a catalog with more than 64 sources.
	ErrTooManySources = errors.New("the catalog holds at most 64 sources")
	// ErrDuplicateSource reports two sources with the same ID.
	ErrDuplicateSource = errors.New("duplicate")
	// ErrCredentialHostMismatch reports one credential name used for two endpoint hosts.
	ErrCredentialHostMismatch = errors.New("one credential serves one host only")
	// ErrIdentityHostMismatch reports one identity name used for two endpoint hosts.
	ErrIdentityHostMismatch = errors.New("one identity serves one host only")
	// ErrIdentityRequired reports a source of a kind that needs an identity name and has none. IdentityRequiredError
	// matches it and names the kind.
	ErrIdentityRequired = errors.New("the kind needs an identity_env name")
	// ErrIdentityNotUsed reports a source of a kind that does not use an identity name but has one. IdentityNotUsedError
	// matches it and names the kind.
	ErrIdentityNotUsed = errors.New("the kind does not use an identity_env name")
	// ErrAccessAboveKindCeiling reports an access level above the highest level its kind accepts. KindCeilingError matches
	// it and names the kind and the level.
	ErrAccessAboveKindCeiling = errors.New("the access level is above the most the kind allows")
	// ErrUnknownSource reports a source ID that the catalog does not hold. UnknownSourceError matches it.
	ErrUnknownSource = errors.New("unknown source")
	// ErrNoSourceOfKind reports that no source of the requested kind is configured. NoSourceOfKindError matches it and
	// names the kind.
	ErrNoSourceOfKind = errors.New("no source of the kind is configured")
	// ErrWrongSourceKind reports a source ID that names a source of another kind.
	ErrWrongSourceKind = errors.New("the source is of another kind")
)

// Lookup errors that the use cases return.
var (
	// ErrUnsupportedSource reports a source ID that no gateway is registered for.
	ErrUnsupportedSource = errors.New("unsupported source")
	// ErrExportWriteFailed reports that the export sink could not store the document.
	ErrExportWriteFailed = errors.New("could not write the exported document")
)

// Operation argument errors. The catalog binder and the parameter kinds return these before any gateway call. Each text
// is fixed and names no value.
var (
	// ErrInvalidID reports an ID outside ^[A-Za-z0-9_-]+$.
	ErrInvalidID = errors.New("invalid ID")
	// ErrInvalidUUID reports a value that is not a canonical UUID.
	ErrInvalidUUID = errors.New("invalid UUID")

	// ErrInvalidShareID refuses a share ID that is neither a UUID nor a slug of lower-case letters, digits, and hyphens.
	ErrInvalidShareID = errors.New("invalid share ID")
	// ErrInvalidName reports a name that is empty, longer than 255 characters, or holds a control character.
	ErrInvalidName = errors.New("invalid name: 1 to 255 characters, no control characters")
	// ErrInvalidText reports a text longer than 1 MiB.
	ErrInvalidText = errors.New("text must be at most 1 MiB")
	// ErrStandardInputUnsupported refuses a text whose whole value is "-". Retriever does not read standard input, so the
	// dash would otherwise be sent as the literal text (finding B1).
	ErrStandardInputUnsupported = errors.New("standard input is not supported; pass the text as the argument")
	// ErrInvalidInt reports a number outside the range the operation states.
	ErrInvalidInt = errors.New("number is out of range")
	// ErrInvalidBool reports a flag value that is neither true nor false.
	ErrInvalidBool = errors.New("value must be true or false")
	// ErrInvalidEnum reports a value that is not one of the closed list of the operation.
	ErrInvalidEnum = errors.New("value is not one of the allowed values")
	// ErrInvalidIDList reports a list that holds more than 50 IDs, or an ID that is not valid.
	ErrInvalidIDList = errors.New("invalid ID list: at most 50 valid IDs")
	// ErrInvalidTime reports a time that is not RFC 3339 with a zone.
	ErrInvalidTime = errors.New("invalid time: use RFC 3339 with a zone")
	// ErrInvalidURL reports a URL that is not an HTTPS URL with a DNS host and no credentials.
	ErrInvalidURL = errors.New("invalid URL: HTTPS only, with a DNS host name and no credentials")
	// ErrInvalidIndex reports a fractional index that is not 1 to 256 printable ASCII characters.
	ErrInvalidIndex = errors.New("invalid index: use 1 to 256 printable ASCII characters")
	// ErrInvalidLimit reports a page limit outside 1 to MaxPageLimit.
	ErrInvalidLimit = errors.New("limit must be from 1 to 100")
	// ErrInvalidOffset reports a negative page offset.
	ErrInvalidOffset = errors.New("offset must not be negative")
	// ErrInvalidQuery reports a search query with no text in it, or one longer than 1000 characters.
	ErrInvalidQuery = errors.New("invalid query: at least one character and at most 1000 characters")
	// ErrInvalidColor reports a colour that is not a hex colour of three or six digits, such as #3a7bd5.
	ErrInvalidColor = errors.New("invalid color: use a hex color such as #3a7bd5 or #3ad")
	// ErrInvalidIcon reports an icon that is blank, longer than 255 characters, or holds a control character.
	ErrInvalidIcon = errors.New("invalid icon: use 1 to 255 characters with no control character")
	// ErrInvalidEmail reports an e-mail address that is not one name@host address of at most MaxEmailLength characters with
	// no space or control character. The address itself is never part of the text.
	ErrInvalidEmail = errors.New("invalid e-mail address: use one address such as name@example.com, with no space")
	// ErrInvalidScore reports a search score that is not a number from -1 to 1.
	ErrInvalidScore = errors.New("invalid score: use a number from -1 to 1")
	// ErrMissingArgument reports a required argument that the caller did not give.
	ErrMissingArgument = errors.New("a required argument is missing")

	// ErrExactlyOneArgument refuses a call that names none, or more than one, of the arguments that select one target.
	ErrExactlyOneArgument = errors.New("exactly one of the arguments that select the target is required")
	// ErrUnknownArgument reports an argument that the operation does not declare.
	ErrUnknownArgument = errors.New("unknown argument")
	// ErrInvalidOutputFormat reports an output format other than text or json for a read command.
	ErrInvalidOutputFormat = errors.New("output format must be text or json")
	// ErrUnsupportedOperation reports an operation that the catalog does not hold, or that the source kind does not
	// support.
	ErrUnsupportedOperation = errors.New("unsupported operation")
	// ErrAccessLevelTooLow reports that the source's access level is below the level the operation needs.
	// AccessLevelError names the source and the two levels.
	ErrAccessLevelTooLow = errors.New("the source's access level is too low for this operation")
	// ErrOperationUnavailable reports that the server does not offer the endpoint. OperationUnavailableError names it.
	ErrOperationUnavailable = errors.New("this Outline server does not offer the endpoint")
	// ErrSourceRequired refuses a write that names no source. A change always names where it goes.
	ErrSourceRequired = errors.New("this operation changes the wiki, so name its source with --source or the source argument")
	// ErrWrongOperationClass reports an operation that a port of another effect class was asked to run.
	ErrWrongOperationClass = errors.New("the operation is not of this port's effect class")
	// ErrWriteOutcomeUnknown reports a write whose request was sent but whose reply is missing or unusable: a timeout, a
	// dropped connection, a 5xx reply, a 2xx reply that is not a readable success, or a body over the cap. The change may
	// or may not have happened, so the write is never retried by Retriever.
	ErrWriteOutcomeUnknown = errors.New("the change may or may not have happened; check before you retry")
	// ErrLastRevisionRequired refuses an MCP document update that replaces the text without the revision it read
	//.
	ErrLastRevisionRequired = errors.New("last_revision is required when text replaces the document; read the revision with get_document first")
)

// Upstream errors. A source gateway returns these, and the CLI and GraphQL layers map them.
var (
	// ErrCollectionNotFound reports that the source holds no collection with the requested ID.
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrDocumentNotFound reports that the source holds no document with the requested ID.
	ErrDocumentNotFound = errors.New("document not found")
	// ErrUnresolvedName reports that a name a write takes, such as a category or a priority, matches no entry, or more
	// than one, in the source. The write sends nothing.
	ErrUnresolvedName = errors.New("a name matches no entry, or more than one, in the source")

	// ErrNotFound reports that the server holds no item for the ID of a read whose kind has no own sentinel.
	ErrNotFound = errors.New("not found")
	// ErrKeyRejected reports that the source refused the API key (HTTP 401).
	ErrKeyRejected = errors.New("Outline rejected the API key")
	// ErrAccessDenied reports that the source refused access for this key (HTTP 403).
	ErrAccessDenied = errors.New("Outline denied access")
	// ErrSourceRateLimited reports that the source kept answering HTTP 429 after the retries.
	ErrSourceRateLimited = errors.New("Outline rate limited the request")
	// ErrRetryAfterTooLong reports a Retry-After wait over 60 seconds, which is not waited out.
	ErrRetryAfterTooLong = errors.New("Outline asked to wait longer than 60 seconds")
	// ErrRedirected reports a redirect, which the client never follows.
	ErrRedirected = errors.New("Outline sent a redirect, which is not followed")
	// ErrTimedOut reports a request that got no reply in time.
	ErrTimedOut = errors.New("request to Outline timed out")
	// ErrHTTPSRequired reports a source URL that is not HTTPS.
	ErrHTTPSRequired = errors.New("HTTPS is required for Outline")
	// ErrUnreachable reports a network failure before any reply.
	ErrUnreachable = errors.New("Outline could not be reached")
	// ErrBadRequest reports an HTTP 400 reply: the server refused the request as malformed or out of range.
	ErrBadRequest = errors.New("the server refused the request")
	// ErrUpstreamFailure reports an HTTP 5xx reply from the source.
	ErrUpstreamFailure = errors.New("Outline returned an error")
	// ErrUnexpectedResponse reports a reply whose shape the client does not expect. Its text never repeats the body.
	ErrUnexpectedResponse = errors.New("unexpected response from Outline")
	// ErrPageCapReached reports that a listing hit the hard page cap of 20 pages.
	ErrPageCapReached = errors.New("Outline listing reached the page cap of 20 pages")
	// ErrReplyTooLarge reports a reply over the 10 MiB body cap. The caller can act on it: a smaller limit.
	ErrReplyTooLarge = errors.New("the reply was larger than 10 MiB; use a smaller limit")
	// ErrDocumentConflict reports a document write that the server refused because the document changed since the
	// revision the caller named.
	ErrDocumentConflict = errors.New("the document changed since the revision you read; read it again")
)
