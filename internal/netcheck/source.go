package netcheck

import (
	"errors"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"time"
)

var errAddressFamily = errors.New("目标没有与所选源 IP 匹配的地址。")

type LocalSource struct {
	Address   string `json:"address"`
	Interface string `json:"interface"`
}

func LocalSources() ([]LocalSource, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := []LocalSource{}
	seen := map[netip.Addr]bool{}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, item := range addresses {
			prefix, err := netip.ParsePrefix(item.String())
			if err != nil {
				continue
			}
			addr := prefix.Addr().Unmap()
			if addr.IsUnspecified() || addr.IsMulticast() {
				continue
			}
			if addr.Is6() && addr.IsLinkLocalUnicast() {
				addr = addr.WithZone(strconv.Itoa(iface.Index))
			}
			if !seen[addr] {
				seen[addr] = true
				result = append(result, LocalSource{Address: addr.String(), Interface: iface.Name})
			}
		}
	}
	return result, nil
}

func ParseSource(value string) (netip.Addr, error) {
	if value == "" {
		return netip.Addr{}, nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, errors.New("请选择本机已配置的源 IP。")
	}
	addr = addr.Unmap()
	available, err := LocalSources()
	if err != nil {
		return netip.Addr{}, err
	}
	if !slices.ContainsFunc(available, func(item LocalSource) bool { return item.Address == addr.String() }) {
		return netip.Addr{}, errors.New("请选择本机已配置的源 IP。")
	}
	return addr, nil
}

func pickDestination(addresses []netip.Addr, source netip.Addr) netip.Addr {
	for _, address := range addresses {
		if !source.IsValid() || source.Is4() == address.Is4() {
			return address
		}
	}
	return netip.Addr{}
}

func sourceDialer(source netip.Addr, timeout time.Duration) net.Dialer {
	dialer := net.Dialer{Timeout: timeout}
	if source.IsValid() {
		dialer.LocalAddr = net.TCPAddrFromAddrPort(netip.AddrPortFrom(source, 0))
	}
	return dialer
}
