package outlinestub_test

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// TestStartedCertificateCoversEveryCatalogHost checks that a started stub holds a certificate for the endpoint host
// of each test catalog. A gateway checks the TLS name of its source's endpoint, so each host must pass that check.
func TestStartedCertificateCoversEveryCatalogHost(t *testing.T) {
	stub := outlinestub.Start(t)
	data, err := stub.CAPEM()
	if err != nil {
		t.Fatalf("CAPEM: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("CAPEM holds no PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	for _, host := range outlinestub.CatalogHosts {
		if err := cert.VerifyHostname(host); err != nil {
			t.Errorf("the stub certificate does not cover %q: %v", host, err)
		}
	}
}
