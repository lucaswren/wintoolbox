package netcheck

import (
	"encoding/binary"
	"net/netip"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ipSuccess             = 0
	ipNetUnreachable      = 11002
	ipHostUnreachable     = 11003
	ipProtocolUnreachable = 11004
	ipPortUnreachable     = 11005
	ipTimeout             = 11010
	ipExpired             = 11013
)

var (
	ipHelper    = windows.NewLazySystemDLL("iphlpapi.dll")
	createEcho4 = ipHelper.NewProc("IcmpCreateFile")
	createEcho6 = ipHelper.NewProc("Icmp6CreateFile")
	closeEcho   = ipHelper.NewProc("IcmpCloseHandle")
	sendEcho4   = ipHelper.NewProc("IcmpSendEcho")
	sendEcho6   = ipHelper.NewProc("Icmp6SendEcho2")
)

type echoHandle struct {
	handle  uintptr
	address netip.Addr
}

type echoReply struct {
	address netip.Addr
	status  uint32
	ms      int
}

// The IP Helper echo APIs use the eight-byte IP_OPTION_INFORMATION32 ABI on amd64.
type echoOptions struct {
	ttl, tos, flags, optionsSize uint8
	optionsData                  uint32
}

func openEcho(address netip.Addr) (echoHandle, error) {
	create := createEcho4
	if address.Is6() {
		create = createEcho6
	}
	if err := create.Find(); err != nil {
		return echoHandle{}, err
	}
	handle, _, err := create.Call()
	if handle == uintptr(windows.InvalidHandle) {
		return echoHandle{}, err
	}
	return echoHandle{handle: handle, address: address}, nil
}

func (h echoHandle) close() { closeEcho.Call(h.handle) }

func (h echoHandle) probe(ttl uint8, timeout uint32) echoReply {
	options := echoOptions{ttl: ttl}
	payload := [16]byte{}
	buffer := [256]byte{}
	if h.address.Is4() {
		dest := h.address.As4()
		n, _, err := sendEcho4.Call(h.handle, uintptr(binary.LittleEndian.Uint32(dest[:])),
			uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), uintptr(unsafe.Pointer(&options)),
			uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(timeout))
		if n == 0 {
			return echoReply{status: uint32(err.(syscall.Errno))}
		}
		return echoReply{
			address: netip.AddrFrom4([4]byte(buffer[:4])),
			status:  binary.LittleEndian.Uint32(buffer[4:8]),
			ms:      int(binary.LittleEndian.Uint32(buffer[8:12])),
		}
	}

	scope, _ := strconv.ParseUint(h.address.Zone(), 10, 32)
	source := windows.RawSockaddrInet6{Family: windows.AF_INET6}
	dest := windows.RawSockaddrInet6{Family: windows.AF_INET6, Addr: h.address.As16(), Scope_id: uint32(scope)}
	n, _, err := sendEcho6.Call(h.handle, 0, 0, 0,
		uintptr(unsafe.Pointer(&source)), uintptr(unsafe.Pointer(&dest)),
		uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), uintptr(unsafe.Pointer(&options)),
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(timeout))
	if n == 0 {
		return echoReply{status: uint32(err.(syscall.Errno))}
	}
	// ICMPV6_ECHO_REPLY contains a packed IPV6_ADDRESS_EX at offset zero.
	// Its 26 bytes are followed by two alignment bytes before the status.
	return echoReply{
		address: netip.AddrFrom16([16]byte(buffer[6:22])),
		status:  binary.LittleEndian.Uint32(buffer[28:32]),
		ms:      int(binary.LittleEndian.Uint32(buffer[32:36])),
	}
}
