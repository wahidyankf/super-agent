package envfile

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// errFault is the fault a test makes one read, check, open, or close of the env file return.
var errFault = errors.New("injected file fault")

// fakeFileInfo is a file description with a fixed mode. It is not an os file, so os.SameFile never treats it as the same
// file as another description.
type fakeFileInfo struct {
	mode fs.FileMode
}

func (i fakeFileInfo) Name() string       { return ".env.local" }
func (i fakeFileInfo) Size() int64        { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode  { return i.mode }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fakeFileInfo) Sys() any           { return nil }

// fakeOpenFile is an open env file whose read, description, and close are set by the test.
type fakeOpenFile struct {
	r        io.Reader
	info     fs.FileInfo
	statErr  error
	closeErr error
}

func (f *fakeOpenFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *fakeOpenFile) Stat() (fs.FileInfo, error) { return f.info, f.statErr }
func (f *fakeOpenFile) Close() error               { return f.closeErr }

// failingReader is a reader that always fails, so the read of the env file can fail on demand.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errFault }

// sameFake reports whether two fake descriptions are the same file. os.SameFile matches only real descriptions, so a unit
// test compares its fake descriptions by value instead.
func sameFake(a, b fs.FileInfo) bool { return a == b }

// statReturning returns a check that describes the env file as info.
func statReturning(info fs.FileInfo) func(string) (fs.FileInfo, error) {
	return func(string) (fs.FileInfo, error) { return info, nil }
}

// openReturning returns an open that hands back one fake file.
func openReturning(f fs.File) func(string) (fs.File, error) {
	return func(string) (fs.File, error) { return f, nil }
}

// noEnvironment is a process environment with no variable set.
func noEnvironment(string) string { return "" }

// unitCredentials returns Credentials that read the given environment and the given file calls, and no real file.
func unitCredentials(getenv func(string) string, files Files) *Credentials {
	if files.SameFile == nil {
		files.SameFile = sameFake
	}
	return &Credentials{envPath: "/work/project/.env.local", getenv: getenv, files: files}
}

// mustCredentialRef returns the reference for one API key variable, failing the test when the name is not valid.
func mustCredentialRef(t *testing.T, name string) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef(name)
	if err != nil {
		t.Fatalf("NewCredentialRef(%s): %v", name, err)
	}
	return ref
}

// mustIdentityRef returns the reference for one e-mail variable, failing the test when the name is not valid.
func mustIdentityRef(t *testing.T, name string) domain.IdentityRef {
	t.Helper()
	ref, err := domain.NewIdentityRef(name)
	if err != nil {
		t.Fatalf("NewIdentityRef(%s): %v", name, err)
	}
	return ref
}

// untouchedFiles is a set of file calls that fail the test when a code path reads the file. A path that must not read
// the file uses it, so a wrong read is caught.
func untouchedFiles(t *testing.T) Files {
	t.Helper()
	return Files{
		Lstat: func(string) (fs.FileInfo, error) {
			t.Error("the env file was checked when it should not have been")
			return nil, errFault
		},
		Open: func(string) (fs.File, error) {
			t.Error("the env file was opened when it should not have been")
			return nil, errFault
		},
	}
}

// TestNewKeepsTheAbsolutePathAndTheRealCalls checks the constructor of an absolute path: it builds Credentials that read
// the real environment and the real files, and it touches neither at construction.
func TestNewKeepsTheAbsolutePathAndTheRealCalls(t *testing.T) {
	c, err := New("/work/project/.env.local")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if c.envPath != "/work/project/.env.local" || c.getenv == nil || c.files.Lstat == nil || c.files.Open == nil {
		t.Fatalf("New = %+v; want the path with the real environment and file calls", c)
	}
}

// TestAnEmptyReferenceIsRefusedBeforeAnyRead checks that the zero reference is refused before the environment or the file
// is read, so the reader never reads a variable that no reference names.
func TestAnEmptyReferenceIsRefusedBeforeAnyRead(t *testing.T) {
	neverRead := func(string) string {
		t.Error("the environment was read for an empty reference")
		return ""
	}
	c := unitCredentials(neverRead, untouchedFiles(t))
	if _, err := c.APIKey(domain.CredentialRef{}); !errors.Is(err, domain.ErrInvalidCredentialRef) {
		t.Fatalf("APIKey error = %v, want ErrInvalidCredentialRef", err)
	}
	if _, err := c.Identity(domain.IdentityRef{}); !errors.Is(err, domain.ErrInvalidIdentityRef) {
		t.Fatalf("Identity error = %v, want ErrInvalidIdentityRef", err)
	}
}

// TestTheProcessValueWinsAndTheFileIsNotRead checks the first rule: a non-empty process value is used, and the env file
// is not read at all.
func TestTheProcessValueWinsAndTheFileIsNotRead(t *testing.T) {
	env := map[string]string{"OUTLINE_API_KEY": "process-key", "JIRA_EMAIL": "owner@example.test"}
	c := unitCredentials(func(name string) string { return env[name] }, untouchedFiles(t))
	key, err := c.APIKey(mustCredentialRef(t, "OUTLINE_API_KEY"))
	if err != nil || key.Reveal() != "process-key" {
		t.Fatalf("APIKey = %q, %v; want the process value", key.Reveal(), err)
	}
	identity, err := c.Identity(mustIdentityRef(t, "JIRA_EMAIL"))
	if err != nil || identity.Reveal() != "owner@example.test" {
		t.Fatalf("Identity = %v, %v; want the process value", identity, err)
	}
}

