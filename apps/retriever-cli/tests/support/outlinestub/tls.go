package outlinestub

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// CatalogHosts are the endpoint hosts of the test catalogs. A gateway checks the TLS name of its source's endpoint,
// so the certificate of a started stub is valid for each of these names.
var CatalogHosts = []string{"docs.example.test", "wiki.engineering.example.com"}

// stubCertificate is the self-signed certificate that every TLS stub serves. It is made once per process.
var stubCertificate = sync.OnceValues(makeStubCertificate)

// makeStubCertificate returns a self-signed certificate that is valid for CatalogHosts and the loopback addresses.
// The certificate is its own trust root, so the PEM of the leaf is the CA file a client trusts.
func makeStubCertificate() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "outlinestub"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              CatalogHosts,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// NewTLSServer serves h over TLS on a loopback port with the stub certificate. The caller closes the server. It is
// the TLS server of Start, and of the engines that have no testing.TB.
func NewTLSServer(h http.Handler) (*httptest.Server, error) {
	cert, err := stubCertificate()
	if err != nil {
		return nil, err
	}
	srv := httptest.NewUnstartedServer(h)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	return srv, nil
}
