package engine

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// privateV4 returns an address of the 10.0.0.0/8 range. It is built from octets, so no fixture holds a private literal.
func privateV4(last byte) netip.Addr {
	return netip.AddrFrom4([4]byte{10, 0, 0, last})
}

// lanV4 returns an address of the 192.168.0.0/16 range, built from octets.
func lanV4(last byte) netip.Addr {
	return netip.AddrFrom4([4]byte{192, 168, 0, last})
}

// publicDoc returns an address of the documentation range, which is public for this check.
func publicDoc(last byte) netip.Addr {
	return netip.AddrFrom4([4]byte{198, 51, 100, last})
}

// fakeConn is a connection that does nothing; the dialer returns it so a test can see that a dial happened.
type fakeConn struct{ net.Conn }

// recordingNetwork returns a Network whose lookup gives the addresses and whose dialer records each address dialed.
type recordingNetwork struct {
	answers []netip.Addr
	lookupE error
	dialErr error
	dialed  []string
}

// network returns the Network of the recording fake.
func (r *recordingNetwork) network() Network {
	return Network{
		Dial: func(_ context.Context, _, address string) (net.Conn, error) {
			r.dialed = append(r.dialed, address)
			if r.dialErr != nil {
				return nil, r.dialErr
			}
			return fakeConn{}, nil
		},
		Lookup: func(_ context.Context, _ string) ([]netip.Addr, error) {
			return r.answers, r.lookupE
		},
	}
}

// TestS5_AnIPLiteralHostIsRefusedBeforeAnyLookupOrDial checks that a host that is an IP address is refused at once,
// with or without the catalog setting.
func TestS5_AnIPLiteralHostIsRefusedBeforeAnyLookupOrDial(t *testing.T) {
	cases := []string{
		"127.0.0.1:443",
		"[::1]:443",
		"203.0.113.9:443",
		"169.254.169.254:443",
	}
	for _, address := range cases {
		for _, allow := range []bool{false, true} {
			rec := &recordingNetwork{lookupE: errors.New("lookup must not run"), dialErr: errors.New("dial must not run")}
			_, err := guardDial(context.Background(), "tcp", address, rec.network(), allow)
			if !errors.Is(err, ErrNetworkRefused) {
				t.Fatalf("guardDial(%q, allow=%v) = %v, want ErrNetworkRefused", address, allow, err)
			}
			if len(rec.dialed) != 0 {
				t.Fatalf("an IP literal reached the dialer: %v", rec.dialed)
			}
		}
	}
}

// TestS5_LocalNamesAreRefusedWithAndWithoutTheCatalogSetting checks that localhost and its subdomains never connect.
func TestS5_LocalNamesAreRefusedWithAndWithoutTheCatalogSetting(t *testing.T) {
	for _, address := range []string{"localhost:443", "api.localhost:443", "a.b.localhost:443"} {
		for _, allow := range []bool{false, true} {
			rec := &recordingNetwork{answers: []netip.Addr{publicDoc(7)}}
			if _, err := guardDial(context.Background(), "tcp", address, rec.network(), allow); !errors.Is(err, ErrNetworkRefused) {
				t.Fatalf("guardDial(%q, allow=%v) = %v, want ErrNetworkRefused", address, allow, err)
			}
			if len(rec.dialed) != 0 {
				t.Fatalf("a local name reached the dialer: %v", rec.dialed)
			}
		}
	}
}

// TestS5_APrivateAddressFromTheResolverIsRefusedBeforeAnyDial checks that a resolver answer in a private range fails the
// call with no dial, so the host never sees the key.
func TestS5_APrivateAddressFromTheResolverIsRefusedBeforeAnyDial(t *testing.T) {
	rec := &recordingNetwork{answers: []netip.Addr{privateV4(7)}}
	_, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false)
	if !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("err = %v, want ErrNetworkRefused", err)
	}
	if len(rec.dialed) != 0 {
		t.Fatalf("a private answer was dialed: %v", rec.dialed)
	}
}

// TestS5_AMixedAnswerIsRefusedWhenAnyAddressIsPrivate checks that one private address refuses the whole host.
func TestS5_AMixedAnswerIsRefusedWhenAnyAddressIsPrivate(t *testing.T) {
	rec := &recordingNetwork{answers: []netip.Addr{publicDoc(7), lanV4(9)}}
	if _, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false); !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("err = %v, want ErrNetworkRefused", err)
	}
	if len(rec.dialed) != 0 {
		t.Fatalf("a mixed answer was dialed: %v", rec.dialed)
	}
}

// TestS5_ThePublicAddressIsDialedAsTheCheckedIP checks that the dial goes to the address the check accepted, with the
// port of the request, and not to the host name.
func TestS5_ThePublicAddressIsDialedAsTheCheckedIP(t *testing.T) {
	rec := &recordingNetwork{answers: []netip.Addr{publicDoc(7)}}
	conn, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false)
	if err != nil || conn == nil {
		t.Fatalf("guardDial = %v, %v; want a connection", conn, err)
	}
	if len(rec.dialed) != 1 || rec.dialed[0] != "198.51.100.7:443" {
		t.Fatalf("dialed %v, want only the checked address 198.51.100.7:443", rec.dialed)
	}
}

