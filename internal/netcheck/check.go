package netcheck

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const (
	basicProbes  = 4
	MaxRounds    = 10
	MaxTCPProbes = 10
	maxHops      = 24
)

type Basic struct {
	DNS       string   `json:"dns"`
	Addresses []string `json:"addresses"`
	Address   string   `json:"address"`
	Ping      string   `json:"ping"`
	Sent      int      `json:"sent"`
	Received  int      `json:"received"`
	AverageMS int      `json:"averageMs"`
	TCP       string   `json:"tcp"`
	Port      int      `json:"port"`
	Source    string   `json:"source"`
	TCPStats  TCPStats `json:"tcpStats"`
}

type TCPStats struct {
	Attempts  int     `json:"attempts"`
	Successes int     `json:"successes"`
	MinMS     float64 `json:"minMs"`
	AverageMS float64 `json:"averageMs"`
	MaxMS     float64 `json:"maxMs"`
}

type Hop struct {
	Number    int      `json:"number"`
	Addresses []string `json:"addresses"`
	Sent      int      `json:"sent"`
	Received  int      `json:"received"`
	AverageMS int      `json:"averageMs"`
	totalMS   int
}

type Route struct {
	Status          string `json:"status"`
	Address         string `json:"address"`
	Hops            []Hop  `json:"hops"`
	Reached         bool   `json:"reached"`
	Source          string `json:"source"`
	RoundsCompleted int    `json:"roundsCompleted"`
	Rounds          int    `json:"rounds"`
}

func targetAddress(value string) (netip.Addr, string, error) {
	host := strings.TrimSpace(value)
	literal := host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		literal = host[1 : len(host)-1]
	}
	if ip, err := netip.ParseAddr(literal); err == nil {
		ip = ip.Unmap()
		if !ip.IsUnspecified() && !ip.IsMulticast() && (ip.Zone() == "" || ip.IsLinkLocalUnicast()) &&
			(!ip.Is6() || !ip.IsLinkLocalUnicast() || ip.Zone() != "") {
			if zone := ip.Zone(); zone != "" {
				index, err := strconv.ParseUint(zone, 10, 32)
				if err != nil || index == 0 {
					return netip.Addr{}, "", errors.New("请输入有效域名或 IP 地址。")
				}
			}
			return ip, "", nil
		}
		return netip.Addr{}, "", errors.New("请输入有效域名或 IP 地址。")
	}
	name := strings.TrimSuffix(host, ".")
	if len(name) > 0 && len(name) <= 253 {
		valid := true
		for _, label := range strings.Split(name, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				valid = false
				break
			}
			for _, char := range label {
				if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-') {
					valid = false
					break
				}
			}
		}
		if valid {
			return netip.Addr{}, host, nil
		}
	}
	return netip.Addr{}, "", errors.New("请输入有效域名或 IP 地址。")
}

