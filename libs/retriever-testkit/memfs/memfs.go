// Package memfs is an in-memory file tree for tests. It holds regular files and folders under absolute slash paths, and
// it offers the file calls that a loader makes, so a unit test can run the real loaders without a disk. It is test
// support: no production code imports it.
package memfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
)

// Errors that the tree returns for a path that names the wrong kind of entry.
var (
	errNotFolder = errors.New("not a folder")
	errIsFolder  = errors.New("is a folder")
)

// node is one entry of the tree. A folder has no data. Its identity is the pointer, so a replaced file is a new node.
type node struct {
	dir  bool
	data []byte
}

// FS is an in-memory file tree. The zero value is not usable; build one with New.
type FS struct {
	// nodes maps each clean absolute path to its entry. The root folder is always present.
	nodes map[string]*node
}

// New returns an empty tree that holds only the root folder.
func New() *FS {
	return &FS{nodes: map[string]*node{"/": {dir: true}}}
}

// clean returns the clean form of an absolute path, and an error for a relative one.
func clean(name string) (string, error) {
	if !path.IsAbs(name) {
		return "", fmt.Errorf("memfs: the path %q is not absolute", name)
	}
	return path.Clean(name), nil
}

// pathError wraps an error for one path, in the shape the os package uses.
func pathError(op, name string, err error) error {
	return &fs.PathError{Op: op, Path: name, Err: err}
}

// MkdirAll creates the folder and each folder above it. It fails when one of those names a file.
func (f *FS) MkdirAll(name string) error {
	p, err := clean(name)
	if err != nil {
		return err
	}
	cur := "/"
	if p == "/" {
		return nil
	}
	for _, seg := range strings.Split(p[1:], "/") {
		cur = path.Join(cur, seg)
		n, ok := f.nodes[cur]
		if !ok {
			f.nodes[cur] = &node{dir: true}
			continue
		}
		if !n.dir {
			return pathError("mkdir", name, errNotFolder)
		}
	}
	return nil
}

// WriteFile stores data as the file at name, replacing a file that is already there with a new one. The folder that
// holds the file must exist, as it does for os.WriteFile.
func (f *FS) WriteFile(name string, data []byte) error {
	p, err := clean(name)
	if err != nil {
		return err
	}
	if p == "/" {
		return pathError("open", name, errIsFolder)
	}
	parent, ok := f.nodes[path.Dir(p)]
	if !ok {
		return pathError("open", name, fs.ErrNotExist)
	}
	if !parent.dir {
		return pathError("open", name, errNotFolder)
	}
	if n, ok := f.nodes[p]; ok && n.dir {
		return pathError("open", name, errIsFolder)
	}
	f.nodes[p] = &node{data: append([]byte(nil), data...)}
	return nil
}

// ReadFile returns the bytes of the file at name. A missing name returns an error that matches fs.ErrNotExist.
func (f *FS) ReadFile(name string) ([]byte, error) {
	p, err := clean(name)
	if err != nil {
		return nil, err
	}
	n, ok := f.nodes[p]
	switch {
	case !ok:
		return nil, pathError("open", name, fs.ErrNotExist)
	case n.dir:
		return nil, pathError("read", name, errIsFolder)
	}
	return append([]byte(nil), n.data...), nil
}

// IsRegular reports whether a regular file exists at name. A folder and a missing name report false.
func (f *FS) IsRegular(name string) bool {
	p, err := clean(name)
	if err != nil {
		return false
	}
	n, ok := f.nodes[p]
	return ok && !n.dir
}

// Lstat describes the entry at name, a folder or a file, without following anything. A missing name returns an error
// that matches fs.ErrNotExist. The description is the form that catalog and envfile check before they open a file.
func (f *FS) Lstat(name string) (fs.FileInfo, error) {
	p, err := clean(name)
	if err != nil {
		return nil, err
	}
	n, ok := f.nodes[p]
	if !ok {
		return nil, pathError("lstat", name, fs.ErrNotExist)
	}
	return info{name: path.Base(p), n: n}, nil
}

// Open returns a reader over the file at name. A folder, or a missing name, is an error.
func (f *FS) Open(name string) (fs.File, error) {
	p, err := clean(name)
	if err != nil {
		return nil, err
	}
	n, ok := f.nodes[p]
	switch {
	case !ok:
		return nil, pathError("open", name, fs.ErrNotExist)
	case n.dir:
		return nil, pathError("open", name, errIsFolder)
	}
	return &file{info: info{name: path.Base(p), n: n}, r: bytes.NewReader(n.data)}, nil
}

// SameFile reports whether two descriptions name one entry of this tree. A description this tree did not make never
// matches, as os.SameFile never matches a foreign description.
func (f *FS) SameFile(a, b fs.FileInfo) bool {
	ai, okA := a.(info)
	bi, okB := b.(info)
	return okA && okB && ai.n == bi.n
}

// Files returns the regular files directly inside the folder at dir, keyed by file name. A missing folder holds no
// files and is not an error. A name that is a file, not a folder, is an error.
func (f *FS) Files(dir string) (map[string][]byte, error) {
	p, err := clean(dir)
	if err != nil {
		return nil, err
	}
	n, ok := f.nodes[p]
	switch {
	case !ok:
		return map[string][]byte{}, nil
	case !n.dir:
		return nil, pathError("readdir", dir, errNotFolder)
	}
	files := map[string][]byte{}
	for key, child := range f.nodes {
		if child.dir || path.Dir(key) != p || key == p {
			continue
		}
		files[path.Base(key)] = append([]byte(nil), child.data...)
	}
	return files, nil
}

// Walk returns every regular file under the folder at root, keyed by its slash path relative to root. A missing root
// holds no files and is not an error.
func (f *FS) Walk(root string) (map[string][]byte, error) {
	p, err := clean(root)
	if err != nil {
		return nil, err
	}
	prefix := p + "/"
	if p == "/" {
		prefix = "/"
	}
	files := map[string][]byte{}
	for key, n := range f.nodes {
		if n.dir || key == p || !strings.HasPrefix(key, prefix) {
			continue
		}
		files[strings.TrimPrefix(key, prefix)] = append([]byte(nil), n.data...)
	}
	return files, nil
}

// info describes one entry. Its Sys value is the node, which gives SameFile its identity.
type info struct {
	name string
	n    *node
}

// Name returns the base name of the entry.
func (i info) Name() string { return i.name }

// Size returns the number of bytes in a file, and zero for a folder.
func (i info) Size() int64 { return int64(len(i.n.data)) }

// Mode returns a folder mode for a folder, and a regular mode for a file.
func (i info) Mode() fs.FileMode {
	if i.n.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

// ModTime returns the zero time, because the tree keeps no clock.
func (i info) ModTime() time.Time { return time.Time{} }

// IsDir reports whether the entry is a folder.
func (i info) IsDir() bool { return i.n.dir }

// Sys returns the node, the identity that SameFile compares.
func (i info) Sys() any { return i.n }

// file is an open file of the tree: a reader over a snapshot of its bytes.
type file struct {
	info info
	r    *bytes.Reader
}

// Read reads the next bytes of the file.
func (f *file) Read(p []byte) (int, error) { return f.r.Read(p) }

// Stat describes the file that was opened.
func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }

// Close releases nothing, because the tree holds no handle.
func (f *file) Close() error { return nil }
