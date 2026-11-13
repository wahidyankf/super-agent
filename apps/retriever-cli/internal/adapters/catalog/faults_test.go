package catalog

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// errFault is the fault a test makes one file step return.
var errFault = errors.New("injected file fault")

// fakeStatInfo is a file description with a fixed mode. It is not an os file, so os.SameFile never treats it as the
// same file as another description.
type fakeStatInfo struct {
	mode fs.FileMode
}

func (i fakeStatInfo) Name() string       { return "sources.yml" }
func (i fakeStatInfo) Size() int64        { return 0 }
func (i fakeStatInfo) Mode() fs.FileMode  { return i.mode }
func (i fakeStatInfo) ModTime() time.Time { return time.Time{} }
func (i fakeStatInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fakeStatInfo) Sys() any           { return nil }

// fakeOpened is an open catalog file whose reads, description, and close are set by the test.
type fakeOpened struct {
	r       io.Reader
	info    fs.FileInfo
	statErr error
}

func (f *fakeOpened) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *fakeOpened) Stat() (fs.FileInfo, error) { return f.info, f.statErr }
func (f *fakeOpened) Close() error               { return nil }

// failingReader is a reader that always fails, so the read of a catalog file can fail on demand.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errFault }

// statReturning returns a stat function that describes the path as info.
func statReturning(info fs.FileInfo) func(string) (fs.FileInfo, error) {
	return func(string) (fs.FileInfo, error) { return info, nil }
}

// openReturning returns an open function that hands back one fake file.
func openReturning(f fileReader) func(string) (fileReader, error) {
	return func(string) (fileReader, error) { return f, nil }
}

// realFileInfo returns the description of one real empty file in a test folder. os.SameFile can match only real
// descriptions, so a test that needs the same file twice takes it from here.
func realFileInfo(t *testing.T) fs.FileInfo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sources.yml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// TestALoadFaultIsReportedAsACatalogErrorNamingItsStep checks that each file step that fails returns a catalog error that
// names the step and never the file, so a path or a value never reaches the message.
func TestALoadFaultIsReportedAsACatalogErrorNamingItsStep(t *testing.T) {
	regular := fakeStatInfo{mode: 0o644}
	same := realFileInfo(t)
	tests := []struct {
		name  string
		files fileOps
		want  string
	}{
		{
			name: "a check that fails for a reason other than a missing file is reported",
			files: fileOps{
				lstat: func(string) (fs.FileInfo, error) { return nil, errFault },
			},
			want: "could not check the file",
		},
		{
			name: "a file that cannot be opened is reported",
			files: fileOps{
				lstat: statReturning(regular),
				open:  func(string) (fileReader, error) { return nil, errFault },
			},
			want: "could not open the file",
		},
		{
			name: "a file whose description fails after it opens is reported",
			files: fileOps{
				lstat: statReturning(regular),
				open:  openReturning(&fakeOpened{r: strings.NewReader("a: 1"), info: regular, statErr: errFault}),
			},
			want: "could not check the file",
		},
		{
			name: "a file that changes between the check and the open is reported",
			files: fileOps{
				lstat: statReturning(regular),
				open:  openReturning(&fakeOpened{r: strings.NewReader("a: 1"), info: fakeStatInfo{mode: 0o644}}),
			},
			want: "changed while it was being opened",
		},
		{
			name: "a file that cannot be read is reported",
			files: fileOps{
				lstat: statReturning(same),
				open:  openReturning(&fakeOpened{r: failingReader{}, info: same}),
			},
			want: "could not read the file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &Loader{path: "/work/project/sources.yml", files: tt.files}
			_, err := loader.Load()
			var catalogErr domain.CatalogError
			if !errors.As(err, &catalogErr) || catalogErr.Message != "sources catalog: "+tt.want {
				t.Fatalf("Load error = %v; want the catalog rule %q", err, tt.want)
			}
		})
	}
}

// TestAFaultInTheCatalogRuleIsNamedAsAnInvalidSource checks the fallback texts of the rule maps. A rule that no case
// names must say so, and never pass a wrong rule through.
func TestAFaultInTheCatalogRuleIsNamedAsAnInvalidSource(t *testing.T) {
	if got := fieldRule(errFault); got != "invalid" {
		t.Fatalf("fieldRule of an unknown fault = %q, want %q", got, "invalid")
	}
	if got := invariantRule(errFault); got != "invalid" {
		t.Fatalf("invariantRule of an unknown fault = %q, want %q", got, "invalid")
	}
	if got := catalogFault(errFault); got.Error() != "sources catalog: sources: invalid" {
		t.Fatalf("catalogFault of an unknown fault = %q, want the generic invalid rule", got.Error())
	}
	if got := catalogFault(domain.ErrTooManySources); got.Error() != "sources catalog: sources: "+domain.ErrTooManySources.Error() {
		t.Fatalf("catalogFault of too many sources = %q", got.Error())
	}
}
