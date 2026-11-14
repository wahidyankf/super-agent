package ports

import (
	"context"
	"testing"
)

// fileSystemFake and exportSinkFake satisfy the outbound ports. The test below fails to compile when a port changes shape,
// which is the only behaviour an interface-only package has.
type fileSystemFake struct{}

func (fileSystemFake) ReadFile(string) ([]byte, error) { return nil, nil }
func (fileSystemFake) Exists(string) bool              { return false }

type exportSinkFake struct{}

func (exportSinkFake) Write(_ context.Context, name string, _ []byte) (string, error) {
	return name, nil
}

// TestPortsAreSatisfiableByTheirFakes checks that the outbound ports keep the shapes the core and the adapters depend on.
func TestPortsAreSatisfiableByTheirFakes(t *testing.T) {
	var fs FileSystem = fileSystemFake{}
	var sink ExportSink = exportSinkFake{}
	p := Ports{FS: fs}
	if p.FS == nil {
		t.Fatal("Ports must hold the file system it is given")
	}
	if _, err := sink.Write(context.Background(), "name", nil); err != nil {
		t.Fatalf("a fake sink must write: %v", err)
	}
}
