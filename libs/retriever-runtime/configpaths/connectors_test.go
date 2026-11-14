package configpaths

import (
	"errors"
	"testing"
)

// connectorInputs returns inputs for the connectors folder: the catalog and the env file come from flags, so only the
// connectors rule reads the environment, and the working directory is /work.
func connectorInputs(env map[string]string) Inputs {
	return Inputs{
		CatalogFlag: "/c/sources.yml",
		EnvFileFlag: "/e/.env",
		Getenv:      func(name string) string { return env[name] },
		Getwd:       func() (string, error) { return "/work", nil },
		Exists:      func(string) bool { return false },
	}
}

func TestTheConnectorsFlagIsTakenAsGivenWhenAbsolute(t *testing.T) {
	in := connectorInputs(nil)
	in.ConnectorsFlag = "/opt/specs"
	paths, err := Resolve(in)
	if err != nil || paths.Connectors != "/opt/specs" {
		t.Fatalf("Resolve() = %+v, %v; want the flag folder", paths, err)
	}
}

func TestARelativeConnectorsFlagResolvesAgainstTheWorkingDirectory(t *testing.T) {
	in := connectorInputs(nil)
	in.ConnectorsFlag = "specs"
	paths, err := Resolve(in)
	if err != nil || paths.Connectors != "/work/specs" {
		t.Fatalf("Resolve() = %+v, %v; want /work/specs", paths, err)
	}
}

func TestTheDefaultConnectorsFolderIsUnderTheXDGConfigFolder(t *testing.T) {
	paths, err := Resolve(connectorInputs(map[string]string{"XDG_CONFIG_HOME": "/cfg"}))
	if err != nil || paths.Connectors != "/cfg/retriever/connectors" {
		t.Fatalf("Resolve() = %+v, %v; want /cfg/retriever/connectors", paths, err)
	}
}

func TestTheDefaultConnectorsFolderFallsBackToHomeDotConfig(t *testing.T) {
	paths, err := Resolve(connectorInputs(map[string]string{"HOME": "/var/example"}))
	if err != nil || paths.Connectors != "/var/example/.config/retriever/connectors" {
		t.Fatalf("Resolve() = %+v, %v; want /var/example/.config/retriever/connectors", paths, err)
	}
}

// TestNoUserFolderLeavesNoDefaultConnectorsFolder checks that a missing user folder does not refuse the other paths: it
// leaves the default folder empty, and the run reads no spec.
func TestNoUserFolderLeavesNoDefaultConnectorsFolder(t *testing.T) {
	paths, err := Resolve(connectorInputs(nil))
	if err != nil || paths.Connectors != "" || paths.Catalog != "/c/sources.yml" {
		t.Fatalf("Resolve() = %+v, %v; want no default folder and the catalog kept", paths, err)
	}
}

func TestARelativeConnectorsFlagWithoutAWorkingDirectoryIsRefused(t *testing.T) {
	in := connectorInputs(nil)
	in.ConnectorsFlag = "specs"
	in.Getwd = func() (string, error) { return "", errors.New("no working directory") }
	if _, err := Resolve(in); !errors.Is(err, ErrRefused) {
		t.Fatalf("Resolve() error = %v, want ErrRefused", err)
	}
}
