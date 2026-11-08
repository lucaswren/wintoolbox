package netcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"strings"
	"time"
)

type Egress struct {
	Family   int    `json:"family"`
	Endpoint string `json:"endpoint"`
	Source   string `json:"source"`
	PublicIP string `json:"publicIp"`
	Status   string `json:"status"`
}

func LookupEgress(ctx context.Context, source netip.Addr) []Egress {
	results := []Egress{
		{Family: 4, Endpoint: "api.ipify.org"},
		{Family: 6, Endpoint: "api6.ipify.org"},
	}
	done := make(chan Egress, len(results))
	for _, result := range results {
		go func() {
			if source.IsValid() && (result.Family == 4) != source.Is4() {
				result.Status = "skipped-family"
				done <- result
				return
			}
			done <- lookupEgress(ctx, result, source)
		}()
	}
	for range results {
		result := <-done
		if result.Family == 4 {
			results[0] = result
		} else {
			results[1] = result
		}
	}
	return results
}

func lookupEgress(ctx context.Context, result Egress, source netip.Addr) Egress {
	result.Status = "unavailable"
	dialer := sourceDialer(source, 5*time.Second)
	network := fmt.Sprintf("tcp%d", result.Family)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
		DisableKeepAlives: true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport:     transport,
		Timeout:       8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if addr, ok := info.Conn.LocalAddr().(*net.TCPAddr); ok {
				result.Source = addr.AddrPort().Addr().Unmap().String()
			}
		},
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, "https://"+result.Endpoint, nil)
	if err != nil {
		return result
	}
	response, err := client.Do(req)
	if err != nil {
		result.Status = tcpFailure(err)
		return result
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, 129))
	if err != nil || len(payload) > 128 {
		return result
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(string(payload)))
	if err != nil || ip.Zone() != "" || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.Is4() != (result.Family == 4) || ip.Is4In6() {
		return result
	}
	result.PublicIP, result.Status = ip.String(), "ok"
	return result
}