// TestS5_TheCatalogSettingAllowsPrivateButNeverUnspecifiedOrMulticast checks that the catalog setting connects to a
// loopback address, and that an unspecified or a multicast address stays refused.
func TestS5_TheCatalogSettingAllowsPrivateButNeverUnspecifiedOrMulticast(t *testing.T) {
	loopback := netip.MustParseAddr("127.0.0.1")
	rec := &recordingNetwork{answers: []netip.Addr{loopback}}
	if _, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), true); err != nil {
		t.Fatalf("the catalog setting refused a loopback address: %v", err)
	}
	if len(rec.dialed) != 1 || rec.dialed[0] != "127.0.0.1:443" {
		t.Fatalf("dialed %v, want 127.0.0.1:443", rec.dialed)
	}
	for _, bad := range []netip.Addr{netip.AddrFrom4([4]byte{0, 0, 0, 0}), netip.MustParseAddr("224.0.0.1"), netip.MustParseAddr("ff02::1")} {
		rec := &recordingNetwork{answers: []netip.Addr{bad}}
		if _, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), true); !errors.Is(err, ErrNetworkRefused) {
			t.Fatalf("guardDial to %v with the catalog setting = %v, want ErrNetworkRefused", bad, err)
		}
		if len(rec.dialed) != 0 {
			t.Fatalf("%v was dialed", bad)
		}
	}
}

// TestS5_AddressClassesAreRefusedUnlessTheCatalogSettingAllowsThem checks each class of address the check names.
func TestS5_AddressClassesAreRefusedUnlessTheCatalogSettingAllowsThem(t *testing.T) {
	local := map[string]netip.Addr{
		"ten-slash-eight": privateV4(3),
		"one-ninety-two":  lanV4(3),
		"loopback":        netip.MustParseAddr("127.0.0.2"),
		"link-local":      netip.MustParseAddr("169.254.1.1"),
		"shared-space":    netip.AddrFrom4([4]byte{100, 64, 0, 9}),
		"unique-local-v6": netip.MustParseAddr("fd00::1"),
		"mapped-private":  netip.AddrFrom16(privateV4(3).As16()),
	}
	for name, a := range local {
		if err := checkAddress(a, false); !errors.Is(err, ErrNetworkRefused) {
			t.Fatalf("%s (%v) without the setting = %v, want ErrNetworkRefused", name, a, err)
		}
		if err := checkAddress(a, true); err != nil {
			t.Fatalf("%s (%v) with the setting = %v, want it allowed", name, a, err)
		}
	}
	never := map[string]netip.Addr{
		"unspecified-v6":         netip.IPv6Unspecified(),
		"interface-local-mcast":  netip.MustParseAddr("ff01::1"),
		"link-local-multicast-6": netip.MustParseAddr("ff02::2"),
	}
	for name, a := range never {
		if err := checkAddress(a, true); !errors.Is(err, ErrNetworkRefused) {
			t.Fatalf("%s (%v) with the setting = %v, want it refused always", name, a, err)
		}
	}
	if err := checkAddress(publicDoc(1), false); err != nil {
		t.Fatalf("a public address was refused: %v", err)
	}
	if err := checkAddress(netip.Addr{}, true); !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("an invalid address = %v, want refusal", err)
	}
}

// TestS5_ADialErrorIsReturnedAndIsNotARefusal checks that a failed dial to a checked address keeps its own error, so a
// later caller can tell an unreachable host from a refused one.
func TestS5_ADialErrorIsReturnedAndIsNotARefusal(t *testing.T) {
	boom := errors.New("connection refused by the test dialer")
	rec := &recordingNetwork{answers: []netip.Addr{publicDoc(7), publicDoc(8)}, dialErr: boom}
	_, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false)
	if !errors.Is(err, boom) || errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("err = %v, want the dial error and no refusal", err)
	}
	if len(rec.dialed) != 2 {
		t.Fatalf("dialed %v, want each checked address tried once", rec.dialed)
	}
}

// TestS5_ALookupFailureOrEmptyAnswerIsUnreachable checks the two ways a host gives no address.
func TestS5_ALookupFailureOrEmptyAnswerIsUnreachable(t *testing.T) {
	boom := errors.New("no such host in the test resolver")
	rec := &recordingNetwork{lookupE: boom}
	if _, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false); !errors.Is(err, boom) {
		t.Fatalf("a lookup failure = %v, want its own error", err)
	}
	rec = &recordingNetwork{}
	if _, err := guardDial(context.Background(), "tcp", "docs.example.test:443", rec.network(), false); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("an empty answer = %v, want ErrUnreachable", err)
	}
}

// TestS5_AnAddressWithoutAPortIsRefused checks that a malformed address never reaches the resolver.
func TestS5_AnAddressWithoutAPortIsRefused(t *testing.T) {
	rec := &recordingNetwork{answers: []netip.Addr{publicDoc(7)}}
	if _, err := guardDial(context.Background(), "tcp", "docs.example.test", rec.network(), false); !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("err = %v, want ErrNetworkRefused", err)
	}
}

// TestS5_TheTransportDialsThroughTheGuard checks that the transport the engine builds sends every dial through the
// guard, so a request can never skip it.
func TestS5_TheTransportDialsThroughTheGuard(t *testing.T) {
	rec := &recordingNetwork{answers: []netip.Addr{privateV4(3)}}
	tr := guardedTransport(rec.network(), false, nil)
	if _, err := tr.DialContext(context.Background(), "tcp", "docs.example.test:443"); !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("the transport dial = %v, want ErrNetworkRefused", err)
	}
	if len(rec.dialed) != 0 {
		t.Fatalf("the transport dialed a private address: %v", rec.dialed)
	}
	if tr.Proxy != nil {
		t.Fatal("the transport reads a proxy from the environment")
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("the transport does not require TLS 1.2 or later")
	}
}
