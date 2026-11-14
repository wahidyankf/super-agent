//go:build integration

package main

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"os"
	"path/filepath"
	"testing"
)

// TestStartCertificateCoversEveryCatalogHost checks that the CA file the command writes is valid for each endpoint host
// of the test catalogs. retriever checks the TLS name of each source's endpoint, so each host must pass that check.
func TestStartCertificateCoversEveryCatalogHost(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	server, err := start(config{caFile: caFile}, system)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(server.Close)

	data, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("read the CA file: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("the CA file holds no PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	for _, host := range outlinestub.CatalogHosts {
		if err := cert.VerifyHostname(host); err != nil {
			t.Errorf("the stub CA file does not cover %q: %v", host, err)
		}
	}
}
