package netcheck

import (
	"context"
	"net"
	"net/netip"
	"os"
	"testing"
)

func TestTargetAddress(t *testing.T) {
	for _, value := range []string{"example.com", "local-host", "example.com.", "127.0.0.1", "[::1]", "fe80::1%3"} {
		if _, _, err := targetAddress(value); err != nil {
			t.Errorf("rejected %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "https://example.com", "example.com:443", "-a.com", "a..com", "::", "224.0.0.1", "fe80::1", "fe80::1%nope"} {
		if _, _, err := targetAddress(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestCheckAndTraceLoopback(t *testing.T) {
	if os.Getenv("WINTOOLBOX_NETWORK_INTEGRATION") != "1" {
		t.Skip("requires Windows ICMP and TCP loopback access")
	}
	addresses, status, err := resolve(context.Background(), "localhost")
	if err != nil || status != "resolved" || len(addresses) == 0 || !addresses[0].IsLoopback() {
		t.Fatalf("localhost resolution: %v, %s, %v", addresses, status, err)
	}
	for _, address := range []string{"127.0.0.1", "::1"} {
		t.Run(address, func(t *testing.T) {
			listener, err := net.Listen("tcp", net.JoinHostPort(address, "0"))
			if err != nil {
				t.Skipf("loopback listener unavailable: %v", err)
			}
			defer listener.Close()
			port := listener.Addr().(*net.TCPAddr).Port
			basic, err := Check(context.Background(), address, port)
			if err != nil || basic.DNS != "literal" || basic.TCP != "connected" || basic.Ping != "reply" || basic.Received != basicProbes {
				t.Fatalf("check: %+v, %v", basic, err)
			}
			route, err := Trace(context.Background(), address)
			if err != nil || !route.Reached || len(route.Hops) != 1 || route.Hops[0].Received != routeRounds ||
				len(route.Hops[0].Addresses) != 1 || route.Hops[0].Addresses[0] != address {
				t.Fatalf("trace: %+v, %v", route, err)
			}
			if status, err := CheckLocalListener(context.Background(), port); err != nil || status != "loopback-only" {
				t.Fatalf("local listener: %s, %v", status, err)
			}
			_ = listener.Close()
			if result := checkTCP(context.Background(), netip.MustParseAddr(address), port); result != "refused" {
				t.Fatalf("closed loopback port: %s", result)
			}
		})
	}
}

func TestRouteStatistics(t *testing.T) {
	route := Route{}
	for _, sample := range []echoReply{
		{address: netip.MustParseAddr("192.0.2.1"), status: ipExpired, ms: 12},
		{status: ipTimeout},
		{address: netip.MustParseAddr("192.0.2.2"), status: ipExpired, ms: 16},
	} {
		if !route.record(1, sample) {
			t.Fatalf("unexpected failure: %+v", sample)
		}
	}
	hop := route.Hops[0]
	if hop.Sent != 3 || hop.Received != 2 || hop.AverageMS != 14 || len(hop.Addresses) != 2 {
		t.Fatalf("wrong statistics: %+v", hop)
	}
}

func TestRouteDoesNotCountLocalProbeFailure(t *testing.T) {
	route := Route{}
	if route.record(1, echoReply{status: 11050}) || len(route.Hops) != 0 {
		t.Fatalf("local failure created a hop: %+v", route)
	}
	if !route.record(1, echoReply{status: ipTimeout}) {
		t.Fatal("a timed-out probe should count")
	}
	if route.record(1, echoReply{status: ipExpired}) || route.Hops[0].Sent != 1 {
		t.Fatalf("missing router address changed statistics: %+v", route)
	}
}
