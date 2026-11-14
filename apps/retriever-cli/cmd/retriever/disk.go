package main

import (
	"os"
	"path/filepath"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/markdownfiles"
)

// The functions in this file are the only place the real disk and the process environment reach the libraries. A
// library receives an fs.FS rooted at the folder of its file, and the file's name is the base of its path, so opening a
// loader or a reader reads nothing: the file is read only when a load or a key lookup runs.

// openEnvFile returns the credentials reader of the local env file at path, over the real files and environment.
func openEnvFile(path string) (*envfile.Credentials, error) {
	return envfile.NewWithFiles(path, os.Getenv, envfile.Files{FS: os.DirFS(filepath.Dir(path)), SameFile: os.SameFile})
}

// openCatalog returns the loader of the sources catalog at path, over the real files, for the kinds the root registered.
func openCatalog(path string, kinds domain.KindSet) (*catalog.Loader, error) {
	return catalog.NewWithFiles(path, catalog.Files{FS: os.DirFS(filepath.Dir(path)), SameFile: os.SameFile}, kinds)
}

// exportFolder returns the export sink over the real folder dir, with the calls of package os.
func exportFolder(dir string) *markdownfiles.Sink {
	return markdownfiles.New(dir, createTemp, markdownfiles.Folder{MkdirAll: os.MkdirAll, Rename: os.Rename, Remove: os.Remove})
}

// createTemp is the real temporary file creator of the export sink. A failed creation returns a typed nil in the
// interface, which the sink never uses, because it checks the error first.
func createTemp(dir, pattern string) (markdownfiles.TempFile, error) {
	return os.CreateTemp(dir, pattern)
}