func resolve(ctx context.Context, target string) ([]netip.Addr, string, error) {
	ip, name, err := targetAddress(target)
	if err != nil {
		return nil, "", err
	}
	if ip.IsValid() {
		return []netip.Addr{ip}, "literal", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", name)
	if err != nil {
		var dnsErr *net.DNSError
		switch {
		case ctx.Err() != nil:
			return nil, "timeout", nil
		case errors.As(err, &dnsErr) && dnsErr.Timeout():
			return nil, "timeout", nil
		case errors.As(err, &dnsErr) && dnsErr.IsNotFound:
			return nil, "not-found", nil
		default:
			return nil, "unavailable", nil
		}
	}
	var usable []netip.Addr
	for _, address := range addresses {
		address = address.Unmap()
		if !address.IsUnspecified() && !address.IsMulticast() && !slices.Contains(usable, address) {
			usable = append(usable, address)
		}
	}
	if len(usable) == 0 {
		return nil, "not-found", nil
	}
	return usable, "resolved", nil
}

func Check(ctx context.Context, target string, port, count int, source netip.Addr) (Basic, error) {
	if port < 0 || port > 65535 {
		return Basic{}, errors.New("请输入 1 到 65535 之间的端口号。")
	}
	if count < 1 || count > MaxTCPProbes {
		return Basic{}, errors.New("TCPing 次数须为 1 到 10。")
	}
	addresses, dns, err := resolve(ctx, target)
	result := Basic{DNS: dns, Port: port, Ping: "skipped", TCP: "skipped"}
	if err != nil || len(addresses) == 0 {
		return result, err
	}
	for _, address := range addresses[:min(16, len(addresses))] {
		result.Addresses = append(result.Addresses, address.String())
	}
	address := pickDestination(addresses, source)
	if !address.IsValid() {
		return result, errAddressFamily
	}
	result.Address = address.String()
	if source.IsValid() {
		result.Source = source.String()
	}

	// TCP and Ping test the same address and neither depends on the other.
	var tcp <-chan tcpResult
	if port != 0 {
		done := make(chan tcpResult, 1)
		tcp = done
		go func() { done <- sampleTCP(ctx, address, port, count, source) }()
	}
	checkPing(ctx, address, source, &result)
	if tcp != nil {
		outcome := <-tcp
		result.TCP = outcome.status
		result.TCPStats = outcome.stats
	}
	return result, nil
}

type tcpResult struct {
	status string
	stats  TCPStats
}

func sampleTCP(ctx context.Context, address netip.Addr, port, count int, source netip.Addr) tcpResult {
	result := tcpResult{status: "unavailable"}
	dialer := sourceDialer(source, 3*time.Second)
	endpoint := net.JoinHostPort(address.String(), strconv.Itoa(port))
	var total float64
	for i := 0; i < count && ctx.Err() == nil; i++ {
		start := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		result.stats.Attempts++
		if err == nil {
			_ = conn.Close()
			result.stats.Successes++
			total += elapsed
			if result.stats.Successes == 1 {
				result.stats.MinMS = elapsed
			}
			result.stats.MinMS = min(result.stats.MinMS, elapsed)
			result.stats.MaxMS = max(result.stats.MaxMS, elapsed)
			result.stats.AverageMS = total / float64(result.stats.Successes)
			result.status = "connected"
		} else if result.stats.Successes == 0 {
			result.status = tcpFailure(err)
		}
		if i < count-1 && !waitProbe(ctx, 250*time.Millisecond) {
			break
		}
	}
	return result
}

func tcpFailure(err error) string {
	var netErr net.Error
	switch {
	case errors.Is(err, windows.WSAECONNREFUSED):
		return "refused"
	case errors.Is(err, windows.WSAENETUNREACH), errors.Is(err, windows.WSAEHOSTUNREACH):
		return "unreachable"
	case errors.As(err, &netErr) && netErr.Timeout():
		return "timeout"
	default:
		return "unavailable"
	}
}

func checkPing(ctx context.Context, address, source netip.Addr, result *Basic) {
	handle, err := openEcho(address, source)
	if err != nil {
		result.Ping = "unavailable"
		return
	}
	defer handle.close()
	result.Ping = "no-reply"
	total := 0
	for i := 0; i < basicProbes && ctx.Err() == nil; i++ {
		response := handle.probe(128, 800)
		result.Sent++
		switch response.status {
		case ipSuccess:
			result.Received++
			total += response.ms
		case ipTimeout, ipExpired, ipNetUnreachable, ipHostUnreachable, ipProtocolUnreachable, ipPortUnreachable:
			// A router error is not an echo reply from the target.
		default:
			result.Ping = "unavailable"
			return
		}
		if i < basicProbes-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}
	if result.Received > 0 {
		result.Ping = "reply"
		result.AverageMS = total / result.Received
	}
}

func Trace(ctx context.Context, target string, rounds int, source netip.Addr) (Route, error) {
	if rounds < 1 || rounds > MaxRounds {
		return Route{}, errors.New("MTR 轮数须为 1 到 10。")
	}
	addresses, dns, err := resolve(ctx, target)
	result := Route{Status: dns, Rounds: rounds}
	if ctx.Err() != nil {
		result.Status = traceStopStatus(ctx)
		return result, nil
	}
	if err != nil || len(addresses) == 0 {
		return result, err
	}
	address := pickDestination(addresses, source)
	if !address.IsValid() {
		return result, errAddressFamily
	}
	result.Address, result.Status = address.String(), "complete"
	if source.IsValid() {
		result.Source = source.String()
	}
	handle, err := openEcho(address, source)
	if err != nil {
		result.Status = "probe-unavailable"
		return result, nil
	}
	defer handle.close()

	for round := 0; round < rounds; round++ {
		for ttl := 1; ttl <= maxHops; ttl++ {
			if ctx.Err() != nil {
				result.Status = traceStopStatus(ctx)
				return result, nil
			}
			started := time.Now()
			reply := handle.probe(uint8(ttl), 450)
			if reply.status == ipExpired && reply.ms == 0 {
				reply.ms = int(time.Since(started).Milliseconds())
			}
			if !result.record(ttl, reply) {
				result.Status = "probe-unavailable"
				return result, nil
			}
			if reply.status == ipSuccess {
				result.Reached = true
				break
			}
		}
		result.RoundsCompleted++
		if ctx.Err() != nil {
			result.Status = traceStopStatus(ctx)
			return result, nil
		}
		if round < rounds-1 && !waitProbe(ctx, 200*time.Millisecond) {
			result.Status = traceStopStatus(ctx)
			return result, nil
		}
	}
	return result, nil
}

func traceStopStatus(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.Canceled) {
		return "stopped"
	}
	return "interrupted"
}

func waitProbe(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (r *Route) record(ttl int, reply echoReply) bool {
	responded := reply.status == ipSuccess || reply.status == ipExpired
	switch reply.status {
	case ipSuccess, ipExpired:
		if !reply.address.IsValid() || reply.address.IsUnspecified() {
			return false
		}
	case ipTimeout, ipNetUnreachable, ipHostUnreachable, ipProtocolUnreachable, ipPortUnreachable:
	default:
		return false
	}
	if ttl > len(r.Hops) {
		r.Hops = append(r.Hops, Hop{Number: ttl})
	}
	hop := &r.Hops[ttl-1]
	hop.Sent++
	if responded {
		hop.Received++
		hop.totalMS += reply.ms
		hop.AverageMS = hop.totalMS / hop.Received
		ip := reply.address.String()
		if !slices.Contains(hop.Addresses, ip) {
			hop.Addresses = append(hop.Addresses, ip)
		}
	}
	return true
}
