package main

import (
	"errors"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// connectorsFolderCode is the stable code of a connectors folder that the run cannot use: one that --connectors-dir names
// and that does not exist, or one that cannot be read. The text of the error names no path.
const connectorsFolderCode = "retriever.connector.folder_unavailable"

// folderError reports a connectors folder that the run cannot use. Its rule is a fixed phrase, never a path.
type folderError struct {
	rule string
}

// Error returns the fixed text of the failure.
func (e folderError) Error() string {
	return "the connectors folder " + e.rule
}

// ErrorCode returns connectorsFolderCode, so the failure keeps one code in every command that reports it.
func (folderError) ErrorCode() string {
	return connectorsFolderCode
}

// loadConnectors reads and validates the spec files of the connectors folder dir, and returns the set the run registers.
// An empty dir loads nothing. A default folder that does not exist loads nothing too. A folder that --connectors-dir named
// (named is true) must exist, so a missing one is a folderError, and so is a folder that cannot be read. A spec file that
// is a link, a folder, too large, or unreadable is reported as a spec failure of that file.
func loadConnectors(h host, dir string, named bool) (connectorspec.Set, error) {
	if dir == "" {
		return connectorspec.Load(nil)
	}
	files, err := specfolder.Read(h.openConnectors(dir), spec.MaxBytes)
	var file specfolder.FileError
	switch {
	case errors.Is(err, specfolder.ErrNoFolder) && !named:
		return connectorspec.Load(nil)
	case errors.Is(err, specfolder.ErrNoFolder):
		return connectorspec.Set{}, folderError{rule: "does not exist"}
	case errors.As(err, &file):
		return connectorspec.Set{}, &connectorspec.Error{File: file.Name, Rule: file.Rule, Path: ""}
	case err != nil:
		// The folder reader reports only a missing folder, a file that fails, and a folder that cannot be read. Every other
		// failure of listing the folder is a folder that cannot be read.
		return connectorspec.Set{}, folderError{rule: "cannot be read"}
	}
	return connectorspec.Load(files)
}

// specBuilders returns one gateway builder for each spec kind of the set. Each builder runs the generic engine over the
// validated spec of its kind, with the ports of the host and the credentials of the env file.
func specBuilders(set connectorspec.Set, seam engineSeam, creds *envfile.Credentials) map[domain.SourceKind]registry.Builder {
	builders := map[domain.SourceKind]registry.Builder{}
	for _, s := range set.Specs {
		model := s.Model
		builders[s.Kind] = func(src domain.Source) (ports.SourceGateway, error) {
			return buildSpec(model, src, seam, creds)
		}
	}
	return builders
}

// buildSpec returns the engine of one spec source. The credentials supply the key and the account address, and the seam
// supplies the clock, the network, and the options.
func buildSpec(model *spec.Spec, src domain.Source, seam engineSeam, creds *envfile.Credentials) (ports.SourceGateway, error) {
	eng, err := engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       seam.clock,
		Network:     seam.network,
		Transport:   seam.transport,
		Roots:       seam.roots,
	}, seam.options...)
	if err != nil {
		return nil, err
	}
	return eng, nil
}
