package ports

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// SourceGateway reads one external source wiki. Each source has one adapter that implements it.
// Unknown collections and documents return domain.ErrCollectionNotFound and domain.ErrDocumentNotFound.
type SourceGateway interface {
	// ListCollections returns every collection the source holds.
	ListCollections(ctx context.Context) ([]domain.Collection, error)
	// CollectionTree returns the documents of one collection as a flat list with parent IDs and depths.
	CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error)
	// Document returns one document with its Markdown body.
	Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error)
}

// SourceReader runs the read operations of the catalog against one source. The use case binds the arguments first, so a
// reader receives only validated Args for an operation of the catalog. The
// reader validates the Args again before it builds a request (operations.Validate), and an operation the reader does
// not serve returns domain.ErrUnsupportedOperation. The Record it returns holds the fields of the operation's row.
type SourceReader interface {
	Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error)
}

// SourceWriter runs the change operations of the catalog against one source, every operation of a non-read effect,
// destructive ones included. It is the write port: a read operation sent to it returns domain.ErrWrongOperationClass,
// and an operation the writer does not serve returns domain.ErrUnsupportedOperation. The use case binds the arguments
// before it calls a writer. A writer that times out after it sent a request returns domain.ErrWriteOutcomeUnknown, and
// the caller never retries that call.
type SourceWriter interface {
	Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error)
}

// Credentials supplies the API key that a credential name holds. A missing key returns a domain.MissingKeyError.
type Credentials interface {
	APIKey(ref domain.CredentialRef) (domain.ApiKey, error)
}

// Identities supplies the non-secret identity (an account e-mail address) that a source needs beside its key. A missing
// identity returns a domain.MissingKeyError, and a malformed one returns a domain.InvalidIdentityError.
type Identities interface {
	Identity(ref domain.IdentityRef) (domain.Identity, error)
}

// GatewayResolver returns the gateway that reads one configured source. It builds the gateway on first use and
// returns the same one for the same source afterwards. An unsupported kind returns domain.ErrUnsupportedSource.
// Reader returns the same source as a SourceReader, and it returns domain.ErrUnsupportedSource when the source has
// no reader. Writer does the same for a SourceWriter.
type GatewayResolver interface {
	For(source domain.Source) (SourceGateway, error)
	Reader(source domain.Source) (SourceReader, error)
	Writer(source domain.Source) (SourceWriter, error)
}

// CatalogLoader reads the sources catalog once. A problem with the catalog returns a domain.CatalogError.
type CatalogLoader interface {
	Load() (domain.Catalog, error)
}
