package memfs

import (
	"io/fs"
	"path"
)

// Folder returns the in-memory folder dir as an fs.FS. Its names are relative to dir, the way os.DirFS names a file of the
// real folder, so a library that reads through an fs.FS reads the same entries as it would from disk.
func (f *FS) Folder(dir string) fs.FS {
	return folder{fsys: f, dir: dir}
}

// folder is the view of one folder of an in-memory tree.
type folder struct {
	fsys *FS
	dir  string
}

// Open opens the entry name under the folder.
func (v folder) Open(name string) (fs.File, error) {
	return v.fsys.Open(path.Join(v.dir, name))
}

// Stat describes the entry name under the folder without following a link, as the in-memory tree has none to follow.
func (v folder) Stat(name string) (fs.FileInfo, error) {
	return v.fsys.Lstat(path.Join(v.dir, name))
}
