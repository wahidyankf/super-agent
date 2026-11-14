// Package specfolder reads the connector spec files of one folder through an fs.FS. The composition root supplies that
// fs.FS over the real folder, so this package names no disk path and reads only the entries it is given. It reads each
// spec file's bytes once and returns them with the SHA-256 that a catalog pin is checked against. It parses nothing: the
// spec module does that.
package specfolder

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"strings"
)

// ErrNoFolder reports that the folder does not exist. Its text carries no path, because a path may name a personal folder.
var ErrNoFolder = errors.New("the connectors folder does not exist")

// ErrFolderUnreadable reports a folder that exists and that cannot be read. Its text carries no path.
var ErrFolderUnreadable = errors.New("could not read the connectors folder")

// File is one spec file as read: its name in the folder, its bytes, and the lower-case hex SHA-256 of those bytes.
type File struct {
	// Name is the file name, without the folder.
	Name string
	// Data holds the bytes the file held when it was read.
	Data []byte
	// Digest is the lower-case hex SHA-256 of Data.
	Digest string
}

// FileError reports one spec file that cannot be read. Name is the file name, and Rule states the broken rule in fixed
// words. The text never holds the bytes of the file.
type FileError struct {
	// Name is the file name, without the folder.
	Name string
	// Rule is the broken rule, in fixed words.
	Rule string
}

// Error returns the file name and the rule, such as "connector spec file x.yml: is larger than the size limit".
func (e FileError) Error() string {
	return "connector spec file " + e.Name + ": " + e.Rule
}

// Read returns the spec files of the folder in name order. A spec file is an entry whose name ends in .yml or .yaml, and
// every other entry is skipped. A spec file that is a symbolic link, a folder, or another non-regular entry, that cannot
// be opened or read, or that holds more than maxBytes bytes fails with a FileError. A folder that does not exist returns
// ErrNoFolder, and one that cannot be read returns ErrFolderUnreadable.
func Read(folder fs.FS, maxBytes int) ([]File, error) {
	entries, err := fs.ReadDir(folder, ".")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNoFolder
		}
		return nil, ErrFolderUnreadable
	}
	var files []File
	for _, entry := range entries {
		if !isSpecName(entry.Name()) {
			continue
		}
		if !entry.Type().IsRegular() {
			return nil, FileError{Name: entry.Name(), Rule: "must be a regular file, not a symlink or a folder"}
		}
		data, err := readFile(folder, entry.Name(), maxBytes)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		files = append(files, File{Name: entry.Name(), Data: data, Digest: hex.EncodeToString(sum[:])})
	}
	return files, nil
}

// isSpecName reports whether a file name has the extension of a spec file.
func isSpecName(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}

// readFile returns the bytes of one spec file. It reads at most maxBytes plus one byte, so a larger file is detected
// without holding all of it.
func readFile(folder fs.FS, name string, maxBytes int) ([]byte, error) {
	f, err := folder.Open(name)
	if err != nil {
		return nil, FileError{Name: name, Rule: "could not be read"}
	}
	data, readErr := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	closeErr := f.Close()
	if readErr != nil || closeErr != nil {
		return nil, FileError{Name: name, Rule: "could not be read"}
	}
	if len(data) > maxBytes {
		return nil, FileError{Name: name, Rule: "is larger than the size limit"}
	}
	return data, nil
}
