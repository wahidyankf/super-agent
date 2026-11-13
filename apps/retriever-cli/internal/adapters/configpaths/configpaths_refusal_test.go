package configpaths

import (
	"errors"
	"testing"
)

// TestAnUnreadableWorkingDirectoryRefusesEachRelativePath checks that a relative path needs the working directory, so a
// working directory that cannot be read refuses it, whichever file the path names.
func TestAnUnreadableWorkingDirectoryRefusesEachRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		flagCat string
		flagEnv string
	}{
		{name: "a relative catalog flag is refused", flagCat: "catalogs/work.yml"},
		{name: "a relative env-file flag is refused", flagEnv: "secrets/.env.work"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := fixture()
			in.CatalogFlag = tt.flagCat
			in.EnvFileFlag = tt.flagEnv
			in.Getwd = func() (string, error) { return "", errors.New("no directory") }
			if _, err := Resolve(in); !errors.Is(err, ErrRefused) {
				t.Fatalf("error = %v; want ErrRefused", err)
			}
		})
	}
}

// TestAnUnreadableWorkingDirectoryRefusesTheEnvFileAfterACatalogFlag checks that the catalog and the env file are resolved
// one after the other, so a refusal on the second file stops the whole resolution, even when the first file is absolute.
func TestAnUnreadableWorkingDirectoryRefusesTheEnvFileAfterACatalogFlag(t *testing.T) {
	in := fixture()
	in.CatalogFlag = "/etc/retriever/catalog.yml"
	in.Getwd = func() (string, error) { return "", errors.New("no directory") }
	got, err := Resolve(in)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error = %v; want ErrRefused", err)
	}
	if got != (Paths{}) {
		t.Fatalf("Resolve returned paths %+v with the refusal; want none", got)
	}
}