// TestAMissingEnvFileIsAMissingKey checks that a missing env file is not an error: the key is reported as missing and
// named by its variable only.
func TestAMissingEnvFileIsAMissingKey(t *testing.T) {
	files := Files{
		Lstat: func(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist },
		Open:  func(string) (fs.File, error) { t.Error("a missing env file was opened"); return nil, errFault },
	}
	c := unitCredentials(noEnvironment, files)
	_, err := c.APIKey(mustCredentialRef(t, "OUTLINE_API_KEY"))
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != "OUTLINE_API_KEY" {
		t.Fatalf("APIKey error = %v; want a MissingKeyError naming the variable", err)
	}
}

// TestTheFileValueIsReadWhenTheProcessHasNone checks the second rule: with an empty process value, the variable is read
// from the local env file, and no other variable is taken from it.
func TestTheFileValueIsReadWhenTheProcessHasNone(t *testing.T) {
	info := fakeFileInfo{mode: 0o600}
	files := Files{
		Lstat: statReturning(info),
		Open: openReturning(&fakeOpenFile{
			r:    strings.NewReader("OTHER_API_KEY=other\nOUTLINE_API_KEY=file-key\nJIRA_EMAIL=owner@example.test\n"),
			info: info,
		}),
	}
	c := unitCredentials(noEnvironment, files)
	key, err := c.APIKey(mustCredentialRef(t, "OUTLINE_API_KEY"))
	if err != nil || key.Reveal() != "file-key" {
		t.Fatalf("APIKey = %q, %v; want the file value", key.Reveal(), err)
	}
}

// TestAFileFaultIsReturnedWithoutTheFileValue checks each step of reading the env file. A fault is returned with its
// cause or with a rule that names the step, and the file's value is never part of the error.
func TestAFileFaultIsReturnedWithoutTheFileValue(t *testing.T) {
	same := fakeFileInfo{mode: 0o600}
	tests := []struct {
		name    string
		files   Files
		wantIs  error
		wantMsg string
	}{
		{
			name: "a check that fails for a reason other than a missing file is returned",
			files: Files{
				Lstat: func(string) (fs.FileInfo, error) { return nil, errFault },
			},
			wantIs: errFault,
		},
		{
			name: "a path that is not a regular file is refused",
			files: Files{
				Lstat: statReturning(fakeFileInfo{mode: fs.ModeSymlink}),
			},
			wantMsg: "must be a regular file, not a symlink or a folder",
		},
		{
			name: "a file that cannot be opened is returned",
			files: Files{
				Lstat: statReturning(same),
				Open:  func(string) (fs.File, error) { return nil, errFault },
			},
			wantIs: errFault,
		},
		{
			name: "a file whose description fails after it opens is returned",
			files: Files{
				Lstat: statReturning(same),
				Open:  openReturning(&fakeOpenFile{r: strings.NewReader("OUTLINE_API_KEY=value\n"), info: same, statErr: errFault}),
			},
			wantIs: errFault,
		},
		{
			name: "a file that changes between the check and the open is refused",
			files: Files{
				Lstat: statReturning(same),
				Open:  openReturning(&fakeOpenFile{r: strings.NewReader("OUTLINE_API_KEY=value\n"), info: fakeFileInfo{mode: 0o644}}),
			},
			wantMsg: "changed while it was being opened",
		},
		{
			name: "a file that cannot be read is returned",
			files: Files{
				Lstat: statReturning(same),
				Open:  openReturning(&fakeOpenFile{r: failingReader{}, info: same}),
			},
			wantIs: errFault,
		},
		{
			name: "a file that fails to close after a read is returned",
			files: Files{
				Lstat: statReturning(same),
				Open: openReturning(&fakeOpenFile{
					r:        strings.NewReader("OUTLINE_API_KEY=value\n"),
					info:     same,
					closeErr: errFault,
				}),
			},
			wantIs: errFault,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := unitCredentials(noEnvironment, tt.files)
			key, err := c.APIKey(mustCredentialRef(t, "OUTLINE_API_KEY"))
			if err == nil {
				t.Fatalf("APIKey = %q with no error; want the fault", key.Reveal())
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("APIKey error = %v, want %v", err, tt.wantIs)
			}
			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("APIKey error = %v, want it to say %q", err, tt.wantMsg)
			}
			if strings.Contains(err.Error(), "value") {
				t.Fatalf("APIKey error = %v names a file value", err)
			}
		})
	}
}

// TestADotenvLineWithTextAfterAClosingQuoteNamesItsLineOnly checks that a malformed line is refused with its number, and
// the line's text never appears in the error.
func TestADotenvLineWithTextAfterAClosingQuoteNamesItsLineOnly(t *testing.T) {
	_, err := lookup(strings.NewReader("OTHER=1\nOUTLINE_API_KEY=\"secret-value\" trailing\n"), "OUTLINE_API_KEY")
	if err == nil || !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), "text after a closing quote") {
		t.Fatalf("lookup error = %v; want the line number and the rule", err)
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("lookup error = %v names the value", err)
	}
}

// TestAReadFaultInTheDotenvTextIsReturned checks that a failed read of the env text is returned, with its cause.
func TestAReadFaultInTheDotenvTextIsReturned(t *testing.T) {
	if _, err := lookup(failingReader{}, "OUTLINE_API_KEY"); !errors.Is(err, errFault) {
		t.Fatalf("lookup error = %v, want the read fault", err)
	}
}

// TestIdentityReturnsTheFileFaultToo checks that Identity reads the env file by the same rules as APIKey, so a fault in
// the file stops it before the address is checked.
func TestIdentityReturnsTheFileFaultToo(t *testing.T) {
	files := Files{Lstat: func(string) (fs.FileInfo, error) { return nil, errFault }}
	c := unitCredentials(noEnvironment, files)
	if _, err := c.Identity(mustIdentityRef(t, "JIRA_EMAIL")); !errors.Is(err, errFault) {
		t.Fatalf("Identity error = %v, want the check fault", err)
	}
}
